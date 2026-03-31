package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterWebSocketRoutes(rg *gin.RouterGroup, h handler.WebSocketHandler) {
	rg.GET("/contests/owner/:owner/name/:name", middleware.AuthMiddlewareWS(), h.ContestWSConnection)
}
