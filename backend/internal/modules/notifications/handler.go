package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"backend/internal/httpx"
	"backend/internal/middleware"
)

type Handler struct {
	hub      *Hub
	svc      *Service
	replayer Replayer // optional; nil disables /since
	upgrader websocket.Upgrader
	log      *zap.Logger
}

func NewHandler(hub *Hub, svc *Service, replayer Replayer, allowedOrigins []string, log *zap.Logger) *Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			_, ok := allowed[origin]
			return ok
		},
	}
	return &Handler{hub: hub, svc: svc, replayer: replayer, upgrader: upgrader, log: log}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/notifications/ws", h.handle)
	r.Get("/notifications/unread-count", h.unreadCount)
	r.Post("/notifications/mark-seen", h.markSeen)
	r.Get("/notifications/since", h.since)
}

// since returns events newer than ?id=<event-id>. Returns 501 if the
// publisher transport has no durable log.
func (h *Handler) since(w http.ResponseWriter, r *http.Request) {
	if h.replayer == nil {
		httpx.WriteError(w, httpx.NewError(http.StatusNotImplemented, "replay_unsupported", "event replay is not configured"))
		return
	}
	sinceID := r.URL.Query().Get("id")
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	events, err := h.replayer.Replay(r.Context(), sinceID, limit)
	if err != nil {
		h.log.Warn("notifications_replay", zap.Error(err))
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	n, err := h.svc.UnreadCount(r.Context(), userID)
	if err != nil {
		h.log.Warn("notifications_unread_count", zap.Error(err))
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int64{"count": n})
}

func (h *Handler) markSeen(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	at, err := h.svc.MarkSeen(r.Context(), userID)
	if err != nil {
		h.log.Warn("notifications_mark_seen", zap.Error(err))
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"seenAt": at.Format(time.RFC3339Nano)})
}

// wsReplayLimit caps backlog replay on connect to fit the send buffer.
const wsReplayLimit = 100

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	since := r.URL.Query().Get("since")

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws_upgrade", zap.Error(err))
		return
	}
	client := NewClient(h.hub, conn, userID, h.log)
	// Register synchronously so racing publishers don't broadcast before the hub knows the client.
	h.hub.Register(client)

	// Replay backlog directly before writeLoop starts — synchronous writes don't race the live pump.
	// Live events arriving during replay buffer in client.send; clients dedupe by event.id.
	if since != "" && h.replayer != nil {
		h.replayBacklog(r.Context(), conn, userID, since)
	}

	// Detach from r.Context(): cancelled when handler returns, which would tear down the upgrade.
	// Lifecycle is bounded by readLoop (peer disconnect / read deadline) and conn.Close().
	go client.Run(context.Background())
}

// replayBacklog runs before writeLoop, so it is the only writer on conn.
// Errors are logged; the client reconciles via its own dedup.
func (h *Handler) replayBacklog(ctx context.Context, conn *websocket.Conn, userID uuid.UUID, sinceID string) {
	events, err := h.replayer.Replay(ctx, sinceID, wsReplayLimit)
	if err != nil {
		h.log.Warn("ws_replay", zap.Error(err), zap.String("user", userID.String()))
		return
	}
	for _, evt := range events {
		// Drop user-targeted events for other users; broadcasts (zero RecipientID) pass through.
		if evt.RecipientID != uuid.Nil && evt.RecipientID != userID {
			continue
		}
		raw, err := json.Marshal(evt)
		if err != nil {
			continue
		}
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
			h.log.Warn("ws_replay_write", zap.Error(err))
			return
		}
	}
}
