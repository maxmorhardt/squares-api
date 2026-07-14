package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterGameRoutes(rg *gin.RouterGroup, h handler.GameHandler, verifier middleware.TokenVerifier) {
	rg.GET("/upcoming", middleware.AuthMiddleware(verifier), h.GetUpcoming)
}
