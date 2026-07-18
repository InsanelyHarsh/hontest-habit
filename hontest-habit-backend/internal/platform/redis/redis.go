package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// Config holds Redis connection parameters.
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// ConfigFromEnv reads REDIS_* keys via viper.
func ConfigFromEnv() Config {
	return Config{
		Host:     viper.GetString("REDIS_HOST"),
		Port:     viper.GetInt("REDIS_PORT"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       viper.GetInt("REDIS_DB"),
	}
}

// New creates a Redis client and verifies connectivity. The caller is
// responsible for calling client.Close().
func New(ctx context.Context, cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	slog.Info("redis: connected", "addr", client.Options().Addr, "db", cfg.DB)
	return client, nil
}
