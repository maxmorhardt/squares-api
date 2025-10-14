package util

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func LoggerFromContext(c *gin.Context) *slog.Logger {
	if requestLogger, ok := c.Get(model.LoggerKey); ok {
		return requestLogger.(*slog.Logger)
	}

	return slog.Default()
}

func ClaimsFromContext(c *gin.Context) *model.Claims {
	if claims, ok := c.Get(model.ClaimsKey); ok {
		return claims.(*model.Claims)
	}
	
	return nil
}