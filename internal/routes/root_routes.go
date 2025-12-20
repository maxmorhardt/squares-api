package routes

import (
	"github.com/gin-gonic/gin"
	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func RegisterRootRoutes(rg *gin.RouterGroup, contactHandler handler.ContactHandler) {
	rg.GET("/health", handler.HealthCheck)
	rg.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	rg.POST("/contact", middleware.ContactRateLimitMiddleware(), contactHandler.SubmitContact)
	rg.PATCH("/contact/:id", middleware.AuthMiddleware("squares-admins"), contactHandler.UpdateContactSubmission)
}
