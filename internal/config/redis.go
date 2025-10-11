package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func InitRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: redisHost,
		DB:   0,
	})

	ctx := context.Background()
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		slog.Error("failed to connect to redis", "error", err)
		panic(err)
	}
}