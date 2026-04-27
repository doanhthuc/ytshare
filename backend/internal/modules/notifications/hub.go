package notifications

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// broadcastBuffer caps in-flight events; sustained overflow surfaces as a dropped-broadcast warning.
const broadcastBuffer = 256

// Hub fans out events. All client-set mutations happen in the owner goroutine (run);
// callers post messages over channels — no mutex, no send-on-closed-channel.
// Indexed by userID so SendToUser is O(k) in that user's connections, not O(N).
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

// recipient == uuid.Nil means broadcast; otherwise route only to that user.
type dispatchMsg struct {
	recipient uuid.UUID
	payload   []byte
}

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

// Close stops the hub and closes every client's send channel. Idempotent; blocks until drained.
func (h *Hub) Close() {
	h.closeOnce.Do(func() { close(h.stop) })
	<-h.done
}

// Register blocks until the owner acks, so racing publishers can't broadcast before registration.
func (h *Hub) Register(c *Client) {
	ack := make(chan struct{})
	select {
	case h.register <- registerReq{client: c, ack: ack}:
		<-ack
	case <-h.stop:
		close(c.send)
	}
}

// Unregister is idempotent and non-blocking; the owner dedupes concurrent requests.
func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.stop:
	}
}

func (h *Hub) Count() int {
	return int(h.count.Load())
}

// Broadcast is non-blocking; events are dropped when the dispatch queue is full.
func (h *Hub) Broadcast(evt Event) {
	h.publish(evt, uuid.Nil)
}

// SendToUser routes only to userID's connections on this replica; no buffering for offline users.
func (h *Hub) SendToUser(userID uuid.UUID, evt Event) {
	if userID == uuid.Nil {
		// Almost certainly a caller bug; warn and broadcast rather than silently drop.
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

// clientRegistry: not safe for concurrent use; called only from the run goroutine.
type clientRegistry struct {
	byUser map[uuid.UUID]map[*Client]struct{}
	total  int
	count  *atomic.Int64
	log    *zap.Logger
}

func newClientRegistry(count *atomic.Int64, log *zap.Logger) *clientRegistry {
	return &clientRegistry{
		byUser: make(map[uuid.UUID]map[*Client]struct{}),
		count:  count,
		log:    log,
	}
}

func (r *clientRegistry) add(c *Client) {
	set, ok := r.byUser[c.userID]
	if !ok {
		set = make(map[*Client]struct{})
		r.byUser[c.userID] = set
	}
	if _, exists := set[c]; exists {
		return
	}
	set[c] = struct{}{}
	r.total++
	r.count.Store(int64(r.total))
}

func (r *clientRegistry) drop(c *Client, reason string) {
	set, ok := r.byUser[c.userID]
	if !ok {
		return
	}
	if _, exists := set[c]; !exists {
		return
	}
	delete(set, c)
	if len(set) == 0 {
		delete(r.byUser, c.userID)
	}
	r.total--
	close(c.send)
	r.count.Store(int64(r.total))
	r.log.Info(reason, zap.Int("clients", r.total))
}

func (r *clientRegistry) deliver(c *Client, payload []byte) bool {
	select {
	case c.send <- payload:
		return true
	default:
		r.log.Warn("ws_slow_client_dropped", zap.String("user", c.userID.String()))
		return false
	}
}

func (r *clientRegistry) deliverTo(set map[*Client]struct{}, payload []byte) {
	for c := range set {
		if !r.deliver(c, payload) {
			r.drop(c, "ws_client_unregistered")
		}
	}
}

func (r *clientRegistry) dispatch(msg dispatchMsg) {
	if msg.recipient == uuid.Nil {
		for _, set := range r.byUser {
			r.deliverTo(set, msg.payload)
		}
		return
	}
	if set, ok := r.byUser[msg.recipient]; ok {
		r.deliverTo(set, msg.payload)
	}
}

func (r *clientRegistry) closeAll() {
	for _, set := range r.byUser {
		for c := range set {
			close(c.send)
		}
	}
	r.count.Store(0)
}

// run is the sole reader/writer of the clients map and closer of send channels.
func (h *Hub) run() {
	reg := newClientRegistry(&h.count, h.log)
	defer close(h.done)
	defer reg.closeAll()

	for {
		select {
		case <-h.stop:
			return
		case req := <-h.register:
			reg.add(req.client)
			close(req.ack)
			h.log.Info("ws_client_registered", zap.Int("clients", reg.total))
		case c := <-h.unregister:
			reg.drop(c, "ws_client_unregistered")
		case msg := <-h.dispatch:
			reg.dispatch(msg)
		}
	}
}
