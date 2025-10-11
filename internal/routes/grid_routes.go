package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterSquaresRoutes(rg *gin.RouterGroup) {
	rg.POST("", middleware.RoleMiddleware(config.OIDCVerifier), handler.CreateGridHandler)
	rg.GET("", middleware.RoleMiddleware(config.OIDCVerifier, "squares-admin"), handler.GetAllGridsHandler)
	rg.GET("/user/:username", middleware.RoleMiddleware(config.OIDCVerifier), handler.GetGridsByUserHandler)
	rg.GET("/:id", middleware.RoleMiddleware(config.OIDCVerifier), handler.GetGridByIDHandler)
	rg.PATCH("/cell/:id", middleware.RoleMiddleware(config.OIDCVerifier), handler.UpdateGridCellHandler)
}