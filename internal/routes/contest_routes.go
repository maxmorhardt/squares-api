package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func RegisterContestRoutes(rg *gin.RouterGroup, h handler.ContestHandler, verifier middleware.TokenVerifier) {
	rg.GET("/owner/:owner", middleware.AuthMiddleware(verifier), h.GetContestsByOwner)

	rg.PUT("", middleware.AuthMiddleware(verifier), h.CreateContest)
	rg.PATCH("/:id", middleware.AuthMiddleware(verifier), h.UpdateContest)
	rg.POST("/:id/start", middleware.AuthMiddleware(verifier), h.StartContest)
	rg.POST("/:id/quarter-result", middleware.AuthMiddleware(verifier), h.RecordQuarterResult)
	rg.DELETE("/:id", middleware.AuthMiddleware(verifier), h.DeleteContest)

	rg.PATCH("/:id/squares/:squareId", middleware.AuthMiddleware(verifier), h.UpdateSquare)
	rg.POST("/:id/squares/clear-mine", middleware.AuthMiddleware(verifier), h.ClearMySquares)
	rg.POST("/:id/squares/:squareId/clear", middleware.AuthMiddleware(verifier), h.ClearSquare)
}
