package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterParticipantRoutes(rg *gin.RouterGroup, h handler.ParticipantHandler, userService service.UserService) {
	rg.GET("", middleware.AuthMiddleware(userService), h.GetParticipants)
	rg.PATCH("/:userId", middleware.AuthMiddleware(userService), h.UpdateParticipant)
	rg.DELETE("/:userId", middleware.AuthMiddleware(userService), h.RemoveParticipant)
}

func RegisterMyContestsRoute(rg *gin.RouterGroup, h handler.ParticipantHandler, userService service.UserService) {
	rg.GET("", middleware.AuthMiddleware(userService), h.GetMyContests)
}
