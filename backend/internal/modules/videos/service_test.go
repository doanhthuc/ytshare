package videos_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/jobs"
	"backend/internal/modules/notifications"
	"backend/internal/modules/users"
	"backend/internal/modules/videos"
)

type fakeUserRepo struct {
	user *users.User
}

func (f *fakeUserRepo) Create(_ context.Context, _ *users.User) error { return nil }
func (f *fakeUserRepo) FindByEmail(_ context.Context, _ string) (*users.User, error) {
	return nil, users.ErrNotFound
}

func (f *fakeUserRepo) FindByID(_ context.Context, id uuid.UUID) (*users.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, users.ErrNotFound
	}
	return f.user, nil
}

func (f *fakeUserRepo) SetLastNotificationsSeenAt(_ context.Context, id uuid.UUID, at time.Time) error {
	if f.user == nil || f.user.ID != id {
		return users.ErrNotFound
	}
	f.user.LastNotificationsSeenAt = &at
	return nil
}

type fakeVideoRepo struct {
	mu     sync.Mutex
	stored []videos.Video
}

func (r *fakeVideoRepo) Create(_ context.Context, v *videos.Video) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v.ID = uuid.New()
	v.CreatedAt = time.Now()
	r.stored = append(r.stored, *v)
	return nil
}

func (r *fakeVideoRepo) List(_ context.Context, limit, offset int) ([]videos.Video, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if offset > len(r.stored) {
		offset = len(r.stored)
	}
	end := offset + limit
	if end > len(r.stored) {
		end = len(r.stored)
	}
	out := append([]videos.Video(nil), r.stored[offset:end]...)
	return out, int64(len(r.stored)), nil
}

func (r *fakeVideoRepo) FindByYouTubeID(_ context.Context, ytid string) (*videos.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.stored {
		if r.stored[i].YouTubeID == ytid {
			return &r.stored[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeVideoRepo) CountSharedAfter(_ context.Context, after *time.Time, excludeUserID uuid.UUID) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, v := range r.stored {
		if v.SharedByID == excludeUserID {
			continue
		}
		if after != nil && !v.CreatedAt.After(*after) {
			continue
		}
		n++
	}
	return n, nil
}

func (r *fakeVideoRepo) FindByID(_ context.Context, id uuid.UUID) (*videos.Video, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.stored {
		if r.stored[i].ID == id {
			return &r.stored[i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func newSvc(t *testing.T, sharer *users.User) (*videos.Service, *fakeVideoRepo, *jobs.Worker) {
	t.Helper()
	log := zap.NewNop()
	repo := &fakeVideoRepo{}
	worker := jobs.NewWorker(1, 8, log)
	t.Cleanup(worker.Stop)

	svc := videos.NewService(
		repo,
		&fakeUserRepo{user: sharer},
		cache.NewMemoryCache(),
		notifications.NewLocalPublisher(notifications.NewHub(log), log),
		worker,
		log,
	)
	return svc, repo, worker
}

func TestService_Share_Success(t *testing.T) {
	t.Parallel()
	user := &users.User{ID: uuid.New(), Email: "alice@example.com", Name: "Alice"}
	svc, repo, _ := newSvc(t, user)

	view, err := svc.Share(context.Background(), user.ID, videos.ShareRequest{
		URL:   "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
		Title: "Never Gonna Give You Up",
	})
	require.NoError(t, err)
	assert.Equal(t, "dQw4w9WgXcQ", view.YouTubeID)
	assert.Equal(t, "Never Gonna Give You Up", view.Title)
	assert.Equal(t, user.ID, view.SharedBy.ID)
	assert.Len(t, repo.stored, 1)
}

func TestService_Share_InvalidURL(t *testing.T) {
	t.Parallel()
	user := &users.User{ID: uuid.New(), Email: "x@y.z", Name: "X"}
	svc, _, _ := newSvc(t, user)

	_, err := svc.Share(context.Background(), user.ID, videos.ShareRequest{
		URL: "https://vimeo.com/12345",
	})
	assert.True(t, errors.Is(err, videos.ErrInvalidURL))
}

func TestService_Share_DuplicateRejected(t *testing.T) {
	t.Parallel()
	user := &users.User{ID: uuid.New(), Email: "x@y.z", Name: "X"}
	svc, _, _ := newSvc(t, user)

	url := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
	_, err := svc.Share(context.Background(), user.ID, videos.ShareRequest{URL: url})
	require.NoError(t, err)

	_, err = svc.Share(context.Background(), user.ID, videos.ShareRequest{URL: url})
	assert.True(t, errors.Is(err, videos.ErrAlreadyShared))
}

func TestService_List_UsesCache(t *testing.T) {
	t.Parallel()
	user := &users.User{ID: uuid.New(), Email: "x@y.z", Name: "X"}
	svc, repo, _ := newSvc(t, user)

	_, err := svc.Share(context.Background(), user.ID, videos.ShareRequest{
		URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
	})
	require.NoError(t, err)

	first, err := svc.List(context.Background(), 20, 0)
	require.NoError(t, err)
	assert.Len(t, first.Items, 1)

	// Cache should serve the stale list after the underlying repo is wiped.
	repo.mu.Lock()
	repo.stored = nil
	repo.mu.Unlock()

	second, err := svc.List(context.Background(), 20, 0)
	require.NoError(t, err)
	assert.Len(t, second.Items, 1, "expected cached response")
}
