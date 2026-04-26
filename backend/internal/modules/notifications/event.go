// Package notifications owns the WebSocket fan-out and the in-process
// pub-sub used by background jobs.
package notifications

import (
	"time"

	"github.com/google/uuid"
)

// EventType enumerates the events broadcast over the WebSocket.
type EventType string

const (
	// EventVideoShared is emitted whenever a user shares a new video.
	EventVideoShared EventType = "video_shared"
)

// Event is the wire format sent to every connected client.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// VideoSharedPayload is the payload of an EventVideoShared event.
type VideoSharedPayload struct {
	VideoID      uuid.UUID `json:"videoId"`
	YouTubeID    string    `json:"youtubeId"`
	Title        string    `json:"title"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	SharedByID   uuid.UUID `json:"sharedById"`
	SharedByName string    `json:"sharedByName"`
}
