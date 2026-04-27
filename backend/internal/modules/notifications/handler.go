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

// Handler exposes the WebSocket endpoint.
type Handler struct {
	hub      *Hub
	svc      *Service
	replayer Replayer // optional; nil disables /since
	upgrader websocket.Upgrader
	log      *zap.Logger
}

// NewHandler constructs the WebSocket handler. replayer is optional —
// pass nil for transports without a durable log.
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
				return true // same-origin / non-browser clients
			}
			_, ok := allowed[origin]
			return ok
		},
	}
	return &Handler{hub: hub, svc: svc, replayer: replayer, upgrader: upgrader, log: log}
}

// RegisterRoutes mounts the WebSocket route. The route is wrapped in
// the auth middleware so only authenticated users can subscribe.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/notifications/ws", h.handle)
	r.Get("/notifications/unread-count", h.unreadCount)
	r.Post("/notifications/mark-seen", h.markSeen)
	r.Get("/notifications/since", h.since)
}

// since returns events newer than ?id=<event-id>. The client persists
// the last event ID it processed and calls this on WebSocket reconnect
// so a brief network blip does not silently drop a notification.
//
// Returns 501 if the publisher transport has no durable log.
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

// wsReplayLimit caps how many backlog events we replay on connect.
// Sized so the burst stays under the client's send buffer headroom and
// the upgrade handshake doesn't stall on a huge backlog.
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
	// Register synchronously before returning so a publisher racing the
	// upgrade cannot broadcast into a hub that does not yet know about
	// this client.
	h.hub.Register(client)

	// If the client supplied ?since=<event-id>, push any backlog events
	// directly to the connection before writeLoop starts. writeLoop is
	// not running yet, so these synchronous writes don't race the live
	// pump. Live events that arrive while we're replaying are buffered
	// in client.send and pumped immediately after Run starts — clients
	// dedupe by event.id to handle the small overlap window.
	if since != "" && h.replayer != nil {
		h.replayBacklog(r.Context(), conn, userID, since)
	}

	// Detach from r.Context(): the request context is cancelled the moment
	// this handler returns, which would tear down the upgraded connection
	// immediately. The client lifecycle is bounded instead by the read
	// loop (peer disconnect / read deadline) and by conn.Close().
	go client.Run(context.Background())
}

// replayBacklog writes events newer than sinceID directly to conn. It
// runs before writeLoop, so it is the only writer on the connection.
// On error we just log and continue — the client will reconcile via
// its own dedup once live events arrive.
func (h *Handler) replayBacklog(ctx context.Context, conn *websocket.Conn, userID uuid.UUID, sinceID string) {
	events, err := h.replayer.Replay(ctx, sinceID, wsReplayLimit)
	if err != nil {
		h.log.Warn("ws_replay", zap.Error(err), zap.String("user", userID.String()))
		return
	}
	for _, evt := range events {
		// Drop events targeted at a different user. (Broadcast events
		// have RecipientID == zero, so they pass through.)
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
