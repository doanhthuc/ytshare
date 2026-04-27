package videos

import (
	"time"

	"github.com/google/uuid"
)

type ShareRequest struct {
	URL         string `json:"url"         validate:"required,url"`
	Title       string `json:"title"       validate:"omitempty,max=255"`
	Description string `json:"description" validate:"omitempty,max=4096"`
}

type VideoView struct {
	ID           uuid.UUID `json:"id"`
	YouTubeID    string    `json:"youtubeId"`
	URL          string    `json:"url"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ThumbnailURL string    `json:"thumbnailUrl"`
	SharedAt     time.Time `json:"sharedAt"`
	SharedBy     SharedBy  `json:"sharedBy"`
}

type SharedBy struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type ListResponse struct {
	Items []VideoView `json:"items"`
	Total int64       `json:"total"`
}

func ToView(v Video) VideoView {
	return VideoView{
		ID:           v.ID,
		YouTubeID:    v.YouTubeID,
		URL:          v.URL,
		Title:        v.Title,
		Description:  v.Description,
		ThumbnailURL: v.ThumbnailURL,
		SharedAt:     v.CreatedAt,
		SharedBy: SharedBy{
			ID:    v.SharedBy.ID,
			Name:  v.SharedBy.Name,
			Email: v.SharedBy.Email,
		},
	}
}
