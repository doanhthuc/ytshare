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

const DefaultStreamKey = "notifications:events"

// streamMaxLen approximate trim ceiling; tune via WithMaxLen for a larger replay window.
const streamMaxLen = 10_000

const streamReadBlock = 5 * time.Second

const replayMaxLimit = 500

// StreamPublisher provides at-least-once delivery via Redis Streams; callers must dedupe.
type StreamPublisher struct {
	client *redis.Client
	key    string
	maxLen int64
	log    *zap.Logger
}

type StreamOption func(*streamConfig)

type streamConfig struct {
	key    string
	maxLen int64
}

func WithStreamKey(k string) StreamOption {
	return func(c *streamConfig) { c.key = k }
}

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

func NewStreamPublisher(client *redis.Client, log *zap.Logger, opts ...StreamOption) *StreamPublisher {
	cfg := resolveConfig(opts)
	return &StreamPublisher{
		client: client,
		key:    cfg.key,
		maxLen: cfg.maxLen,
		log:    log,
	}
}

// Publish appends evt to the stream. Stored under a single "data" field so adding
// fields to Event doesn't require coordinated schema changes mid-rollout.
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
	id, err := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: p.key,
		MaxLen: p.maxLen,
		Approx: true,
		Values: map[string]any{"data": raw},
	}).Result()
	if err != nil {
		return fmt.Errorf("notifications: xadd: %w", err)
	}
	p.log.Info("notifications_published",
		zap.String("event_id", evt.ID.String()),
		zap.String("stream_id", id),
		zap.String("type", string(evt.Type)),
	)
	return nil
}

// Replay returns events strictly after sinceID. "" or "0" means everything.
// Exclusive semantics ("(<id>") match how clients store the last-processed ID.
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

// Subscriber pumps stream entries into the local Hub. One per replica.
// Reads independently (no consumer groups) so every replica sees every event.
type Subscriber struct {
	client *redis.Client
	hub    *Hub
	key    string
	log    *zap.Logger
}

func NewSubscriber(client *redis.Client, hub *Hub, log *zap.Logger, opts ...StreamOption) *Subscriber {
	cfg := resolveConfig(opts)
	return &Subscriber{
		client: client,
		hub:    hub,
		key:    cfg.key,
		log:    log,
	}
}

// Run pumps stream entries into hub until ctx is cancelled. Starts from "$"
// (only new events) so reconnected clients aren't double-toasted from older entries.
func (s *Subscriber) Run(ctx context.Context) error {
	lastID := "$"
	for {
		streams, err := s.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{s.key, lastID},
			Block:   streamReadBlock,
			Count:   100,
		}).Result()
		switch {
		case err == nil:
		case errors.Is(err, redis.Nil):
			continue
		case ctx.Err() != nil:
			return nil //nolint:nilerr // ctx cancellation is a clean shutdown signal
		default:
			s.log.Warn("notifications_xread", zap.Error(err))
			// Back off so a Redis outage doesn't hot-loop.
			select {
			case <-ctx.Done():
				return nil //nolint:nilerr // ctx cancellation is a clean shutdown signal
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
	s.log.Info("notifications_received",
		zap.String("event_id", evt.ID.String()),
		zap.String("stream_id", msg.ID),
		zap.String("type", string(evt.Type)),
		zap.Int("clients", s.hub.Count()),
	)
	if evt.RecipientID == uuid.Nil {
		s.hub.Broadcast(evt)
	} else {
		s.hub.SendToUser(evt.RecipientID, evt)
	}
}
