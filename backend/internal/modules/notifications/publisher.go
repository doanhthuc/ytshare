package notifications

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Replayer is implemented only by transports with a durable log; transports
// without one (LocalPublisher) cause /notifications/since to return 501.
type Replayer interface {
	Replay(ctx context.Context, sinceID string, limit int) ([]Event, error)
}

// Publisher is the seam between event producers and the cross-replica transport.
type Publisher interface {
	Publish(ctx context.Context, evt Event) error
}

// LocalPublisher fans out through the in-process Hub; use for single-instance and tests.
type LocalPublisher struct {
	hub *Hub
	log *zap.Logger
}

func NewLocalPublisher(hub *Hub, log *zap.Logger) *LocalPublisher {
	return &LocalPublisher{hub: hub, log: log}
}

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
