package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"backend/internal/modules/users"
)

// VideoCounter is declared here to avoid an import cycle with the videos package.
type VideoCounter interface {
	CountSharedAfter(ctx context.Context, after *time.Time, excludeUserID uuid.UUID) (int64, error)
}

// MaxUnread caps the API value so the client renders "9+" without a precise large count.
const MaxUnread = 99

type Service struct {
	users  users.Repository
	videos VideoCounter
}

func NewService(usersRepo users.Repository, videosRepo VideoCounter) *Service {
	return &Service{users: usersRepo, videos: videosRepo}
}

// UnreadCount counts shares since the user's last-seen marker, excluding their own, capped at MaxUnread.
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

// MarkSeen sets the user's last-seen marker to now, zeroing unread count across devices.
func (s *Service) MarkSeen(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	now := time.Now().UTC()
	if err := s.users.SetLastNotificationsSeenAt(ctx, userID, now); err != nil {
		return time.Time{}, err
	}
	return now, nil
}
