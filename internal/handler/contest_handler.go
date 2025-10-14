package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

// @Summary Create a new Contest
// @Description Creates a new 10x10 contest
// @Tags contests
// @Accept json
// @Produce json
// @Param contest body model.CreateContestRequest true "Contest"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests [put]
func CreateContestHandler(c *gin.Context) {
	log := util.LoggerFromContext(c)
	repo := repository.NewContestRepository()

	var req model.CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind create contest json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	user := c.GetString(model.UserKey)
	if !service.ValidateNewContest(c, req, user) {
		return
	}
	
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)
	xLabelsJSON, yLabelsJSON := initLabels()

	contest := model.Contest{
		Name:     req.Name,
		XLabels:  xLabelsJSON,
		YLabels:  yLabelsJSON,
		HomeTeam: req.HomeTeam,
		AwayTeam: req.AwayTeam,
		Owner:    req.Owner,
		Status:   "ACTIVE",
	}

	if err := repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to create new contest: %s", err), c))
		return
	}

	log.Info("created contest", "name", req.Name, "contest_id", contest.ID, "owner", req.Owner)
	c.JSON(http.StatusOK, contest)
}

func initLabels() ([]byte, []byte) {
	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range 10 {
		xLabels[i] = -1
		yLabels[i] = -1
	}
		
	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	return xLabelsJSON, yLabelsJSON
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
	log := util.LoggerFromContext(c)
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

// @Summary Update contest
// @Description Updates the values of a contest
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id} [patch]
func UpdateContestHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, nil)
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
	log := util.LoggerFromContext(c)
	repo := repository.NewContestRepository()

	username := c.Param("username")
	if username == "" || !service.IsDeclaredUser(c, username) {
		log.Error("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid username", c))

		return
	}

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
// @Description Returns a single contest by its ID
// @Tags contests
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Router /contests/{id} [get]
func GetContestByIDHandler(c *gin.Context) {
	log := util.LoggerFromContext(c)
	repo := repository.NewContestRepository()

	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Error("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Error("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid contest id: %s", contestIDParam), c))
		return
	}

	ctx := c.Request.Context()

	contest, err := repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest by id", "contest_id", contestID, "error", err)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
		return
	}

	log.Info("contest retrieved successfully", "contest_id", contest.ID)
	c.JSON(http.StatusOK, contest)
}

// @Summary Update a single square in a contest
// @Description Updates the value of a specific square in a contest
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Square ID"
// @Param square body model.UpdateSquareRequest true "Square"
// @Success 200 {object} model.Square
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/square/{id} [patch]
func UpdateSquareHandler(c *gin.Context) {
	log := util.LoggerFromContext(c)
	repo := repository.NewContestRepository()
	redisService := service.NewRedisService()

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

	req.Value = strings.ToUpper(req.Value)
	if !service.ValidateSquareUpdate(c, req) {
		return
	}

	squareID, err := uuid.Parse(squareIDParam)
	if err != nil {
		log.Error("invalid square id", "param", squareIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid square id: %s", squareIDParam), c))
		return
	}

	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	updatedSquare, err := repo.UpdateSquare(ctx, squareID, req.Value, user)
	if err != nil {
		log.Error("failed to update square", "square_id", squareID, "value", req.Value, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to update square: %s", err), c))
		return
	}

	if err := redisService.PublishSquareUpdate(ctx, updatedSquare.ContestID, user, updatedSquare.ID, updatedSquare.Value); err != nil {
		log.Error("failed to publish square update", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID, "error", err)
	}

	log.Info("square updated successfully", "square_id", squareID, "value", req.Value)
	c.JSON(http.StatusOK, updatedSquare)
}

// @Summary Randomize contest labels
// @Description Randomizes the X and Y labels for a specific contest with numbers 0-9 (no repeats)
// @Tags contests
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/randomize-labels [post]
func RandomizeContestLabelsHandler(c *gin.Context) {
	log := util.LoggerFromContext(c)
	repo := repository.NewContestRepository()
	redisService := service.NewRedisService()

	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Error("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Error("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid contest id: %s", contestIDParam), c))
		return
	}

	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	xLabels := generateRandomizedLabels()
	yLabels := generateRandomizedLabels()

	updatedContest, err := repo.UpdateLabels(ctx, contestID, xLabels, yLabels, user)
	if err != nil {
		log.Error("failed to update contest labels", "contest_id", contestID, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to randomize contest labels: %s", err), c))
		return
	}

	if err := redisService.PublishLabelsUpdate(ctx, updatedContest.ID, user, xLabels, yLabels); err != nil {
		log.Error("failed to publish contest update", "contestId", updatedContest.ID, "error", err)
	}

	log.Info("contest labels randomized successfully", "contest_id", contestID)
	c.JSON(http.StatusOK, updatedContest)
}

func generateRandomizedLabels() []int8 {
	labels := make([]int8, 10)
	for i := range int8(10) {
		labels[i] = i
	}

	for i := len(labels) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		labels[i], labels[j] = labels[j], labels[i]
	}

	return labels
}
