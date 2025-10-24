package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

const rateLimitExceededMessage = "Rate limit exceeded. Please slow down your requests"

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

	return mgin.NewMiddleware(
		limiter.New(store, rate),
		mgin.WithErrorHandler(func(c *gin.Context, err error) {
			log := util.LoggerFromGinContext(c)

			log.Warn("rate limit error", "error", err)
			c.JSON(http.StatusTooManyRequests, model.NewAPIError(
				http.StatusTooManyRequests,
				rateLimitExceededMessage,
				c,
			))

			c.Abort()
		}),
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			log := util.LoggerFromGinContext(c)

			log.Warn("rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, model.NewAPIError(
				http.StatusTooManyRequests,
				rateLimitExceededMessage,
				c,
			))

			c.Abort()
		}),
	)
}