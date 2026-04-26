package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"backend/internal/httpx"
	"backend/internal/modules/auth"
)

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
)

// Authenticator returns middleware that validates the access token in the
// `Authorization: Bearer ...` header and stuffs the user id into the context.
func Authenticator(tokens *auth.TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if raw == "" {
				// allow query-string fallback for the WebSocket upgrade.
				raw = r.URL.Query().Get("access_token")
			}
			token := stripBearer(raw)
			if token == "" {
				httpx.WriteError(w, httpx.NewError(
					http.StatusUnauthorized, "unauthorized", "missing access token",
				))
				return
			}

			claims, err := tokens.VerifyAccess(token)
			if err != nil {
				httpx.WriteError(w, httpx.NewError(
					http.StatusUnauthorized, "unauthorized", "invalid access token",
				))
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user id (or uuid.Nil).
func UserIDFromContext(ctx context.Context) uuid.UUID {
	if v, ok := ctx.Value(ctxKeyUserID).(uuid.UUID); ok {
		return v
	}
	return uuid.Nil
}

func stripBearer(raw string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(raw, prefix) {
		return strings.TrimSpace(raw[len(prefix):])
	}
	return strings.TrimSpace(raw)
}
