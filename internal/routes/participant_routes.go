package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterParticipantRoutes(rg *gin.RouterGroup, h handler.ParticipantHandler, verifier middleware.TokenVerifier) {
	rg.GET("", middleware.AuthMiddleware(verifier), h.GetParticipants)
	rg.PATCH("/:userId", middleware.AuthMiddleware(verifier), h.UpdateParticipant)
	rg.DELETE("/:userId", middleware.AuthMiddleware(verifier), h.RemoveParticipant)
}

func RegisterMyContestsRoute(rg *gin.RouterGroup, h handler.ParticipantHandler, verifier middleware.TokenVerifier) {
	rg.GET("", middleware.AuthMiddleware(verifier), h.GetMyContests)
}
