package videos

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository persists Video records.
type Repository interface {
	Create(ctx context.Context, v *Video) error
	List(ctx context.Context, limit, offset int) ([]Video, int64, error)
	FindByYouTubeID(ctx context.Context, youtubeID string) (*Video, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Video, error)
	CountSharedAfter(ctx context.Context, after *time.Time, excludeUserID uuid.UUID) (int64, error)
}

type gormRepo struct {
	db *gorm.DB
}

// NewRepository constructs the default Repository implementation.
func NewRepository(db *gorm.DB) Repository {
	return &gormRepo{db: db}
}

// Create inserts a new video row.
func (r *gormRepo) Create(ctx context.Context, v *Video) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(v).Error; err != nil {
		return fmt.Errorf("videos: create: %w", err)
	}
	return nil
}

// List returns the most recent videos with their sharer eagerly loaded.
func (r *gormRepo) List(ctx context.Context, limit, offset int) ([]Video, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var (
		items []Video
		total int64
	)
	if err := r.db.WithContext(ctx).Model(&Video{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("videos: count: %w", err)
	}
	err := r.db.WithContext(ctx).
		Preload("SharedBy").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("videos: list: %w", err)
	}
	return items, total, nil
}

// FindByYouTubeID returns the video matching the supplied YouTube id, if any.
func (r *gormRepo) FindByYouTubeID(ctx context.Context, youtubeID string) (*Video, error) {
	var v Video
	err := r.db.WithContext(ctx).Where("youtube_id = ?", youtubeID).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CountSharedAfter counts videos created strictly after `after` that were
// not shared by `excludeUserID`. A nil `after` counts every video except the
// caller's own — i.e. the first-time bell badge before any "seen" marker.
func (r *gormRepo) CountSharedAfter(ctx context.Context, after *time.Time, excludeUserID uuid.UUID) (int64, error) {
	q := r.db.WithContext(ctx).Model(&Video{}).Where("shared_by_id <> ?", excludeUserID)
	if after != nil {
		q = q.Where("created_at > ?", *after)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return 0, fmt.Errorf("videos: count shared after: %w", err)
	}
	return n, nil
}

// FindByID returns a single video by id.
func (r *gormRepo) FindByID(ctx context.Context, id uuid.UUID) (*Video, error) {
	var v Video
	err := r.db.WithContext(ctx).Preload("SharedBy").Where("id = ?", id).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
