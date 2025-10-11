package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		panic(err)
	}
}

func PublishGridUpdate(ctx context.Context, message any) error {
	return RedisClient.Publish(ctx, "grid-updates", message).Err()
}
