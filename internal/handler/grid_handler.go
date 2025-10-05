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

	grid := initGridData(&req)

	repo := repository.NewGridRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	if err := repo.Create(ctx, &grid); err != nil {
		log.Error("failed to create grid in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("failed to create new grid: %s", err), c))
		return
	}

	log.Info("grid created successfully", "grid_id", grid.ID, "name", grid.Name)
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
	log.Info("get grid by id handler called")

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