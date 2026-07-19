package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterWebSocketRoutes(rg *gin.RouterGroup, h handler.WebSocketHandler, userService service.UserService) {
	rg.GET("/contests/:id", middleware.AuthMiddlewareWS(userService), h.ContestWSConnection)
}
