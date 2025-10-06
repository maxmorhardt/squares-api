package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		log.Error("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	log.Info("create grid request json bound successfully", "name", req.Name)

	// Initialize XLabels and YLabels arrays with -1
	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := 0; i < 10; i++ {
		xLabels[i] = -1
		yLabels[i] = -1
	}

	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	grid := model.Grid{
		Name:    req.Name,
		XLabels: xLabelsJSON,
		YLabels: yLabelsJSON,
	}

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	if err := repo.Create(ctx, &grid); err != nil {
		log.Error("failed to create grid in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("failed to create new grid: %s", err), c))
		return
	}

	log.Info("grid and cells created successfully", "grid_id", grid.ID, "name", grid.Name)
	c.JSON(http.StatusOK, grid)
}

// @Summary Get all grids
// @Description Returns all grids
// @Tags grids
// @Produce json
// @Success 200 {array} model.GridSwagger
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids [get]
func GetAllGridsHandler(c *gin.Context) {
	log := middleware.FromContext(c)
	log.Info("get all grids handler called")

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grids, err := repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get all grids", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "failed to get all grids", c))
		return
	}

	log.Info("retrieved all grids", "count", len(grids))
	c.JSON(http.StatusOK, grids)
}

// @Summary Get all grids by username
// @Description Returns all grids created by a specific user
// @Tags grids
// @Produce json
// @Param username path string true "Username"
// @Success 200 {array} model.GridSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids/user/{username} [get]
func GetGridsByUserHandler(c *gin.Context) {
	log := middleware.FromContext(c)
	log.Info("get grids by user handler called")

	username := c.Param("username")
	if username == "" {
		log.Error("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "username is required", c))
		return
	}

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grids, err := repo.GetAllByUser(ctx, username)
	if err != nil {
		log.Error("failed to get grids by user", "username", username, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "failed to get grids by user", c))
		return
	}

	log.Info("retrieved grids by user", "username", username, "count", len(grids))
	c.JSON(http.StatusOK, grids)
}

// @Summary Get a grid by ID
// @Description Returns a single grid by ID
// @Tags grids
// @Produce json
// @Param id path string true "Grid ID"
// @Success 200 {object} model.GridSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids/{id} [get]
func GetGridByIDHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	gridID := c.Param("id")
	if gridID == "" {
		log.Error("grid id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "grid id is required", c))
		return
	}

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grid, err := repo.GetByID(ctx, gridID)
	if err != nil {
		log.Error("failed to get grid by id", "id", gridID, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("failed to get grid: %s", err), c))
		return
	}

	if grid.ID == uuid.Nil {
		log.Error("grid not found", "id", gridID)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "grid not found", c))
		return
	}

	log.Info("grid retrieved successfully", "grid_id", grid.ID, "name", grid.Name)
	c.JSON(http.StatusOK, grid)
}

// @Summary Update a single cell in a grid
// @Description Updates the value of a specific cell in a grid
// @Tags grids
// @Accept json
// @Produce json
// @Param id path string true "Grid ID"
// @Param cell body model.UpdateGridCellRequest true "Cell update data"
// @Success 200 {object} model.GridCell
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids/{id}/cell [patch]
func UpdateGridCellHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	gridIDStr := c.Param("id")
	if gridIDStr == "" {
		log.Error("grid id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "grid id is required", c))
		return
	}

	var req model.UpdateGridCellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	gridID, err := uuid.Parse(gridIDStr)
	if err != nil {
		log.Error("invalid grid id", "id", gridIDStr, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "invalid grid id", c))
		return
	}

	repo := repository.NewGridRepository()
	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	if err := repo.UpdateCell(ctx, gridID, req.Row, req.Col, req.Value, user); err != nil {
		log.Error("failed to update grid cell", "grid_id", gridID, "row", req.Row, "col", req.Col, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("failed to update grid cell: %s", err), c))
		return
	}

	log.Info("grid cell updated successfully", "grid_id", gridID, "row", req.Row, "col", req.Col)
	c.JSON(http.StatusOK, gin.H{
		"gridID": gridID,
		"row":    req.Row,
		"col":    req.Col,
		"value":  req.Value,
	})
}