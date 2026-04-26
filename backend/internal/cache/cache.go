// Package cache abstracts the distributed cache used by the services.
//
// The interface lets us swap Redis for an in-memory cache in tests.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrMiss is returned when the key is absent.
var ErrMiss = errors.New("cache: miss")

// Cache is the contract every cache backend must satisfy.
type Cache interface {
	Get(ctx context.Context, key string, dst any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// RedisCache wraps a redis.Client with JSON serialization.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache wraps the supplied client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

var _ Cache = (*RedisCache)(nil)

// Get unmarshals the value at key into dst.
func (c *RedisCache) Get(ctx context.Context, key string, dst any) error {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrMiss
	}
	if err != nil {
		return fmt.Errorf("cache: get %q: %w", key, err)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return nil
}

// Set marshals value and stores it under key with the supplied TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	if err := c.client.Set(ctx, key, raw, ttl).Err(); err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

// Delete removes the supplied keys (no error if missing).
func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: delete: %w", err)
	}
	return nil
}

// MemoryCache is a process-local fallback used in tests.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]memoryItem
}

type memoryItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemoryCache returns an empty in-memory cache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{items: make(map[string]memoryItem)}
}

var _ Cache = (*MemoryCache)(nil)

// Get unmarshals the value into dst.
func (c *MemoryCache) Get(_ context.Context, key string, dst any) error {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return ErrMiss
	}
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return ErrMiss
	}
	return json.Unmarshal(item.value, dst)
}

// Set stores a value with TTL (zero TTL = no expiry).
func (c *MemoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	expires := time.Time{}
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.items[key] = memoryItem{value: raw, expiresAt: expires}
	c.mu.Unlock()
	return nil
}

// Delete removes the supplied keys.
func (c *MemoryCache) Delete(_ context.Context, keys ...string) error {
	c.mu.Lock()
	for _, k := range keys {
		delete(c.items, k)
	}
	c.mu.Unlock()
	return nil
}
