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
	"github.com/maxmorhardt/squares-api/internal/service"
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
		log.Error("failed to bind create grid json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	log.Info("create grid request json bound successfully", "name", req.Name)

	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range 10 {
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
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to create new grid: %s", err), c))
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

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grids, err := repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get all grids", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get all grids: %s", err), c))
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Username is required", c))
		return
	}

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grids, err := repo.GetAllByUser(ctx, username)
	if err != nil {
		log.Error("failed to get grids by user", "username", username, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get grids for user %s: %s", username, err), c))
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Grid id is required", c))
		return
	}

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	grid, err := repo.GetByID(ctx, gridID)
	if err != nil {
		log.Error("failed to get grid by id", "id", gridID, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get grid %s: %s", gridID, err), c))
		return
	}

	if grid.ID == uuid.Nil {
		log.Error("grid not found", "id", gridID)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Grid not found", c))
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
// @Param id path string true "Cell ID"
// @Param cell body model.UpdateGridCellRequest true "Cell update data"
// @Success 200 {object} model.GridCell
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /grids/cell/{id} [patch]
func UpdateGridCellHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	cellIDParam := c.Param("id")
	if cellIDParam == "" {
		log.Error("cell id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Cell id is required", c))
		return
	}

	var req model.UpdateGridCellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	cellID, err := uuid.Parse(cellIDParam)
	if err != nil {
		log.Error("invalid cell id", "param", cellIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid cell id: %s", cellIDParam), c))
		return
	}

	repo := repository.NewGridRepository()
	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	updatedCell, err := repo.UpdateCell(ctx, cellID, req.Value, user)
	if err != nil {
		log.Error("failed to update grid cell", "cell_id", cellID, "value", req.Value, "user", user, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to update grid cell: %s", err), c))
		return
	}

	eventSvc := &service.EventService{}
	if err := eventSvc.PublishCellUpdate(ctx, updatedCell.GridID, updatedCell.ID, updatedCell.Value, user); err != nil {
		log.Error("failed to publish cell update", "gridId", updatedCell.GridID, "cellId", updatedCell.ID, "error", err)
	} else {
		log.Info("cell update published successfully", "gridId", updatedCell.GridID, "cellId", updatedCell.ID)
	}

	log.Info("grid cell updated successfully", "cell_id", cellID, "value", req.Value, "user", user)
	c.JSON(http.StatusOK, updatedCell)
}