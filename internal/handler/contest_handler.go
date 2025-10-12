package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
)

// @Summary Create a new Contest
// @Description Creates a new 10x10 contest with X and Y labels. Contest name must be 1-20 characters with only letters, numbers, spaces, hyphens, and underscores. Team names are optional but follow the same validation rules.
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
	repo := repository.NewContestRepository()

	var req model.CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind create contest json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	if !isValidNameOrTeam(req.Name) {
		log.Error("invalid contest name", "name", req.Name)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores", c))
		return
	}

	if req.HomeTeam != "" && !isValidNameOrTeam(req.HomeTeam) {
		log.Error("invalid home team name", "homeTeam", req.HomeTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores", c))
		return
	}

	if req.AwayTeam != "" && !isValidNameOrTeam(req.AwayTeam) {
		log.Error("invalid away team name", "awayTeam", req.AwayTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores", c))
		return
	}

	user := c.GetString(model.UserKey)
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)

	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range 10 {
		xLabels[i] = -1
		yLabels[i] = -1
	}

	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	contest := model.Contest{
		Name:     req.Name,
		HomeTeam: req.HomeTeam,
		AwayTeam: req.AwayTeam,
		XLabels:  xLabelsJSON,
		YLabels:  yLabelsJSON,
	}

	if err := repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to create new contest: %s", err), c))
		return
	}

	log.Info("create contest request json bound successfully", "name", req.Name)
	c.JSON(http.StatusOK, contest)
}

func isValidNameOrTeam(name string) bool {
	if len(name) == 0 || len(name) > 20 {
		return false
	}
	matches, _ := regexp.MatchString(`^[A-Za-z0-9\s\-_]{1,20}$`, name)

	return matches
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
	repo := repository.NewContestRepository()

	username := c.Param("username")
	if username == "" {
		log.Error("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Username is required", c))
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
// @Description Returns a single contest by its ID (public endpoint, no authentication required)
// @Tags contests
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Router /contests/{id} [get]
func GetContestByIDHandler(c *gin.Context) {
	log := middleware.FromContext(c)
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

	contest, err := repo.GetByID(ctx, contestID.String())
	if err != nil {
		log.Error("failed to get contest by id", "contest_id", contestID, "error", err)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
		return
	}

	log.Info("contest retrieved successfully", "contest_id", contest.ID)
	c.JSON(http.StatusOK, contest)
}

// @Summary Update a single square in a contest
// @Description Updates the value of a specific square in a contest. Value must be 1-3 uppercase letters or numbers only.
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
// @Router /contests/square/{id} [post]
func UpdateSquareHandler(c *gin.Context) {
	log := middleware.FromContext(c)
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
	if !isValidSquareValue(req.Value) {
		log.Error("invalid square value", "value", req.Value)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Value must be 1-3 uppercase letters or numbers", c))
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
		log.Error("failed to update square", "square_id", squareID, "value", req.Value, "user", user, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to update square: %s", err), c))
		return
	}

	if err := redisService.PublishSquareUpdate(ctx, updatedSquare.ContestID, updatedSquare.ID, updatedSquare.Value, user); err != nil {
		log.Error("failed to publish square update", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID, "error", err)
	}

	log.Info("square updated successfully", "square_id", squareID, "value", req.Value, "user", user)
	c.JSON(http.StatusOK, updatedSquare)
}

func isValidSquareValue(val string) bool {
	if val == "" {
		return true
	}

	if len(val) > 3 {
		return false
	}
	matches, _ := regexp.MatchString(`^[A-Z0-9]{1,3}$`, val)

	return matches
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
	log := middleware.FromContext(c)
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

	updatedContest, err := repo.UpdateLabels(ctx, contestID.String(), xLabels, yLabels, user)
	if err != nil {
		log.Error("failed to update contest labels", "contest_id", contestID, "user", user, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to randomize contest labels: %s", err), c))
		return
	}

	if err := redisService.PublishLabelsUpdate(ctx, updatedContest.ID, xLabels, yLabels, user); err != nil {
		log.Error("failed to publish labels update", "contestId", updatedContest.ID, "user", user, "error", err)
	}

	log.Info("contest labels randomized successfully", "contest_id", contestID, "user", user)
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
