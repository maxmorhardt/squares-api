package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRootRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", handler.HealthCheck)
	rg.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
}
