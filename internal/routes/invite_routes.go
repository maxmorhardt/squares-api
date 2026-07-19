package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler, userService service.UserService) {
	rg.GET("/:token", h.GetInvitePreview)
	rg.POST("/:token/redeem", middleware.AuthMiddleware(userService), h.RedeemInvite)
}

func RegisterContestInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler, userService service.UserService) {
	rg.POST("", middleware.AuthMiddleware(userService), h.CreateInvite)
	rg.GET("", middleware.AuthMiddleware(userService), h.GetInvites)
	rg.DELETE("/:inviteId", middleware.AuthMiddleware(userService), h.DeleteInvite)
}
