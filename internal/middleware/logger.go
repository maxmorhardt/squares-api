package middleware

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)
}

func LoggerMiddleware(c *gin.Context) {
	requestId := c.GetHeader("X-Request-ID")
	if requestId == "" {
		requestId = uuid.New().String()
	}

	logger := slog.Default().With(
		"request_id", requestId,
		"client_ip", c.ClientIP(),
	)

	c.Set(model.RequestIDKey, requestId)
	c.Set(model.LoggerKey, logger)

	c.Next()
}