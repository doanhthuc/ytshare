// Package auth handles registration, sign-in and JWT issuing.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"backend/internal/config"
)

// TokenKind distinguishes access tokens from refresh tokens.
type TokenKind string

const (
	// AccessKind is short-lived and embedded in the Authorization header.
	AccessKind TokenKind = "access"
	// RefreshKind is longer-lived and is exchanged for a new access token.
	RefreshKind TokenKind = "refresh"
)

// Claims is our custom JWT claim payload.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Kind   TokenKind `json:"knd"`
	jwt.RegisteredClaims
}

// TokenPair groups a fresh access/refresh tuple.
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// TokenIssuer signs and verifies JWTs using HMAC.
type TokenIssuer struct {
	cfg config.JWTConfig
}

// NewTokenIssuer constructs a TokenIssuer.
func NewTokenIssuer(cfg config.JWTConfig) *TokenIssuer {
	return &TokenIssuer{cfg: cfg}
}

// Issue mints a fresh pair for the supplied user id.
func (t *TokenIssuer) Issue(userID uuid.UUID) (TokenPair, error) {
	now := time.Now()
	access, accessExp, err := t.sign(userID, AccessKind, now, t.cfg.AccessTTL, t.cfg.AccessSecret)
	if err != nil {
		return TokenPair{}, err
	}
	refresh, _, err := t.sign(userID, RefreshKind, now, t.cfg.RefreshTTL, t.cfg.RefreshSecret)
	if err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    accessExp,
	}, nil
}

// VerifyAccess validates an access token and returns its claims.
func (t *TokenIssuer) VerifyAccess(token string) (*Claims, error) {
	return t.verify(token, AccessKind, t.cfg.AccessSecret)
}

// VerifyRefresh validates a refresh token and returns its claims.
func (t *TokenIssuer) VerifyRefresh(token string) (*Claims, error) {
	return t.verify(token, RefreshKind, t.cfg.RefreshSecret)
}

func (t *TokenIssuer) sign(
	userID uuid.UUID,
	kind TokenKind,
	now time.Time,
	ttl time.Duration,
	secret string,
) (string, time.Time, error) {
	exp := now.Add(ttl)
	claims := Claims{
		UserID: userID,
		Kind:   kind,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("auth: sign %s: %w", kind, err)
	}
	return signed, exp, nil
}

func (t *TokenIssuer) verify(token string, want TokenKind, secret string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %s", t.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("auth: parse token: %w", err)
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("auth: invalid token")
	}
	if claims.Kind != want {
		return nil, fmt.Errorf("auth: expected %s token, got %s", want, claims.Kind)
	}
	return claims, nil
}
