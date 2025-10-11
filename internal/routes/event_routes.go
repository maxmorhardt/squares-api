package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
)

func RegisterSSERoutes(rg *gin.RouterGroup) {
	rg.GET("/", handler.SSEHandler)
}