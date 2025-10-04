package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/db"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/datatypes"
)

type CreateGridRequest struct {
	Name    string         `json:"name"`
	Data    [10][10]string `json:"data"`
	XLabels [10]string     `json:"xLabels"`
	YLabels [10]string     `json:"yLabels"`
}

// @Summary Create a new Grid
// @Description Creates a new 10x10 grid with X and Y labels
// @Tags grids
// @Accept json
// @Produce json
// @Param grid body CreateGridRequest true "Grid to create"
// @Success 200 {object} model.GridSwagger
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /grids [post]
func CreateGridHandler(c *gin.Context) {
	var req CreateGridRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid grid data"})
		return
	}

	xLabelsJSON, err := json.Marshal(req.XLabels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid xLabels"})
		return
	}

	yLabelsJSON, err := json.Marshal(req.YLabels)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid yLabels"})
		return
	}

	currentUser := c.GetString("user")

	grid := model.Grid{
		Name:    req.Name,
		Data:    datatypes.JSON(dataJSON),
		XLabels: datatypes.JSON(xLabelsJSON),
		YLabels: datatypes.JSON(yLabelsJSON),
	}

	if err := db.DB.WithContext(context.WithValue(c.Request.Context(), model.ContextUserKey, currentUser)).Create(&grid).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create grid"})
		return
	}

	c.JSON(http.StatusOK, grid)
}