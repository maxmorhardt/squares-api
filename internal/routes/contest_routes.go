package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/handler"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func RegisterContestRoutes(rg *gin.RouterGroup, h handler.ContestHandler, userService service.UserService) {
	rg.GET("/owner/:owner", middleware.AuthMiddleware(userService), h.GetContestsByOwner)

	rg.PUT("", middleware.AuthMiddleware(userService), h.CreateContest)
	rg.PATCH("/:id", middleware.AuthMiddleware(userService), h.UpdateContest)
	rg.POST("/:id/start", middleware.AuthMiddleware(userService), h.StartContest)
	rg.POST("/:id/quarter-result", middleware.AuthMiddleware(userService), h.RecordQuarterResult)
	rg.POST("/:id/quarter-result/rollback", middleware.AuthMiddleware(userService), h.RollbackLastQuarterResult)
	rg.DELETE("/:id", middleware.AuthMiddleware(userService), h.DeleteContest)

	rg.POST("/:id/squares/:squareId/claim", middleware.AuthMiddleware(userService), h.ClaimSquare)
	rg.POST("/:id/squares/clear-mine", middleware.AuthMiddleware(userService), h.ClearMySquares)
	rg.POST("/:id/squares/:squareId/clear", middleware.AuthMiddleware(userService), h.ClearSquare)
}
