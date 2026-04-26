package videos

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"backend/internal/httpx"
	"backend/internal/middleware"
)

// Handler exposes the videos endpoints.
type Handler struct {
	svc *Service
	v   *validator.Validate
}

// NewHandler returns the videos HTTP handler.
func NewHandler(svc *Service, v *validator.Validate) *Handler {
	return &Handler{svc: svc, v: v}
}

// RegisterPublicRoutes mounts the read-only routes (no auth required).
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/videos", h.list)
}

// RegisterPrivateRoutes mounts the routes that require an access token.
func (h *Handler) RegisterPrivateRoutes(r chi.Router) {
	r.Post("/videos", h.share)
}

func (h *Handler) share(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req ShareRequest
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

	view, err := h.svc.Share(r.Context(), userID, req)
	switch {
	case errors.Is(err, ErrInvalidURL):
		httpx.WriteError(w, httpx.NewError(
			http.StatusBadRequest, "invalid_url", "could not parse YouTube URL",
		))
	case errors.Is(err, ErrAlreadyShared):
		httpx.WriteError(w, httpx.NewError(
			http.StatusConflict, "already_shared", "this video has already been shared",
		))
	case err != nil:
		httpx.WriteError(w, err)
	default:
		httpx.JSON(w, http.StatusCreated, view)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	resp, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
