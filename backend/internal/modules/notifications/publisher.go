package notifications

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Replayer is implemented by transports with a durable log. The
// /notifications/since endpoint type-asserts to this interface to
// support reconnect-replay; transports without a log (LocalPublisher)
// simply do not implement it and the endpoint returns 501.
type Replayer interface {
	// Replay returns events with IDs strictly after sinceID, capped at
	// limit. sinceID is opaque to the caller — typically the ID of the
	// last event the client successfully processed.
	Replay(ctx context.Context, sinceID string, limit int) ([]Event, error)
}

// Publisher is the seam between event producers (e.g. videos.Service)
// and the transport that fans events out to every replica.
//
// In single-instance / test environments, LocalPublisher writes straight
// into the in-process Hub. In production, StreamPublisher writes to a
// Redis stream and a per-replica Subscriber reads from the stream and
// calls Hub.Broadcast — so every replica's WebSocket clients see every
// event regardless of which replica produced it.
type Publisher interface {
	Publish(ctx context.Context, evt Event) error
}

// LocalPublisher fans events out through the in-process Hub. Use it in
// single-instance deployments and tests.
type LocalPublisher struct {
	hub *Hub
	log *zap.Logger
}

// NewLocalPublisher wires a publisher that broadcasts directly to hub.
func NewLocalPublisher(hub *Hub, log *zap.Logger) *LocalPublisher {
	return &LocalPublisher{hub: hub, log: log}
}

// Publish forwards evt to the local Hub. The context is accepted for
// interface symmetry with remote publishers but is not consulted —
// hub.Broadcast/SendToUser is non-blocking.
func (p *LocalPublisher) Publish(_ context.Context, evt Event) error {
	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}
	if p.log != nil {
		p.log.Info("notifications_published_local",
			zap.String("event_id", evt.ID.String()),
			zap.String("type", string(evt.Type)),
			zap.Int("clients", p.hub.Count()),
		)
	}
	if evt.RecipientID == uuid.Nil {
		p.hub.Broadcast(evt)
	} else {
		p.hub.SendToUser(evt.RecipientID, evt)
	}
	return nil
}
