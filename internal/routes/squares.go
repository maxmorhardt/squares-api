package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/pkg/auth"
)

func RegisterSquaresRoutes(rg *gin.RouterGroup) {
	rg.GET("/", auth.RoleMiddleware(config.OIDCVerifier), handler.Test)
}
