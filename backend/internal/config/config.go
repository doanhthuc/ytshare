// Package config loads application configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App        AppConfig
	HTTP       HTTPConfig
	DB         DatabaseConfig
	Redis      RedisConfig
	JWT        JWTConfig
	CORS       CORSConfig
	Migrations MigrationsConfig
}

type AppConfig struct {
	Env      string
	LogLevel string
}

type HTTPConfig struct {
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// RedisConfig: when URL is set (e.g. REDIS_URL on Render/Heroku) it takes
// precedence over the discrete fields and is parsed by go-redis.
type RedisConfig struct {
	URL      string
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

// MigrationsConfig is read only by cmd/migrate; the server never auto-migrates.
type MigrationsConfig struct {
	Dir string
}

// DSN returns the libpq-style connection string for GORM.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// URL returns the URL-style connection string required by golang-migrate.
func (c DatabaseConfig) URL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(c.User),
		url.QueryEscape(c.Password),
		c.Host, c.Port, c.Name, c.SSLMode,
	)
}

// Load reads configuration from .env (if present) and the process environment.
func Load() (Config, error) {
	_ = godotenv.Load()

	httpPort, err := envInt("HTTP_PORT", 8080)
	if err != nil {
		return Config{}, err
	}
	dbPort, err := envInt("DB_PORT", 5432)
	if err != nil {
		return Config{}, err
	}
	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	accessTTLMin, err := envInt("JWT_ACCESS_TTL_MIN", 15)
	if err != nil {
		return Config{}, err
	}
	refreshTTLHours, err := envInt("JWT_REFRESH_TTL_HOURS", 24*7)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		App: AppConfig{
			Env:      envStr("APP_ENV", "development"),
			LogLevel: envStr("LOG_LEVEL", "info"),
		},
		HTTP: HTTPConfig{
			Port: httpPort,
		},
		DB: DatabaseConfig{
			Host:     envStr("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     envStr("DB_USER", "ytshare"),
			Password: envStr("DB_PASSWORD", "ytshare"),
			Name:     envStr("DB_NAME", "ytshare"),
			SSLMode:  envStr("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			URL:      envStr("REDIS_URL", ""),
			Addr:     envStr("REDIS_ADDR", "localhost:6379"),
			Password: envStr("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		JWT: JWTConfig{
			AccessSecret:  envStr("JWT_ACCESS_SECRET", "dev-access-secret"),
			RefreshSecret: envStr("JWT_REFRESH_SECRET", "dev-refresh-secret"),
			AccessTTL:     time.Duration(accessTTLMin) * time.Minute,
			RefreshTTL:    time.Duration(refreshTTLHours) * time.Hour,
		},
		CORS: CORSConfig{
			AllowedOrigins: splitAndTrim(envStr(
				"CORS_ALLOWED_ORIGINS",
				"http://localhost:5173,http://localhost:3000",
			)),
		},
		Migrations: MigrationsConfig{
			Dir: envStr("MIGRATIONS_DIR", "./migrations"),
		},
	}

	// DATABASE_URL overrides the discrete DB_* fields.
	if dbURL := envStr("DATABASE_URL", ""); dbURL != "" {
		if err := applyDatabaseURL(&cfg.DB, dbURL); err != nil {
			return Config{}, err
		}
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.JWT.AccessSecret == "" || c.JWT.RefreshSecret == "" {
		return errors.New("config: JWT secrets must be set")
	}
	if c.HTTP.Port <= 0 {
		return errors.New("config: HTTP_PORT must be positive")
	}
	return nil
}

func envStr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: invalid integer for %s: %w", key, err)
	}
	return n, nil
}

// applyDatabaseURL parses a libpq URL into cfg. Missing port defaults to 5432;
// missing sslmode defaults to "require" (most managed Postgres providers require SSL).
func applyDatabaseURL(cfg *DatabaseConfig, raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("config: invalid DATABASE_URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("config: DATABASE_URL must use postgres:// scheme, got %q", u.Scheme)
	}
	cfg.Host = u.Hostname()
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("config: invalid DATABASE_URL port: %w", err)
		}
		cfg.Port = n
	} else {
		cfg.Port = 5432
	}
	if u.User != nil {
		cfg.User = u.User.Username()
		if pw, ok := u.User.Password(); ok {
			cfg.Password = pw
		}
	}
	cfg.Name = strings.TrimPrefix(u.Path, "/")
	if v := u.Query().Get("sslmode"); v != "" {
		cfg.SSLMode = v
	} else {
		cfg.SSLMode = "require"
	}
	return nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
