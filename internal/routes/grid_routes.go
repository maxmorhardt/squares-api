package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterSquaresRoutes(rg *gin.RouterGroup) {
	rg.POST("", middleware.RoleMiddleware(config.OIDCVerifier), handler.CreateGridHandler)
}