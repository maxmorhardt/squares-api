package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ContestHandler interface {
	GetAllContests(c *gin.Context)
	CreateContest(c *gin.Context)
	GetContestByID(c *gin.Context)
	UpdateContest(c *gin.Context)
	RandomizeLabels(c *gin.Context)
	UpdateSquare(c *gin.Context)
	GetContestsByUser(c *gin.Context)
}

type contestHandler struct {
	contestService    service.ContestService
	authService       service.AuthService
	validationService service.ValidationService
}

func NewContestHandler(contestService service.ContestService, authService service.AuthService, validationService service.ValidationService) ContestHandler {
	return &contestHandler{
		contestService: contestService,
		authService:    authService,
		validationService: validationService,
	}
}

// @Summary Get all contests
// @Description Returns all contests
// @Tags contests
// @Produce json
// @Success 200 {array} model.ContestSwagger
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests [get]
func (h *contestHandler) GetAllContests(c *gin.Context) {
	contests, err := h.contestService.GetAllContests(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get all contests: %s", err), c))
		return
	}

	c.JSON(http.StatusOK, contests)
}

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
func (h *contestHandler) CreateContest(c *gin.Context) {
	log := util.LoggerFromContext(c)

	var req model.CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind create contest json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	if !h.authService.IsDeclaredUser(c.Request.Context(), req.Owner) {
		log.Error("user not authorized to create contest", "user", c.GetString(model.UserKey))
		c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, fmt.Sprintf("Not authorized to create contest for user %s", req.Owner), c))
		return
	}

	user := c.GetString(model.UserKey)
	if err := h.validationService.ValidateNewContest(c.Request.Context(), &req, user); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	contest, err := h.contestService.CreateContest(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest already exists", c))
			return
		}
		
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to create new contest", c))
		return
	}

	c.JSON(http.StatusOK, contest)
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
func (h *contestHandler) GetContestByID(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

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

	contest, err := h.contestService.GetContestByID(c.Request.Context(), contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
			return
		}

		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get contest by id %s", contestID), c))
		return
	}

	c.JSON(http.StatusOK, contest)
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
func (h *contestHandler) UpdateContest(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, nil)
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
func (h *contestHandler) RandomizeLabels(c *gin.Context) {
	log := util.LoggerFromContext(c)

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

	updatedContest, err := h.contestService.RandomizeLabels(c.Request.Context(), contestID, c.GetString(model.UserKey))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
			return
		}

		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to update labels for contest: %s", contestID), c))
		return
	}

	c.JSON(http.StatusOK, updatedContest)
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
func (h *contestHandler) UpdateSquare(c *gin.Context) {
	log := util.LoggerFromContext(c)

	squareIDParam := c.Param("id")
	if squareIDParam == "" {
		log.Error("square id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Square ID is required", c))
		return
	}

	squareID, err := uuid.Parse(squareIDParam)
	if err != nil {
		log.Error("invalid square id", "param", squareIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid square id: %s", squareIDParam), c))
		return
	}

	var req model.UpdateSquareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid square update request", c))
		return
	}

	req.Value = strings.ToUpper(req.Value)
	if err := h.validationService.ValidateSquareUpdate(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, err.Error(), c))
		return
	}

	updatedSquare, err := h.contestService.UpdateSquare(c.Request.Context(), squareID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to update square", c))
		return
	}

	c.JSON(http.StatusOK, updatedSquare)
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
func (h *contestHandler) GetContestsByUser(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	username := c.Param("username")
	if username == "" {
		log.Error("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid username", c))
		return
	}

	contests, err := h.contestService.GetContestsByUser(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get contests for user %s", username), c))
		return
	}

	c.JSON(http.StatusOK, contests)
}
