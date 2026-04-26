package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"backend/internal/config"
)

// NewRedis opens a Redis client and verifies the connection with a PING.
func NewRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	var client *redis.Client
	if cfg.URL != "" {
		opt, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("database: parse REDIS_URL: %w", err)
		}
		client = redis.NewClient(opt)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("database: redis ping: %w", err)
	}
	return client, nil
}
