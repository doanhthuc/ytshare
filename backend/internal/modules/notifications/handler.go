package notifications

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"backend/internal/httpx"
	"backend/internal/middleware"
)

// Handler exposes the WebSocket endpoint.
type Handler struct {
	hub      *Hub
	svc      *Service
	upgrader websocket.Upgrader
	log      *zap.Logger
}

// NewHandler constructs the WebSocket handler.
func NewHandler(hub *Hub, svc *Service, allowedOrigins []string, log *zap.Logger) *Handler {
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
	return &Handler{hub: hub, svc: svc, upgrader: upgrader, log: log}
}

// RegisterRoutes mounts the WebSocket route. The route is wrapped in
// the auth middleware so only authenticated users can subscribe.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/notifications/ws", h.handle)
	r.Get("/notifications/unread-count", h.unreadCount)
	r.Post("/notifications/mark-seen", h.markSeen)
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

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws_upgrade", zap.Error(err))
		return
	}
	client := NewClient(h.hub, conn, userID, h.log)
	go client.Run(r.Context())
}
