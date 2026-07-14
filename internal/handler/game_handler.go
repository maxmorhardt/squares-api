package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type GameHandler interface {
	GetUpcoming(c *gin.Context)
}

type gameHandler struct {
	gameService service.GameService
}

func NewGameHandler(gameService service.GameService) GameHandler {
	return &gameHandler{
		gameService: gameService,
	}
}

// @Summary Get upcoming games
// @Description Returns NFL games available to link a contest to when creating one
// @Tags games
// @Produce json
// @Success 200 {array} model.Game
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /games/upcoming [get]
func (h *gameHandler) GetUpcoming(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	games, err := h.gameService.GetUpcoming(c.Request.Context())
	if err != nil {
		log.Error("failed to get upcoming games", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get upcoming games", c))
		return
	}

	c.JSON(http.StatusOK, games)
}
