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
	// create rate limit: 150 requests per minute
	rate, err := limiter.NewRateFromFormatted("150-M")
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
		slog.Error("failed to create redis store for rate limiter", "error", err.Error())
		panic(err)
	}

	// return middleware with error and limit reached handlers
	return mgin.NewMiddleware(
		limiter.New(store, rate),
		// handle redis or store errors
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
		// handle when rate limit is exceeded
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

func ContactRateLimitMiddleware() gin.HandlerFunc {
	// create strict rate limit: 3 requests per day per IP
	rate, err := limiter.NewRateFromFormatted("3-D")
	if err != nil {
		slog.Error("failed to create contact rate limit", "error", err.Error())
		panic(err)
	}

	// create redis store for contact rate limit tracking
	store, err := sredis.NewStoreWithOptions(config.RedisClient, limiter.StoreOptions{
		Prefix:   "contact_rate_limit:",
		MaxRetry: 3,
	})

	if err != nil {
		slog.Error("failed to create redis store for contact rate limiter", "error", err.Error())
		panic(err)
	}

	// return middleware with error and limit reached handlers
	return mgin.NewMiddleware(
		limiter.New(store, rate),
		// handle redis or store errors
		mgin.WithErrorHandler(func(c *gin.Context, err error) {
			log := util.LoggerFromGinContext(c)

			log.Warn("contact rate limit error", "error", err)
			c.JSON(http.StatusTooManyRequests, model.NewAPIError(
				http.StatusTooManyRequests,
				"Rate limit exceeded for contact form",
				c,
			))

			c.Abort()
		}),
		// handle when contact rate limit is exceeded
		mgin.WithLimitReachedHandler(func(c *gin.Context) {
			log := util.LoggerFromGinContext(c)

			log.Warn("contact rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, model.NewAPIError(
				http.StatusTooManyRequests,
				"You have reached the maximum number of contact form submissions for today. Please try again tomorrow.",
				c,
			))

			c.Abort()
		}),
	)
}
