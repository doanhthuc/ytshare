package videos

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

func NewRepository(db *gorm.DB) Repository {
	return &gormRepo{db: db}
}

func (r *gormRepo) Create(ctx context.Context, v *Video) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(v).Error; err != nil {
		return fmt.Errorf("videos: create: %w", err)
	}
	return nil
}

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

func (r *gormRepo) FindByYouTubeID(ctx context.Context, youtubeID string) (*Video, error) {
	var v Video
	err := r.db.WithContext(ctx).Where("youtube_id = ?", youtubeID).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// CountSharedAfter counts videos created after `after` not shared by excludeUserID.
// nil `after` counts every video except caller's own (first-time bell badge).
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

func (r *gormRepo) FindByID(ctx context.Context, id uuid.UUID) (*Video, error) {
	var v Video
	err := r.db.WithContext(ctx).Preload("SharedBy").Where("id = ?", id).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}
