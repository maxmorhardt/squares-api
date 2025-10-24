package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func RegisterContestRoutes(rg *gin.RouterGroup, h handler.ContestHandler) {
	rg.GET("", middleware.AuthMiddleware(model.SquaresAdminGroup), h.GetAllContests)
	rg.PUT("", middleware.AuthMiddleware(), h.CreateContest)
	rg.GET("/:id", h.GetContestByID)
	rg.PATCH(":id", middleware.AuthMiddleware(), h.UpdateContest)
	rg.DELETE("/:id", middleware.AuthMiddleware(), h.DeleteContest)
	rg.POST("/:id/randomize-labels", middleware.AuthMiddleware(), h.RandomizeLabels)
	rg.PATCH("/square/:id", middleware.AuthMiddleware(), h.UpdateSquare)
	rg.GET("/user/:username", middleware.AuthMiddleware(), h.GetContestsByUser)
}
