// Package server wires the HTTP router and lifecycle.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"backend/internal/cache"
	"backend/internal/config"
	"backend/internal/httpx"
	"backend/internal/jobs"
	"backend/internal/middleware"
	"backend/internal/modules/auth"
	"backend/internal/modules/notifications"
	"backend/internal/modules/users"
	"backend/internal/modules/videos"
)

// Deps groups every external dependency needed by the HTTP server.
type Deps struct {
	Config config.Config
	Logger *zap.Logger
	DB     *gorm.DB
	Cache  cache.Cache
	Worker *jobs.Worker
}

// Build constructs a fully-wired *http.Server ready to be started.
func Build(d Deps) *http.Server {
	v := validator.New(validator.WithRequiredStructEnabled())

	userRepo := users.NewRepository(d.DB)

	tokens := auth.NewTokenIssuer(d.Config.JWT)
	authSvc := auth.NewService(userRepo, tokens)
	authHandler := auth.NewHandler(authSvc, v)

	hub := notifications.NewHub(d.Logger)
	videoRepo := videos.NewRepository(d.DB)
	notifSvc := notifications.NewService(userRepo, videoRepo)
	notifHandler := notifications.NewHandler(hub, notifSvc, d.Config.CORS.AllowedOrigins, d.Logger)

	videoSvc := videos.NewService(videoRepo, userRepo, d.Cache, hub, d.Worker, d.Logger)
	videoHandler := videos.NewHandler(videoSvc, v)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestLogger(d.Logger))
	r.Use(middleware.CORS(d.Config.CORS.AllowedOrigins))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes.
		authHandler.RegisterRoutes(r)
		videoHandler.RegisterPublicRoutes(r)

		// Protected routes.
		r.Group(func(pr chi.Router) {
			pr.Use(middleware.Authenticator(tokens))
			videoHandler.RegisterPrivateRoutes(pr)
			notifHandler.RegisterRoutes(pr)
		})
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, "not_found", "route not found"))
	})

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", d.Config.HTTP.Port),
		Handler:           r,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// Run starts the server and blocks until ctx is cancelled, then performs
// a graceful shutdown.
func Run(ctx context.Context, srv *http.Server, log *zap.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("http_listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		log.Info("http_shutting_down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server: shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
