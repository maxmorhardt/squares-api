package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterWebSocketRoutes(rg *gin.RouterGroup) {
	rg.GET("/contests/:contestId", middleware.AuthMiddlewareWS(), handler.WebSocketHandler)
}
