// Package videos owns sharing and listing of YouTube videos.
package videos

import (
	"time"

	"github.com/google/uuid"

	"backend/internal/modules/users"
)

type Video struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"          json:"id"`
	YouTubeID    string     `gorm:"column:youtube_id;size:32;index;not null"  json:"youtubeId"`
	URL          string     `gorm:"size:512;not null"             json:"url"`
	Title        string     `gorm:"size:255;not null"             json:"title"`
	Description  string     `gorm:"type:text"                     json:"description"`
	ThumbnailURL string     `gorm:"size:512"                      json:"thumbnailUrl"`
	SharedByID   uuid.UUID  `gorm:"type:uuid;index;not null"      json:"sharedById"`
	SharedBy     users.User `gorm:"foreignKey:SharedByID"         json:"sharedBy"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (Video) TableName() string { return "videos" }
