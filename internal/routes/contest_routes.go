package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterContestRoutes(rg *gin.RouterGroup) {
	rg.POST("", middleware.RoleMiddleware(config.OIDCVerifier), handler.CreateContestHandler)
	rg.GET("", middleware.RoleMiddleware(config.OIDCVerifier, "squares-admin"), handler.GetAllContestsHandler)
	rg.GET("/user/:username", middleware.RoleMiddleware(config.OIDCVerifier), handler.GetContestsByUserHandler)
	rg.GET("/:id", middleware.RoleMiddleware(config.OIDCVerifier), handler.GetContestByIDHandler)
	rg.PATCH("/square/:id", middleware.RoleMiddleware(config.OIDCVerifier), handler.UpdateSquareHandler)
}
