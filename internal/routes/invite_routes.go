package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler, verifier middleware.TokenVerifier) {
	rg.GET("/:token", h.GetInvitePreview)
	rg.POST("/:token/redeem", middleware.AuthMiddleware(verifier), h.RedeemInvite)
}

func RegisterContestInviteRoutes(rg *gin.RouterGroup, h handler.InviteHandler, verifier middleware.TokenVerifier) {
	rg.POST("", middleware.AuthMiddleware(verifier), h.CreateInvite)
	rg.GET("", middleware.AuthMiddleware(verifier), h.GetInvites)
	rg.DELETE("/:inviteId", middleware.AuthMiddleware(verifier), h.DeleteInvite)
}
