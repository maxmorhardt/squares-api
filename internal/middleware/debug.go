package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/util"
)

func DebugMiddleware(c *gin.Context) {
	if c.Request.URL.Path == "/health" || strings.HasPrefix(c.Request.URL.Path, "/ws") {
		c.Next()
		return
	}

	log := util.LoggerFromGinContext(c)

	// build header attributes for logging
	headerAttrs := []any{"method", c.Request.Method}
	for name, values := range c.Request.Header {
		for _, value := range values {
			headerAttrs = append(headerAttrs, name, value)
		}
	}

	log.Info("Request headers for path " + c.Request.URL.Path, headerAttrs...)

	c.Next()
}
