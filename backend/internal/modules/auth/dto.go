package auth

import (
	"time"

	"github.com/google/uuid"
)

// SignUpRequest is the payload for POST /auth/signup.
type SignUpRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Name     string `json:"name"     validate:"required,min=1,max=120"`
	Password string `json:"password" validate:"required,min=6,max=72"`
}

// SignInRequest is the payload for POST /auth/signin.
type SignInRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=1,max=72"`
}

// RefreshRequest is the payload for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// SessionUser is the user view embedded in auth responses.
type SessionUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

// AuthResponse is the standard envelope for sign-in / sign-up / refresh.
type AuthResponse struct {
	User         SessionUser `json:"user"`
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresAt    time.Time   `json:"expiresAt"`
}
