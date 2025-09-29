package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/pkg/auth"
)

func RegisterSquaresRoutes(r *gin.Engine) {
	r.GET("/", auth.RoleMiddleware(OIDCVerifier()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
		})
	})
}
