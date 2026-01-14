package config

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	RedisClient      *redis.Client
	IsRedisAvailable = false
)

const (
	poolSize       = 15
	minIdleConns   = 3
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
)

func InitRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		slog.Error("REDIS_HOST environment variable is not set")
		panic("REDIS_HOST environment variable is required")
	}

	RedisClient = newRedisClient(redisHost)
	slog.Info("redis client configured", "pool_size", poolSize, "min_idle_conns", minIdleConns)

	if IsRedisAvailable = pingRedis() == nil; !IsRedisAvailable {
		slog.Warn("initial redis connection failed, will retry in background")
		go retryConnectionWithBackoff()
		return
	}

	slog.Info("redis connection established successfully")
	go healthCheck()
}

func newRedisClient(redisHost string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         redisHost,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
	})
}

func pingRedis() error {
	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		slog.Warn("error while pinging redis", "error", err)
		return err;
	}

	return nil;
}

func retryConnectionWithBackoff() {
	backoff := initialBackoff
	for {
		time.Sleep(backoff)
		if pingRedis() == nil {
			IsRedisAvailable = true
			slog.Info("redis connection re-established successfully")
			go healthCheck()
			return
		}

		IsRedisAvailable = false
		slog.Warn("redis reconnection attempt failed", "retry_in_sec", (backoff*2) / time.Second)

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func healthCheck() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if pingRedis() == nil {
			continue
		}

		IsRedisAvailable = false
		slog.Warn("redis health check failed, starting reconnection attempts")
		go retryConnectionWithBackoff()
		return
	}
}
