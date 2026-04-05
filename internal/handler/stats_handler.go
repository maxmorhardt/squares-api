package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type StatsHandler interface {
	GetStats(c *gin.Context)
}

type statsHandler struct {
	statsService service.StatsService
}

func NewStatsHandler(statsService service.StatsService) StatsHandler {
	return &statsHandler{
		statsService: statsService,
	}
}

// GetStats godoc
// @Summary Get platform stats
// @Description Returns public stats including contests created today, squares claimed today, and total active contests
// @Tags stats
// @Produce json
// @Success 200 {object} model.StatsResponse
// @Failure 500 {object} model.APIError
// @Router /stats [get]
func (h *statsHandler) GetStats(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	stats, err := h.statsService.GetStats(c.Request.Context())
	if err != nil {
		log.Error("failed to get stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
