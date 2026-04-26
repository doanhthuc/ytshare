package auth

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"backend/internal/httpx"
)

// Handler exposes the auth endpoints.
type Handler struct {
	svc *Service
	v   *validator.Validate
}

// NewHandler returns the auth HTTP handler.
func NewHandler(svc *Service, v *validator.Validate) *Handler {
	return &Handler{svc: svc, v: v}
}

// RegisterRoutes mounts the auth routes onto the supplied router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", h.signUp)
		r.Post("/signin", h.signIn)
		r.Post("/refresh", h.refresh)
	})
}

func (h *Handler) signUp(w http.ResponseWriter, r *http.Request) {
	var req SignUpRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.v.Struct(req); err != nil {
		httpx.WriteError(w, httpx.NewError(
			http.StatusBadRequest, "validation_failed", err.Error(),
		))
		return
	}
	resp, err := h.svc.SignUp(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.v.Struct(req); err != nil {
		httpx.WriteError(w, httpx.NewError(
			http.StatusBadRequest, "validation_failed", err.Error(),
		))
		return
	}
	resp, err := h.svc.SignIn(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := h.v.Struct(req); err != nil {
		httpx.WriteError(w, httpx.NewError(
			http.StatusBadRequest, "validation_failed", err.Error(),
		))
		return
	}
	resp, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		h.writeAuthError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrEmailTaken):
		httpx.WriteError(w, httpx.NewError(
			http.StatusConflict, "email_taken", "email already registered",
		))
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, httpx.NewError(
			http.StatusUnauthorized, "invalid_credentials", "invalid email or password",
		))
	default:
		httpx.WriteError(w, err)
	}
}
