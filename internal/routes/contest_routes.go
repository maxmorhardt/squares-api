package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterContestRoutes(rg *gin.RouterGroup, h handler.ContestHandler) {
	rg.GET("/owner/:owner/name/:name", h.GetContestByOwnerAndName)
	rg.GET("/owner/:owner", middleware.AuthMiddleware(), h.GetContestsByOwner)

	rg.PUT("", middleware.AuthMiddleware(), h.CreateContest)
	rg.PATCH("/:id", middleware.AuthMiddleware(), h.UpdateContest)
	rg.POST("/:id/start", middleware.AuthMiddleware(), h.StartContest)
	rg.POST("/:id/quarter-result", middleware.AuthMiddleware(), h.RecordQuarterResult)
	rg.DELETE("/:id", middleware.AuthMiddleware(), h.DeleteContest)

	rg.PATCH("/:id/squares/:squareId", middleware.AuthMiddleware(), h.UpdateSquare)
	rg.POST("/:id/squares/:squareId/clear", middleware.AuthMiddleware(), h.ClearSquare)
}
