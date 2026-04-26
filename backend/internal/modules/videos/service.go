package videos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/jobs"
	"backend/internal/modules/notifications"
	"backend/internal/modules/users"
)

// Domain-level error sentinels.
var (
	ErrInvalidURL    = errors.New("videos: invalid url")
	ErrAlreadyShared = errors.New("videos: already shared")
)

const (
	listCacheKey = "videos:list:recent"
	listCacheTTL = 30 * time.Second
)

// Service contains the video sharing business logic.
type Service struct {
	repo   Repository
	users  users.Repository
	cache  cache.Cache
	hub    *notifications.Hub
	worker *jobs.Worker
	log    *zap.Logger
}

// NewService wires the videos Service.
func NewService(
	repo Repository,
	usersRepo users.Repository,
	c cache.Cache,
	hub *notifications.Hub,
	worker *jobs.Worker,
	log *zap.Logger,
) *Service {
	return &Service{
		repo:   repo,
		users:  usersRepo,
		cache:  c,
		hub:    hub,
		worker: worker,
		log:    log,
	}
}

// Share validates a URL, persists the shared video and broadcasts a
// notification asynchronously via the background worker.
func (s *Service) Share(ctx context.Context, sharerID uuid.UUID, req ShareRequest) (VideoView, error) {
	youtubeID, err := ExtractYouTubeID(req.URL)
	if err != nil {
		return VideoView{}, ErrInvalidURL
	}

	if existing, err := s.repo.FindByYouTubeID(ctx, youtubeID); err == nil && existing != nil {
		return VideoView{}, ErrAlreadyShared
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return VideoView{}, err
	}

	sharer, err := s.users.FindByID(ctx, sharerID)
	if err != nil {
		return VideoView{}, fmt.Errorf("videos: load sharer: %w", err)
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fmt.Sprintf("YouTube video %s", youtubeID)
	}

	video := &Video{
		YouTubeID:    youtubeID,
		URL:          WatchURL(youtubeID),
		Title:        title,
		Description:  strings.TrimSpace(req.Description),
		ThumbnailURL: ThumbnailURL(youtubeID),
		SharedByID:   sharer.ID,
		SharedBy:     *sharer,
	}
	if err := s.repo.Create(ctx, video); err != nil {
		return VideoView{}, err
	}

	// Invalidate the cached list and dispatch the broadcast off the request
	// goroutine. The HTTP response returns immediately while the WebSocket
	// fan-out happens in the background worker.
	if err := s.cache.Delete(ctx, listCacheKey); err != nil {
		s.log.Warn("videos_cache_delete", zap.Error(err))
	}

	view := ToView(*video)
	s.worker.Submit(func(_ context.Context) error {
		s.hub.Broadcast(notifications.Event{
			Type:      notifications.EventVideoShared,
			Timestamp: time.Now().UTC(),
			Payload: notifications.VideoSharedPayload{
				VideoID:      video.ID,
				YouTubeID:    video.YouTubeID,
				Title:        video.Title,
				ThumbnailURL: video.ThumbnailURL,
				SharedByID:   sharer.ID,
				SharedByName: sharer.Name,
			},
		})
		return nil
	})

	return view, nil
}

// List returns the most recent shared videos. The first page is cached.
func (s *Service) List(ctx context.Context, limit, offset int) (ListResponse, error) {
	if offset == 0 && limit <= 20 {
		var cached ListResponse
		switch err := s.cache.Get(ctx, listCacheKey, &cached); {
		case err == nil:
			return cached, nil
		case errors.Is(err, cache.ErrMiss):
			// fall through
		default:
			s.log.Warn("videos_cache_get", zap.Error(err))
		}
	}

	items, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}
	views := make([]VideoView, 0, len(items))
	for _, v := range items {
		views = append(views, ToView(v))
	}
	resp := ListResponse{Items: views, Total: total}

	if offset == 0 && limit <= 20 {
		if err := s.cache.Set(ctx, listCacheKey, resp, listCacheTTL); err != nil {
			s.log.Warn("videos_cache_set", zap.Error(err))
		}
	}
	return resp, nil
}
