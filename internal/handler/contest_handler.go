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

// @Summary Create a new Contest
// @Description Creates a new 10x10 contest with X and Y labels
// @Tags contests
// @Accept json
// @Produce json
// @Param contest body model.CreateContestRequest true "Contest to create"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests [post]
func CreateContestHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	var req model.CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind create contest json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	log.Info("create contest request json bound successfully", "name", req.Name)

	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range 10 {
		xLabels[i] = -1
		yLabels[i] = -1
	}

	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	contest := model.Contest{
		Name:    req.Name,
		XLabels: xLabelsJSON,
		YLabels: yLabelsJSON,
	}

	repo := repository.NewContestRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	if err := repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to create new contest: %s", err), c))
		return
	}

	log.Info("contest and squares created successfully", "contest_id", contest.ID, "name", contest.Name)
	c.JSON(http.StatusOK, contest)
}

// @Summary Get all contests
// @Description Returns all contests
// @Tags contests
// @Produce json
// @Success 200 {array} model.ContestSwagger
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests [get]
func GetAllContestsHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	repo := repository.NewContestRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	contests, err := repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get all contests", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get all contests: %s", err), c))
		return
	}

	log.Info("retrieved all contests", "count", len(contests))
	c.JSON(http.StatusOK, contests)
}

// @Summary Get all contests by username
// @Description Returns all contests created by a specific user
// @Tags contests
// @Produce json
// @Param username path string true "Username"
// @Success 200 {array} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/user/{username} [get]
func GetContestsByUserHandler(c *gin.Context) {
	log := middleware.FromContext(c)
	log.Info("get contests by user handler called")

	username := c.Param("username")
	if username == "" {
		log.Error("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Username is required", c))
		return
	}

	repo := repository.NewContestRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	contests, err := repo.GetAllByUser(ctx, username)
	if err != nil {
		log.Error("failed to get contests by user", "username", username, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get contests for user %s: %s", username, err), c))
		return
	}

	log.Info("retrieved contests by user", "username", username, "count", len(contests))
	c.JSON(http.StatusOK, contests)
}

// @Summary Get a contest by ID
// @Description Returns a single contest by ID
// @Tags contests
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id} [get]
func GetContestByIDHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	contestID := c.Param("id")
	if contestID == "" {
		log.Error("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest id is required", c))
		return
	}

	repo := repository.NewContestRepository()
	ctx := context.WithValue(c.Request.Context(), model.UserKey, c.GetString(model.UserKey))

	contest, err := repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest by id", "id", contestID, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get contest %s: %s", contestID, err), c))
		return
	}

	if contest.ID == uuid.Nil {
		log.Error("contest not found", "id", contestID)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
		return
	}

	log.Info("contest retrieved successfully", "contest_id", contest.ID, "name", contest.Name)
	c.JSON(http.StatusOK, contest)
}

// @Summary Update a single square in a contest
// @Description Updates the value of a specific square in a contest
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Square ID"
// @Param square body model.UpdateSquareRequest true "Square update data"
// @Success 200 {object} model.Square
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/square/{id} [patch]
func UpdateSquareHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	squareIDParam := c.Param("id")
	if squareIDParam == "" {
		log.Error("square id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Square id is required", c))
		return
	}

	var req model.UpdateSquareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	squareID, err := uuid.Parse(squareIDParam)
	if err != nil {
		log.Error("invalid square id", "param", squareIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid square id: %s", squareIDParam), c))
		return
	}

	repo := repository.NewContestRepository()
	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	updatedSquare, err := repo.UpdateSquare(ctx, squareID, req.Value, user)
	if err != nil {
		log.Error("failed to update square", "square_id", squareID, "value", req.Value, "user", user, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to update square: %s", err), c))
		return
	}

	redisSvc := service.NewRedisService()
	if err := redisSvc.PublishSquareUpdate(ctx, updatedSquare.ContestID, updatedSquare.ID, updatedSquare.Value, user); err != nil {
		log.Error("failed to publish square update", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID, "error", err)
	} else {
		log.Info("square update published successfully", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID)
	}

	log.Info("square updated successfully", "square_id", squareID, "value", req.Value, "user", user)
	c.JSON(http.StatusOK, updatedSquare)
}
