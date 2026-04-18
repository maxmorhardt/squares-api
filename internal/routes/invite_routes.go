package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler) {
	// public - no auth required for preview
	rg.GET("/:token", h.GetInvitePreview)

	// authenticated - redeem invite
	rg.POST("/:token/redeem", middleware.AuthMiddleware(), h.RedeemInvite)
}

func RegisterContestInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler) {
	// owner-only invite management (under /contests/:id/invites)
	rg.POST("", middleware.AuthMiddleware(), h.CreateInvite)
	rg.GET("", middleware.AuthMiddleware(), h.GetInvites)
	rg.DELETE("/:inviteId", middleware.AuthMiddleware(), h.DeleteInvite)
}
