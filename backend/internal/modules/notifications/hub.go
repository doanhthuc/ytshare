package notifications

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// broadcastBuffer caps the number of in-flight events the publisher may
// queue before the hub has a chance to fan them out. Sized so a brief
// burst of shares does not block request handlers, but a sustained
// backlog surfaces as a dropped-broadcast warning rather than silently
// growing memory.
const broadcastBuffer = 256

// Hub fans out events to every connected client.
//
// All mutations of the clients set happen inside a single owner
// goroutine (run). Register, Unregister, Broadcast and SendToUser post
// messages to that goroutine over channels, so there is no shared
// mutable state, no mutex, and no possibility of sending to a closed
// channel.
//
// The owner indexes clients by userID so that SendToUser is O(k) in the
// number of connections that user has open (multi-tab, multi-device),
// not O(N) in the total fleet.
type Hub struct {
	register   chan registerReq
	unregister chan *Client
	dispatch   chan dispatchMsg
	stop       chan struct{}
	done       chan struct{}
	closeOnce  sync.Once
	count      atomic.Int64
	log        *zap.Logger
}

type registerReq struct {
	client *Client
	ack    chan struct{}
}

// dispatchMsg is what the owner goroutine receives. recipient == zero
// means broadcast; otherwise route only to that user's connections.
type dispatchMsg struct {
	recipient uuid.UUID
	payload   []byte
}

// NewHub constructs a Hub and starts its owner goroutine.
func NewHub(log *zap.Logger) *Hub {
	h := &Hub{
		register:   make(chan registerReq),
		unregister: make(chan *Client, 16),
		dispatch:   make(chan dispatchMsg, broadcastBuffer),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		log:        log,
	}
	go h.run()
	return h
}

// Close stops the hub and closes every client's send channel. It blocks
// until the owner goroutine has drained. Safe to call multiple times.
func (h *Hub) Close() {
	h.closeOnce.Do(func() { close(h.stop) })
	<-h.done
}

// Register adds a client and blocks until the owner has acknowledged it.
// The synchronous handshake means a publisher racing the WebSocket
// upgrade cannot broadcast into a hub that does not yet know about this
// client.
func (h *Hub) Register(c *Client) {
	ack := make(chan struct{})
	select {
	case h.register <- registerReq{client: c, ack: ack}:
		<-ack
	case <-h.stop:
		close(c.send)
	}
}

// Unregister asks the hub to drop a client. It is idempotent and never
// blocks the caller; the owner deduplicates concurrent requests.
func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.stop:
	}
}

// Count returns the current number of connected clients.
func (h *Hub) Count() int {
	return int(h.count.Load())
}

// Broadcast publishes evt to every connected client. The publisher
// never blocks: if the hub's queue is full the event is dropped.
func (h *Hub) Broadcast(evt Event) {
	h.publish(evt, uuid.Nil)
}

// SendToUser publishes evt only to the connections owned by userID.
// Used for personalized notifications. If the user has no live
// connections on this replica the call is a silent no-op (the event is
// not buffered for later — that is the responsibility of the durable
// stream + the /notifications/since endpoint).
func (h *Hub) SendToUser(userID uuid.UUID, evt Event) {
	if userID == uuid.Nil {
		// Treat as broadcast rather than silently dropping; nil
		// recipient is almost certainly a bug at the call site.
		h.log.Warn("ws_send_to_user_nil_id")
		h.Broadcast(evt)
		return
	}
	h.publish(evt, userID)
}

func (h *Hub) publish(evt Event, recipient uuid.UUID) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		h.log.Error("ws_marshal", zap.Error(err))
		return
	}
	select {
	case h.dispatch <- dispatchMsg{recipient: recipient, payload: raw}:
	case <-h.stop:
	default:
		h.log.Warn("ws_dispatch_queue_full",
			zap.String("type", string(evt.Type)),
			zap.String("recipient", recipient.String()))
	}
}

// run owns the clients map. It is the only goroutine that reads or
// writes the map or closes any client's send channel.
func (h *Hub) run() {
	byUser := make(map[uuid.UUID]map[*Client]struct{})
	total := 0
	defer close(h.done)
	defer func() {
		for _, set := range byUser {
			for c := range set {
				close(c.send)
			}
		}
		h.count.Store(0)
	}()

	addClient := func(c *Client) {
		set, ok := byUser[c.userID]
		if !ok {
			set = make(map[*Client]struct{})
			byUser[c.userID] = set
		}
		if _, exists := set[c]; exists {
			return
		}
		set[c] = struct{}{}
		total++
		h.count.Store(int64(total))
	}

	dropClient := func(c *Client, reason string) {
		set, ok := byUser[c.userID]
		if !ok {
			return
		}
		if _, exists := set[c]; !exists {
			return
		}
		delete(set, c)
		if len(set) == 0 {
			delete(byUser, c.userID)
		}
		total--
		close(c.send)
		h.count.Store(int64(total))
		h.log.Info(reason, zap.Int("clients", total))
	}

	deliver := func(c *Client, msg []byte) bool {
		select {
		case c.send <- msg:
			return true
		default:
			h.log.Warn("ws_slow_client_dropped", zap.String("user", c.userID.String()))
			return false
		}
	}

	for {
		select {
		case <-h.stop:
			return

		case req := <-h.register:
			addClient(req.client)
			close(req.ack)
			h.log.Info("ws_client_registered", zap.Int("clients", total))

		case c := <-h.unregister:
			dropClient(c, "ws_client_unregistered")

		case msg := <-h.dispatch:
			if msg.recipient == uuid.Nil {
				// Broadcast — fan out to every client.
				for _, set := range byUser {
					for c := range set {
						if !deliver(c, msg.payload) {
							dropClient(c, "ws_client_unregistered")
						}
					}
				}
			} else {
				// Targeted — only this user's connections.
				set, ok := byUser[msg.recipient]
				if !ok {
					continue
				}
				for c := range set {
					if !deliver(c, msg.payload) {
						dropClient(c, "ws_client_unregistered")
					}
				}
			}
		}
	}
}
