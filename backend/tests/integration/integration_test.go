// Package integration drives the HTTP server end-to-end against an
// in-memory SQLite database and the in-memory cache. It exercises the
// sign-up → share → WebSocket-notify happy path that the spec calls out.
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/jobs"
	"backend/internal/modules/auth"
	"backend/internal/server"
)

func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrateForTests(db))

	log := zap.NewNop()
	worker := jobs.NewWorker(2, 16, log)

	cfg := config.Config{
		HTTP: config.HTTPConfig{Port: 0},
		JWT: config.JWTConfig{
			AccessSecret:  "test-access",
			RefreshSecret: "test-refresh",
			AccessTTL:     5 * time.Minute,
			RefreshTTL:    24 * time.Hour,
		},
		CORS: config.CORSConfig{AllowedOrigins: []string{"*"}},
	}

	srv := server.Build(context.Background(), server.Deps{
		Config: cfg,
		Logger: log,
		DB:     db,
		Cache:  cache.NewMemoryCache(),
		Worker: worker,
	})
	ts := httptest.NewServer(srv.Handler)

	cleanup := func() {
		ts.Close()
		worker.Stop()
	}
	return ts, cleanup
}

func postJSON(t *testing.T, url, token string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(raw))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestEndToEnd_SignUpShareNotify(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	// 1) sign up the publisher
	resp := postJSON(t, ts.URL+"/api/v1/auth/signup", "", map[string]string{
		"email":    "pub@example.com",
		"name":     "Publisher",
		"password": "password123",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var publisher auth.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&publisher))
	resp.Body.Close()

	// 2) sign up the subscriber
	resp = postJSON(t, ts.URL+"/api/v1/auth/signup", "", map[string]string{
		"email":    "sub@example.com",
		"name":     "Subscriber",
		"password": "password123",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var subscriber auth.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&subscriber))
	resp.Body.Close()

	// 3) subscriber opens a WebSocket
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/v1/notifications/ws"
	u, err := url.Parse(wsURL)
	require.NoError(t, err)
	q := u.Query()
	q.Set("access_token", subscriber.AccessToken)
	u.RawQuery = q.Encode()

	conn, dialResp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	if dialResp != nil {
		_ = dialResp.Body.Close()
	}
	defer conn.Close()

	// Dial returns when the HTTP 101 lands on the client side, but the
	// server-side hub registration follows the response. Give it a moment
	// so the broadcast below is not delivered before the subscriber is on
	// the hub's roster.
	time.Sleep(100 * time.Millisecond)

	// 4) publisher shares a video
	resp = postJSON(t, ts.URL+"/api/v1/videos", publisher.AccessToken, map[string]string{
		"url":   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		"title": "Test video",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	// 5) subscriber receives the broadcast
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var event struct {
		Type    string `json:"type"`
		Payload struct {
			Title        string `json:"title"`
			SharedByName string `json:"sharedByName"`
		} `json:"payload"`
	}
	require.NoError(t, json.Unmarshal(msg, &event))
	assert.Equal(t, "video_shared", event.Type)
	assert.Equal(t, "Test video", event.Payload.Title)
	assert.Equal(t, "Publisher", event.Payload.SharedByName)
}

func TestSignIn_BadPassword(t *testing.T) {
	ts, cleanup := newTestServer(t)
	defer cleanup()

	resp := postJSON(t, ts.URL+"/api/v1/auth/signup", "", map[string]string{
		"email":    "x@example.com",
		"name":     "X",
		"password": "password123",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	resp = postJSON(t, ts.URL+"/api/v1/auth/signin", "", map[string]string{
		"email":    "x@example.com",
		"password": "wrong-password",
	})
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
