package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterContactRoute(rg *gin.RouterGroup, h handler.ContactHandler) {
	rg.POST("", middleware.ContactRateLimitMiddleware(), h.SubmitContact)
}
