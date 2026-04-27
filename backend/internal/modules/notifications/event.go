// Package notifications owns the WebSocket fan-out and pub-sub for background jobs.
package notifications

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventVideoShared EventType = "video_shared"
)

// Event is the wire format. ID enables dedup/replay across at-least-once transports.
// RecipientID == uuid.Nil means broadcast; non-zero routes only to that user's connections.
type Event struct {
	ID          uuid.UUID `json:"id"`
	Type        EventType `json:"type"`
	Timestamp   time.Time `json:"timestamp"`
	RecipientID uuid.UUID `json:"recipientId,omitempty"`
	Payload     any       `json:"payload"`
}

type VideoSharedPayload struct {
	VideoID      uuid.UUID `json:"videoId"`
	YouTubeID    string    `json:"youtubeId"`
	Title        string    `json:"title"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	SharedByID   uuid.UUID `json:"sharedById"`
	SharedByName string    `json:"sharedByName"`
}
