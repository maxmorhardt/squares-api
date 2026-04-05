package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
)

func RegisterStatsRoutes(rg *gin.RouterGroup, h handler.StatsHandler) {
	rg.GET("", h.GetStats)
}
