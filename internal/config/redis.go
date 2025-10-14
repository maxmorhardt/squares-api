package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

const (
	poolSize     int = 15
	minIdleConns int = 3
)

func InitRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:         redisHost,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})

	slog.Info("redis connection configured", "pool_size", poolSize, "min_idle_conns", minIdleConns)

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		panic(err)
	}
}
