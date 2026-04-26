package notifications

import (
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Hub fans out events to every connected client.
//
// The implementation uses a goroutine per client with buffered send channels
// so a slow consumer never blocks the publisher.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	log     *zap.Logger
}

// NewHub constructs an empty Hub.
func NewHub(log *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		log:     log,
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	h.log.Info("ws_client_registered", zap.Int("clients", h.Count()))
}

// Unregister removes a client from the hub and closes its send channel.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
	h.log.Info("ws_client_unregistered", zap.Int("clients", h.Count()))
}

// Count returns the current number of connected clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast publishes an Event to every connected client.
//
// If a client's send buffer is full we drop the message for that client
// and disconnect it — slow consumers must not block the publisher.
func (h *Hub) Broadcast(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		h.log.Error("ws_marshal", zap.Error(err))
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- raw:
		default:
			h.log.Warn("ws_slow_client_dropped", zap.String("user", c.userID.String()))
			go func(client *Client) {
				h.Unregister(client)
				_ = client.conn.Close()
			}(c)
		}
	}
}
