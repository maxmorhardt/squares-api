package middleware

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)
}

func LoggerMiddleware(c *gin.Context) {
	// extract or generate request id
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	// create logger with request metadata
	log := slog.Default().With(
		"request_id", requestID,
		"client_ip", c.ClientIP(),
	)

	// store request id and logger in context
	util.SetGinContextValue(c, model.RequestIDKey, requestID)
	util.SetGinContextValue(c, model.LoggerKey, log)

	c.Next()
}
