package config

import (
	"context"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient      *redis.Client
)

const (
	poolSize       = 15
	minIdleConns   = 3
)

func InitRedis() {
	redisClient = newRedisClient(Env().Redis.Host)
	slog.Info("redis client configured", "pool_size", poolSize, "min_idle_conns", minIdleConns)

	if err := pingRedis(); err != nil {
		slog.Error("failed to connect to redis", "error", err)
		panic(err)
	}

	slog.Info("redis connection established successfully")
}

func newRedisClient(redisHost string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         redisHost,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})
}

func pingRedis() error {
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		slog.Warn("error while pinging redis", "error", err)
		return err;
	}

	return nil;
}

func Redis() *redis.Client {
	return redisClient
}