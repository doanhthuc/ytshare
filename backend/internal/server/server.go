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
	"github.com/redis/go-redis/v9"
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

// Deps holds external dependencies for the HTTP server.
// Redis is optional: when nil, the in-process Hub is the only fan-out path.
type Deps struct {
	Config config.Config
	Logger *zap.Logger
	DB     *gorm.DB
	Cache  cache.Cache
	Redis  *redis.Client
	Worker *jobs.Worker
}

// Build wires the HTTP server. ctx bounds background goroutines spawned here
// (e.g. Redis Streams subscriber); cancel before Run shuts down the server.
func Build(ctx context.Context, d Deps) *http.Server {
	v := validator.New(validator.WithRequiredStructEnabled())

	userRepo := users.NewRepository(d.DB)

	tokens := auth.NewTokenIssuer(d.Config.JWT)
	authSvc := auth.NewService(userRepo, tokens)
	authHandler := auth.NewHandler(authSvc, v)

	hub := notifications.NewHub(d.Logger)
	// Tear down on ctx cancel for graceful shutdown of clients' writeLoops.
	go func() {
		<-ctx.Done()
		hub.Close()
	}()

	publisher := buildPublisher(ctx, hub, d)
	replayer, _ := publisher.(notifications.Replayer)

	videoRepo := videos.NewRepository(d.DB)
	notifSvc := notifications.NewService(userRepo, videoRepo)
	notifHandler := notifications.NewHandler(hub, notifSvc, replayer, d.Config.CORS.AllowedOrigins, d.Logger)

	videoSvc := videos.NewService(videoRepo, userRepo, d.Cache, publisher, d.Worker, d.Logger)
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
		authHandler.RegisterRoutes(r)
		videoHandler.RegisterPublicRoutes(r)

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

// buildPublisher selects between Redis Streams (cross-replica fan-out) and
// the in-process Hub when Redis is unavailable.
func buildPublisher(ctx context.Context, hub *notifications.Hub, d Deps) notifications.Publisher {
	if d.Redis == nil {
		d.Logger.Info("notifications_publisher", zap.String("transport", "local"))
		return notifications.NewLocalPublisher(hub, d.Logger)
	}
	d.Logger.Info("notifications_publisher", zap.String("transport", "redis_streams"))
	sub := notifications.NewSubscriber(d.Redis, hub, d.Logger)
	go func() {
		if err := sub.Run(ctx); err != nil {
			d.Logger.Error("notifications_subscriber_exited", zap.Error(err))
		}
	}()
	return notifications.NewStreamPublisher(d.Redis, d.Logger)
}

// Run blocks until ctx is cancelled, then performs graceful shutdown.
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
