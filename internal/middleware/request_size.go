package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const (
	maxRequestSize = 1 << 20 // 1 MB in bytes
)

// RequestSizeLimitMiddleware limits the size of incoming request bodies
func RequestSizeLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.LoggerFromGinContext(c)

		// Set max bytes for the request body
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestSize)

		// Try to read beyond limit to trigger error if oversized
		err := c.Request.ParseForm()
		if err != nil {
			log.Warn("request body too large", "error", err)
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, model.NewAPIError(
				http.StatusRequestEntityTooLarge,
				"Request body too large. Maximum size is 1MB",
				c,
			))
			return
		}

		c.Next()
	}
}
