package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterGameRoutes(rg *gin.RouterGroup, h handler.GameHandler, userService service.UserService) {
	rg.GET("/upcoming", middleware.AuthMiddleware(userService), h.GetUpcoming)
}
