package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterLeaderboardRoutes(rg *gin.RouterGroup, h handler.LeaderboardHandler, userService service.UserService) {
	rg.GET("", h.GetLeaderboard)
	rg.GET("/me", middleware.AuthMiddleware(userService), h.GetMyRank)
}
