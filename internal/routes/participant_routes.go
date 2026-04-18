package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterParticipantRoutes(rg *gin.RouterGroup, h handler.ParticipantHandler) {
	// /contests/:id/participants
	rg.GET("", middleware.AuthMiddleware(), h.GetParticipants)
	rg.PATCH("/:userId", middleware.AuthMiddleware(), h.UpdateParticipant)
	rg.DELETE("/:userId", middleware.AuthMiddleware(), h.RemoveParticipant)
}

func RegisterMyContestsRoute(rg *gin.RouterGroup, h handler.ParticipantHandler) {
	// /contests/me
	rg.GET("", middleware.AuthMiddleware(), h.GetMyContests)
}
