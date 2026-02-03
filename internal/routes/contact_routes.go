package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
)

func RegisterContactRoute(rg *gin.RouterGroup, h handler.ContactHandler) {
	rg.POST("", h.SubmitContact)
}