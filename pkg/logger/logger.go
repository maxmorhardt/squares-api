package logger

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

	requestLogger := slog.Default().With("request_id", requestId)
	c.Set("logger", requestLogger)

	if (c.Request.URL.Path == "/health") {
		c.Next()
		return
	}

	requestLogger.Info("request initiated",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	c.Next()
}

func FromContext(c *gin.Context) *slog.Logger {
	if requestLogger, ok := c.Get("logger"); ok {
		return requestLogger.(*slog.Logger)
	}

	return slog.Default()
}