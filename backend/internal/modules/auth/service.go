package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"backend/internal/modules/users"
)

// Domain-level error sentinels. Handlers map these to HTTP statuses.
var (
	ErrEmailTaken         = errors.New("auth: email already registered")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
)

// Service holds the registration / sign-in business logic.
type Service struct {
	users  users.Repository
	tokens *TokenIssuer
}

// NewService wires the auth Service.
func NewService(repo users.Repository, tokens *TokenIssuer) *Service {
	return &Service{users: repo, tokens: tokens}
}

// SignUp creates a new user and returns an authenticated session.
func (s *Service) SignUp(ctx context.Context, req SignUpRequest) (Response, error) {
	email := normalizeEmail(req.Email)

	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return Response{}, ErrEmailTaken
	} else if !errors.Is(err, users.ErrNotFound) {
		return Response{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Response{}, fmt.Errorf("auth: hash password: %w", err)
	}

	user := &users.User{
		Email:        email,
		Name:         strings.TrimSpace(req.Name),
		PasswordHash: string(hash),
	}
	if err := s.users.Create(ctx, user); err != nil {
		return Response{}, err
	}
	return s.session(user)
}

// SignIn verifies credentials and returns an authenticated session.
func (s *Service) SignIn(ctx context.Context, req SignInRequest) (Response, error) {
	email := normalizeEmail(req.Email)

	user, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, users.ErrNotFound) {
		return Response{}, ErrInvalidCredentials
	}
	if err != nil {
		return Response{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return Response{}, ErrInvalidCredentials
	}
	return s.session(user)
}

// Refresh exchanges a valid refresh token for a fresh pair.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (Response, error) {
	claims, err := s.tokens.VerifyRefresh(req.RefreshToken)
	if err != nil {
		return Response{}, ErrInvalidCredentials
	}
	user, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil {
		return Response{}, ErrInvalidCredentials
	}
	return s.session(user)
}

func (s *Service) session(user *users.User) (Response, error) {
	pair, err := s.tokens.Issue(user.ID)
	if err != nil {
		return Response{}, err
	}
	return Response{
		User: SessionUser{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresAt:    pair.ExpiresAt,
	}, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
