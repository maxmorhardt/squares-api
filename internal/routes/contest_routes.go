package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func RegisterContestRoutes(rg *gin.RouterGroup) {
	rg.GET("", middleware.AuthMiddleware(model.SquaresAdminGroup), handler.GetAllContestsHandler)
	rg.PUT("", middleware.AuthMiddleware(), handler.CreateContestHandler)
	rg.PATCH(":id", middleware.AuthMiddleware(), handler.UpdateContestHandler)
	rg.GET("/:id", middleware.AuthMiddleware(), handler.GetContestByIDHandler)
	rg.GET("/user/:username", middleware.AuthMiddleware(), handler.GetContestsByUserHandler)
	rg.PATCH("/square/:id", middleware.AuthMiddleware(), handler.UpdateSquareHandler)
	rg.POST("/:id/randomize-labels", middleware.AuthMiddleware(), handler.RandomizeContestLabelsHandler)
}
