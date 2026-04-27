package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/internal/modules/auth"
	"backend/internal/modules/users"
)

type memRepo struct {
	byEmail map[string]*users.User
	byID    map[uuid.UUID]*users.User
}

func newMemRepo() *memRepo {
	return &memRepo{
		byEmail: map[string]*users.User{},
		byID:    map[uuid.UUID]*users.User{},
	}
}

func (m *memRepo) Create(_ context.Context, u *users.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *memRepo) FindByEmail(_ context.Context, email string) (*users.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, users.ErrNotFound
	}
	return u, nil
}

func (m *memRepo) FindByID(_ context.Context, id uuid.UUID) (*users.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, users.ErrNotFound
	}
	return u, nil
}

func (m *memRepo) SetLastNotificationsSeenAt(_ context.Context, id uuid.UUID, at time.Time) error {
	u, ok := m.byID[id]
	if !ok {
		return users.ErrNotFound
	}
	u.LastNotificationsSeenAt = &at
	return nil
}

func newService(t *testing.T) (*auth.Service, *memRepo) {
	t.Helper()
	repo := newMemRepo()
	svc := auth.NewService(repo, newIssuer(t))
	return svc, repo
}

func TestService_SignUp(t *testing.T) {
	t.Parallel()
	svc, repo := newService(t)

	resp, err := svc.SignUp(context.Background(), auth.SignUpRequest{
		Email:    "Alice@example.com",
		Name:     "Alice",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", resp.User.Email)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)

	_, err = svc.SignUp(context.Background(), auth.SignUpRequest{
		Email:    "alice@example.com",
		Name:     "Alice 2",
		Password: "password123",
	})
	assert.True(t, errors.Is(err, auth.ErrEmailTaken))

	u, ok := repo.byEmail["alice@example.com"]
	require.True(t, ok)
	assert.NotEqual(t, "password123", u.PasswordHash)
}

func TestService_SignIn(t *testing.T) {
	t.Parallel()
	svc, _ := newService(t)

	_, err := svc.SignUp(context.Background(), auth.SignUpRequest{
		Email:    "bob@example.com",
		Name:     "Bob",
		Password: "password123",
	})
	require.NoError(t, err)

	resp, err := svc.SignIn(context.Background(), auth.SignInRequest{
		Email:    "BOB@example.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, "bob@example.com", resp.User.Email)

	_, err = svc.SignIn(context.Background(), auth.SignInRequest{
		Email:    "bob@example.com",
		Password: "wrong",
	})
	assert.True(t, errors.Is(err, auth.ErrInvalidCredentials))

	_, err = svc.SignIn(context.Background(), auth.SignInRequest{
		Email:    "ghost@example.com",
		Password: "password123",
	})
	assert.True(t, errors.Is(err, auth.ErrInvalidCredentials))
}

func TestService_Refresh(t *testing.T) {
	t.Parallel()
	svc, _ := newService(t)
	resp, err := svc.SignUp(context.Background(), auth.SignUpRequest{
		Email:    "carol@example.com",
		Name:     "Carol",
		Password: "password123",
	})
	require.NoError(t, err)

	refreshed, err := svc.Refresh(context.Background(), auth.RefreshRequest{
		RefreshToken: resp.RefreshToken,
	})
	require.NoError(t, err)
	assert.Equal(t, resp.User.ID, refreshed.User.ID)
	assert.NotEmpty(t, refreshed.AccessToken)

	_, err = svc.Refresh(context.Background(), auth.RefreshRequest{
		RefreshToken: "nope",
	})
	assert.True(t, errors.Is(err, auth.ErrInvalidCredentials))
}
