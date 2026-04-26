// Binary server is the HTTP entry point for the YouTube share service.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"backend/internal/cache"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/jobs"
	"backend/internal/logger"
	"backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		zap.L().Error("fatal", zap.Error(err))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.App.LogLevel)
	zap.ReplaceGlobals(log)
	defer func() { _ = log.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The server NEVER applies schema migrations. Migrations are an
	// explicit operator step — `make migrate`, the one-shot `migrate`
	// service in docker-compose, or your CI/CD pipeline.
	db, err := database.NewPostgres(ctx, cfg.DB)
	if err != nil {
		return err
	}

	redisClient, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	defer func() { _ = redisClient.Close() }()

	worker := jobs.NewWorker(4, 256, log)
	defer worker.Stop()

	srv := server.Build(server.Deps{
		Config: cfg,
		Logger: log,
		DB:     db,
		Cache:  cache.NewRedisCache(redisClient),
		Worker: worker,
	})
	return server.Run(ctx, srv, log)
}
