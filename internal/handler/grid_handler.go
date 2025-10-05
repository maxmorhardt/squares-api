package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
)

// @Summary Create a new Grid
// @Description Creates a new 10x10 grid with X and Y labels
// @Tags grids
// @Accept json
// @Produce json
// @Param grid body model.CreateGridRequest true "Grid to create"
// @Success 200 {object} model.GridSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids [post]
func CreateGridHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	var req model.CreateGridRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	grid := initGridData(&req)

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	if err := repo.Create(ctx, &grid); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("failed to create new grid: %s", err), c))
		return
	}

	c.JSON(http.StatusOK, grid)
}

func initGridData(req *model.CreateGridRequest) model.Grid {
	data := make([][]string, 10)
	for i := range data {
		data[i] = make([]string, 10)
	}

	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range int8(10) {
		xLabels[i] = 0
		yLabels[i] = 0
	}

	dataJSON, _ := json.Marshal(data)
	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	return model.Grid{
		Name:    req.Name,
		Data:    dataJSON,
		XLabels: xLabelsJSON,
		YLabels: yLabelsJSON,
	}
}