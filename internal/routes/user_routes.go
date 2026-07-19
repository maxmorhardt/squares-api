package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterUserRoutes(rg *gin.RouterGroup, h handler.UserHandler, userService service.UserService) {
	auth := middleware.AuthMiddleware(userService)

	rg.GET("", auth, h.GetMe)
	rg.PATCH("", auth, h.UpdateMe)
	rg.DELETE("", auth, h.DeleteMe)
	rg.GET("/stats", auth, h.GetMyStats)
	rg.GET("/active-contests", auth, h.GetMyActiveContests)
}
