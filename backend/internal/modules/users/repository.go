package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotFound = errors.New("users: not found")

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	SetLastNotificationsSeenAt(ctx context.Context, id uuid.UUID, at time.Time) error
}

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepo{db: db}
}

func (r *gormRepo) Create(ctx context.Context, u *User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("users: create: %w", err)
	}
	return nil
}

// FindByEmail returns the user with the email or ErrNotFound.
func (r *gormRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("users: find by email: %w", err)
	}
	return &u, nil
}

func (r *gormRepo) SetLastNotificationsSeenAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	res := r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", id).
		Update("last_notifications_seen_at", at)
	if res.Error != nil {
		return fmt.Errorf("users: set last_notifications_seen_at: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindByID returns the user with the id or ErrNotFound.
func (r *gormRepo) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("users: find by id: %w", err)
	}
	return &u, nil
}
