package auth_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backend/internal/config"
	"backend/internal/modules/auth"
)

func newIssuer(t *testing.T) *auth.TokenIssuer {
	t.Helper()
	return auth.NewTokenIssuer(config.JWTConfig{
		AccessSecret:  "access-secret",
		RefreshSecret: "refresh-secret",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    24 * time.Hour,
	})
}

func TestTokenIssuer_IssueAndVerify(t *testing.T) {
	t.Parallel()
	issuer := newIssuer(t)
	uid := uuid.New()

	pair, err := issuer.Issue(uid)
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.True(t, pair.ExpiresAt.After(time.Now()))

	access, err := issuer.VerifyAccess(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, uid, access.UserID)
	assert.Equal(t, auth.AccessKind, access.Kind)

	refresh, err := issuer.VerifyRefresh(pair.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, uid, refresh.UserID)
	assert.Equal(t, auth.RefreshKind, refresh.Kind)
}

func TestTokenIssuer_RejectsCrossKind(t *testing.T) {
	t.Parallel()
	issuer := newIssuer(t)
	pair, err := issuer.Issue(uuid.New())
	require.NoError(t, err)

	_, err = issuer.VerifyAccess(pair.RefreshToken)
	assert.Error(t, err)

	_, err = issuer.VerifyRefresh(pair.AccessToken)
	assert.Error(t, err)
}

func TestTokenIssuer_RejectsTamperedToken(t *testing.T) {
	t.Parallel()
	issuer := newIssuer(t)
	pair, err := issuer.Issue(uuid.New())
	require.NoError(t, err)

	tampered := pair.AccessToken + "x"
	_, err = issuer.VerifyAccess(tampered)
	assert.Error(t, err)
}
