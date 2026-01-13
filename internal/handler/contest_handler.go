package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ContestHandler interface {
	GetContestByID(c *gin.Context)
	GetContestsByUser(c *gin.Context)

	CreateContest(c *gin.Context)
	UpdateContest(c *gin.Context)
	DeleteContest(c *gin.Context)
	StartContest(c *gin.Context)
	RecordQuarterResult(c *gin.Context)

	UpdateSquare(c *gin.Context)
	ClearSquare(c *gin.Context)
}

type contestHandler struct {
	contestService service.ContestService
	authService    service.AuthService
}

func NewContestHandler(contestService service.ContestService, authService service.AuthService) ContestHandler {
	return &contestHandler{
		contestService: contestService,
		authService:    authService,
	}
}

// ====================
// Getters
// ====================

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

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// get contest from service
	contest, err := h.contestService.GetContestByID(c.Request.Context(), contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get contest", c))
		}
		return
	}

	c.JSON(http.StatusOK, contest)
}

// @Summary Get all contests by username
// @Description Returns all contests created by a specific user with pagination (required)
// @Tags contests
// @Produce json
// @Param username path string true "Username"
// @Param page query int true "Page number" minimum(1)
// @Param limit query int true "Items per page (max 25)" minimum(1) maximum(25)
// @Success 200 {object} model.PaginatedContestResponseSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/user/{username} [get]
func (h *contestHandler) GetContestsByUser(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse username from path
	username := c.Param("username")
	if username == "" {
		log.Warn("username not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid username", c))
		return
	}

	// extract pagination parameters
	page, limit, err := h.extractPaginationParams(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		return
	}

	// get paginated contests from service
	contests, total, err := h.contestService.GetContestsByUserPaginated(c.Request.Context(), username, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to retrieve contests", c))
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	response := model.PaginatedContestResponse{
		Contests:    contests,
		Page:        page,
		Limit:       limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}
	c.JSON(http.StatusOK, response)
}

func (h *contestHandler) extractPaginationParams(c *gin.Context) (int, int, error) {
	// get page parameter
	pageStr := c.Query("page")
	if pageStr == "" {
		return 0, 0, errs.ErrInvalidPage
	}

	// get limit parameter
	limitStr := c.Query("limit")
	if limitStr == "" {
		return 0, 0, errs.ErrInvalidLimit
	}

	// parse and validate page and limit
	var page, limit int
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	} else {
		return 0, 0, errs.ErrInvalidPage
	}

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 25 {
		limit = l
	} else {
		return 0, 0, errs.ErrInvalidLimit
	}

	return page, limit, nil
}

// ====================
// Contest Lifecycle Actions
// ====================

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
	log := util.LoggerFromGinContext(c)

	// parse request body
	var req model.CreateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind create contest json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// sanitize inputs
	req.Name = util.SanitizeInput(req.Name)
	req.HomeTeam = util.SanitizeInput(req.HomeTeam)
	req.AwayTeam = util.SanitizeInput(req.AwayTeam)
	req.Owner = util.SanitizeInput(req.Owner)

	// get authenticated user
	user := c.GetString(model.UserKey)

	if req.Owner != user {
		log.Warn("user not authorized to create contest", "user", user, "owner", req.Owner)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "User not authorized to create contest for specified owner", c))
		return
	}

	// create contest via service
	contest, err := h.contestService.CreateContest(c.Request.Context(), &req, user)
	if err != nil {
		if errors.Is(err, errs.ErrDatabaseUnavailable) {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrContestAlreadyExists) {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to create contest", c))
		}
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
// @Param contest body model.UpdateContestRequest true "Contest update data"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id} [patch]
func (h *contestHandler) UpdateContest(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// parse request body
	var req model.UpdateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	sanitized := util.SanitizeInput(*req.Name)
	req.Name = &sanitized

	if req.HomeTeam != nil {
		sanitized := util.SanitizeInput(*req.HomeTeam)
		req.HomeTeam = &sanitized
	}

	if req.AwayTeam != nil {
		sanitized := util.SanitizeInput(*req.AwayTeam)
		req.AwayTeam = &sanitized
	}

	// get authenticated user
	user := c.GetString(model.UserKey)

	// update contest via service
	updatedContest, err := h.contestService.UpdateContest(c.Request.Context(), contestID, &req, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		} else if errors.Is(err, errs.ErrUnauthorizedContestEdit) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrDatabaseUnavailable) {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		} else {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		}
		return
	}

	c.JSON(http.StatusOK, updatedContest)
}

