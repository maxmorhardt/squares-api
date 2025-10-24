package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimitMiddleware() gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted("50-M")
	if err != nil {
		slog.Error("failed to create rate limit", "error", err.Error())
		panic(err)
	}

	store, err := sredis.NewStoreWithOptions(config.RedisClient, limiter.StoreOptions{
		Prefix:   "rate_limit:",
		MaxRetry: 3,
	})
	
	if err != nil {
		slog.Error("failed to create redis store for rate limiter", "error", err.Error())
		panic(err)
	}

	return mgin.NewMiddleware(limiter.New(store, rate))
}