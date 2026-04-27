package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// DefaultStreamKey is the Redis stream every replica reads from. Every
// event the system fans out is appended here.
const DefaultStreamKey = "notifications:events"

// streamMaxLen caps the stream length. Old entries are discarded with
// approximate trimming (`MAXLEN ~`) so an idle stream doesn't grow
// unbounded. Tune via WithMaxLen if you need a larger replay window.
const streamMaxLen = 10_000

// streamReadBlock is how long XREAD waits for new entries before
// returning empty. Short enough that shutdown is responsive, long
// enough to avoid hot-looping against Redis.
const streamReadBlock = 5 * time.Second

// replayMaxLimit caps the /since response so a misbehaving client cannot
// XRANGE the entire stream into one HTTP body.
const replayMaxLimit = 500

// StreamPublisher publishes events to a Redis stream so that every
// replica subscribed to the same stream sees them.
//
// Redis Streams give us:
//   - durability across replica restarts (subject to retention),
//   - ordered delivery,
//   - replay-from-id for clients that reconnect after a network blip.
//
// At-least-once semantics: callers should treat duplicate IDs as a
// possibility (the Subscriber + clients are responsible for dedup).
type StreamPublisher struct {
	client *redis.Client
	key    string
	maxLen int64
	log    *zap.Logger
}

// StreamOption configures a StreamPublisher or Subscriber.
type StreamOption func(*streamConfig)

type streamConfig struct {
	key    string
	maxLen int64
}

// WithStreamKey overrides the default stream key.
func WithStreamKey(k string) StreamOption {
	return func(c *streamConfig) { c.key = k }
}

// WithMaxLen overrides the approximate max length for trimming.
func WithMaxLen(n int64) StreamOption {
	return func(c *streamConfig) { c.maxLen = n }
}

func resolveConfig(opts []StreamOption) streamConfig {
	c := streamConfig{key: DefaultStreamKey, maxLen: streamMaxLen}
	for _, o := range opts {
		o(&c)
	}
	return c
}

// NewStreamPublisher constructs a publisher that XADDs to a Redis stream.
func NewStreamPublisher(client *redis.Client, log *zap.Logger, opts ...StreamOption) *StreamPublisher {
	cfg := resolveConfig(opts)
	return &StreamPublisher{
		client: client,
		key:    cfg.key,
		maxLen: cfg.maxLen,
		log:    log,
	}
}

// Publish marshals evt and appends it to the stream.
//
// The event payload is stored under a single "data" field rather than
// flattened so that adding fields to Event doesn't require coordinating
// schema changes across replicas mid-rollout.
func (p *StreamPublisher) Publish(ctx context.Context, evt Event) error {
	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	raw, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("notifications: marshal event: %w", err)
	}
	if err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.key,
		MaxLen: p.maxLen,
		Approx: true,
		Values: map[string]any{"data": raw},
	}).Err(); err != nil {
		return fmt.Errorf("notifications: xadd: %w", err)
	}
	return nil
}

// Replay returns events with stream IDs strictly after sinceID, in
// chronological order. Used by the /notifications/since endpoint so a
// reconnecting client can recover events delivered while it was offline.
//
// sinceID accepts any Redis stream ID ("0" for "everything", a real
// "<ms>-<seq>" ID, or "" which is treated as "0"). The exclusive
// semantics ("(<id>") match how clients store the last processed ID
// without needing to pre-increment it.
func (p *StreamPublisher) Replay(ctx context.Context, sinceID string, limit int) ([]Event, error) {
	if limit <= 0 || limit > replayMaxLimit {
		limit = replayMaxLimit
	}
	if sinceID == "" {
		sinceID = "0"
	}
	msgs, err := p.client.XRangeN(ctx, p.key, "("+sinceID, "+", int64(limit)).Result()
	if err != nil {
		return nil, fmt.Errorf("notifications: xrange: %w", err)
	}
	out := make([]Event, 0, len(msgs))
	for _, m := range msgs {
		raw, ok := m.Values["data"].(string)
		if !ok {
			p.log.Warn("notifications_stream_bad_entry", zap.String("id", m.ID))
			continue
		}
		var evt Event
		if err := json.Unmarshal([]byte(raw), &evt); err != nil {
			p.log.Warn("notifications_stream_unmarshal", zap.Error(err), zap.String("id", m.ID))
			continue
		}
		out = append(out, evt)
	}
	return out, nil
}

// Subscriber reads events from the Redis stream and delivers them to
// the local Hub. Run one per replica.
//
// Each replica reads independently from its own cursor, so every
// replica sees every event (broadcast fan-out, not work-queue
// semantics — we deliberately do NOT use consumer groups).
type Subscriber struct {
	client *redis.Client
	hub    *Hub
	key    string
	log    *zap.Logger
}

// NewSubscriber wires a subscriber that pumps stream entries into hub.
func NewSubscriber(client *redis.Client, hub *Hub, log *zap.Logger, opts ...StreamOption) *Subscriber {
	cfg := resolveConfig(opts)
	return &Subscriber{
		client: client,
		hub:    hub,
		key:    cfg.key,
		log:    log,
	}
}

// Run blocks until ctx is cancelled, pumping stream entries into the
// hub. On startup it begins from "$" (only events newer than the
// replica's start), which is the right behaviour for fan-out: clients
// connected to this replica reconnect elsewhere during downtime, so
// replaying older events would just toast users twice.
func (s *Subscriber) Run(ctx context.Context) error {
	lastID := "$"
	for {
		if ctx.Err() != nil {
			return nil
		}
		streams, err := s.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{s.key, lastID},
			Block:   streamReadBlock,
			Count:   100,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.Canceled) {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			s.log.Warn("notifications_xread", zap.Error(err))
			// Back off briefly so a Redis outage does not hot-loop.
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			continue
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				lastID = msg.ID
				s.dispatch(msg)
			}
		}
	}
}

func (s *Subscriber) dispatch(msg redis.XMessage) {
	raw, ok := msg.Values["data"].(string)
	if !ok {
		s.log.Warn("notifications_stream_bad_entry", zap.String("id", msg.ID))
		return
	}
	var evt Event
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		s.log.Warn("notifications_stream_unmarshal", zap.Error(err), zap.String("id", msg.ID))
		return
	}
	if evt.RecipientID == uuid.Nil {
		s.hub.Broadcast(evt)
	} else {
		s.hub.SendToUser(evt.RecipientID, evt)
	}
}