// @Summary Delete contest
// @Description Deletes a contest by id
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Success 204 "Contest deleted successfully"
// @Failure 400 {object} model.APIError "Invalid contest id"
// @Failure 403 {object} model.APIError "Forbidden - user is not the owner"
// @Failure 404 {object} model.APIError "Contest not found"
// @Failure 500 {object} model.APIError "Internal server error"
// @Router /contests/{id} [delete]
func (h *contestHandler) DeleteContest(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// get authenticated user and delete contest
	user := c.GetString(model.UserKey)
	if err = h.contestService.DeleteContest(c.Request.Context(), contestID, user); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		} else if errors.Is(err, errs.ErrUnauthorizedContestDelete) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to delete contest", c))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Start contest
// @Description Starts the contest, transitioning from ACTIVE to Q1 and randomizing labels
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {object} model.ContestSwagger
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/start [post]
func (h *contestHandler) StartContest(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// get authenticated user and start contest
	user := c.GetString(model.UserKey)
	contest, err := h.contestService.StartContest(c.Request.Context(), contestID, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		} else {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		}
		return
	}

	c.JSON(http.StatusOK, contest)
}

// RecordQuarterResult records a quarter result for a contest
// @Summary Record quarter result
// @Description Records the score and winner for a specific quarter
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param quarterResult body model.QuarterResultRequest true "Quarter result data"
// @Success 201 {object} model.QuarterResult
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/quarter-result [post]
func (h *contestHandler) RecordQuarterResult(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	// parse request body
	var req model.QuarterResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind request", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// get authenticated user
	user := c.GetString(model.UserKey)
	if user == "" {
		log.Error("user not found in context")
		c.JSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Unauthorized", c))
		return
	}

	// record quarter result
	result, err := h.contestService.RecordQuarterResult(c.Request.Context(), contestID, req.HomeTeamScore, req.AwayTeamScore, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found", "contest_id", contestID)
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
			return
		}

		if errors.Is(err, gorm.ErrInvalidData) {
			log.Warn("invalid quarter data", "error", err)
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid quarter data", c))
			return
		}

		log.Error("failed to record quarter result", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to record quarter result", c))
		return
	}

	c.JSON(http.StatusCreated, result)
}

// ====================
// Square Actions
// ====================

// @Summary Update a single square in a contest
// @Description Updates the value of a specific square in a contest
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param squareId path string true "Square ID"
// @Param square body model.UpdateSquareRequest true "Square"
// @Success 200 {object} model.Square
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/squares/{squareId} [patch]
func (h *contestHandler) UpdateSquare(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// parse square id from path
	squareIDParam := c.Param("squareId")
	if squareIDParam == "" {
		log.Warn("square id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Square ID is required", c))
		return
	}

	squareID, err := uuid.Parse(squareIDParam)
	if err != nil {
		log.Warn("invalid square id", "param", squareIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid square ID format", c))
		return
	}

	// parse request body
	var req model.UpdateSquareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// sanitize and normalize inputs
	req.Value = strings.ToUpper(util.SanitizeInput(req.Value))
	req.Owner = util.SanitizeInput(req.Owner)

	// get authenticated user
	user := c.GetString(model.UserKey)

	// update square via service
	updatedSquare, err := h.contestService.UpdateSquare(c.Request.Context(), contestID, squareID, &req, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrSquareNotFound), c))
		} else if errors.Is(err, errs.ErrSquareNotEditable) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrUnauthorizedSquareEdit) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrSquareLimitReached) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrClaimsNotFound) {
			c.JSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrDatabaseUnavailable) {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		} else {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		}
		return
	}

	c.JSON(http.StatusOK, updatedSquare)
}

// @Summary Clear square value and owner
// @Description Clears a square's value and owner, making it available for anyone to claim
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param squareId path string true "Square ID"
// @Success 200 {object} model.Square
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/squares/{squareId}/clear [post]
func (h *contestHandler) ClearSquare(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id from path
	contestIDParam := c.Param("id")
	if contestIDParam == "" {
		log.Warn("contest id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Warn("invalid contest id", "param", contestIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID format", c))
		return
	}

	// parse square id from path
	squareIDParam := c.Param("squareId")
	if squareIDParam == "" {
		log.Warn("square id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Square ID is required", c))
		return
	}

	squareID, err := uuid.Parse(squareIDParam)
	if err != nil {
		log.Warn("invalid square id", "param", squareIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid square ID format", c))
		return
	}

	// get authenticated user and clear square
	user := c.GetString(model.UserKey)
	clearedSquare, err := h.contestService.ClearSquare(c.Request.Context(), contestID, squareID, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrSquareNotFound), c))
		} else if errors.Is(err, errs.ErrSquareNotEditable) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrUnauthorizedSquareEdit) {
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		} else if errors.Is(err, errs.ErrDatabaseUnavailable) {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		} else {
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		}
		return
	}

	c.JSON(http.StatusOK, clearedSquare)
}
