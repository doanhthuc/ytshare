package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"backend/internal/modules/users"
)

// VideoCounter is the slice of the videos repository the notifications
// service needs. Defined here so this package does not import the videos
// package and create an import cycle (videos already depends on the
// notifications hub).
type VideoCounter interface {
	CountSharedAfter(ctx context.Context, after *time.Time, excludeUserID uuid.UUID) (int64, error)
}

// MaxUnread caps the value the API returns so the client can render a "9+"
// style badge without paying for a precise large count on the database.
const MaxUnread = 99

// Service exposes per-user notification read-state operations. It piggybacks
// on the videos table — every share is, by definition, a notification — and
// stores the per-user "last seen" marker on the users row (Option A).
type Service struct {
	users  users.Repository
	videos VideoCounter
}

// NewService wires the notifications Service.
func NewService(usersRepo users.Repository, videosRepo VideoCounter) *Service {
	return &Service{users: usersRepo, videos: videosRepo}
}

// UnreadCount returns the number of shares created after the user's
// last-seen marker, excluding their own shares. The result is capped at
// MaxUnread.
func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("notifications: load user: %w", err)
	}
	n, err := s.videos.CountSharedAfter(ctx, u.LastNotificationsSeenAt, userID)
	if err != nil {
		return 0, err
	}
	if n > MaxUnread {
		n = MaxUnread
	}
	return n, nil
}

// MarkSeen sets the user's last-seen marker to NOW(), zeroing the unread
// count for that account on every device.
func (s *Service) MarkSeen(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	now := time.Now().UTC()
	if err := s.users.SetLastNotificationsSeenAt(ctx, userID, now); err != nil {
		return time.Time{}, err
	}
	return now, nil
}
