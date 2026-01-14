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

const (
	defaultRateLimit = "20-S"
	contactRateLimit = "10-D"
)

const (
	rateLimitExceededMessage        = "Rate limit exceeded. Please slow down your requests"
	contactRateLimitExceededMessage = "You have reached the maximum number of contact form submissions for today. Please try again tomorrow"
)

func RateLimitMiddleware() gin.HandlerFunc {
	return createRateLimiter(defaultRateLimit, rateLimitExceededMessage)
}

func ContactRateLimitMiddleware() gin.HandlerFunc {
	return createRateLimiter(contactRateLimit, contactRateLimitExceededMessage)
}

func createRateLimiter(rateLimit string, errorMessage string) gin.HandlerFunc {
	// if redis isnt available continue
	if !config.IsRedisAvailable || config.RedisClient == nil {
		slog.Warn("redis not available, skipping rate limiting")
		return func(c *gin.Context) { 
			c.Next() 
		}
	}

	// create rate limit
	rate, err := limiter.NewRateFromFormatted(rateLimit)
	if err != nil {
		slog.Error("failed to create rate limit", "error", err.Error())
		panic(err)
	}

	// create redis store for rate limit tracking
	store, err := sredis.NewStoreWithOptions(config.RedisClient, limiter.StoreOptions{
		Prefix:   "rate_limit:",
		MaxRetry: 3,
	})

	if err != nil {
		slog.Warn("failed to create redis store for rate limiter, skipping rate limiting", "error", err.Error())
		return func(c *gin.Context) { 
			c.Next() 
		}
	}

	// return middleware with error and limit reached handlers
	return mgin.NewMiddleware(
		limiter.New(store, rate),
		// allow requests through if redis or any error occurs
		// still have nginx rate limiting
		mgin.WithErrorHandler(func(c *gin.Context, err error) {
			log := util.LoggerFromGinContext(c)
			log.Warn("rate limit error, allowing request through", "error", err)
			c.Next()
		}),
		// handle when rate limit is exceeded
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			log := util.LoggerFromGinContext(c)

			log.Warn("rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, model.NewAPIError(
				http.StatusTooManyRequests,
				errorMessage,
				c,
			))

			c.Abort()
		}),
	)
}
