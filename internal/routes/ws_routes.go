package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
)

func RegisterWebSocketRoutes(rg *gin.RouterGroup) {
	rg.GET("/contests/:contestId", handler.WebSocketHandler)
}
