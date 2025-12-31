package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/util"
)

func DebugMiddleware(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// log all request headers
	log.Info("Request Headers",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	for name, values := range c.Request.Header {
		for _, value := range values {
			log.Info("Header",
				"name", name,
				"value", value,
			)
		}
	}

	c.Next()
}
