package handler

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
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

const (
	alphanumericWithSpacesAndSymbols1To20 = `^[A-Za-z0-9\s\-_]{1,20}$`
	upperAlphanumeric1To3                 = `^[A-Z0-9]{1,3}$`
)

type ContestHandler interface {
	GetContestByID(c *gin.Context)
	GetContestsByUser(c *gin.Context)
	GetParticipatingContests(c *gin.Context)

	CreateContest(c *gin.Context)
	UpdateContest(c *gin.Context)
	DeleteContest(c *gin.Context)
	StartContest(c *gin.Context)
	RecordQuarterResult(c *gin.Context)

	GenerateInviteLink(c *gin.Context)

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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
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

	// if user is authenticated, add them as a participant
	user := c.GetString(model.UserKey)
	if user != "" && user != contest.Owner {
		// check if there's an invite token to determine square limit
		squareLimit := 0 // default: unlimited
		if inviteToken := c.Query("invite"); inviteToken != "" {
			// validate and extract square limit from token
			if claims, err := h.authService.ValidateInviteToken(inviteToken); err == nil {
				if claims.ContestID == contestID {
					squareLimit = claims.SquareLimit
					log.Info("using invite token square limit", "contest_id", contestID, "user", user, "square_limit", squareLimit)
				} else {
					log.Warn("invite token contest mismatch", "token_contest", claims.ContestID, "actual_contest", contestID)
				}
			} else {
				log.Warn("invalid invite token", "error", err)
			}
		}

		// add participant in background with square limit from token (or unlimited), don't fail the request if it errors
		go func() {
			if err := h.contestService.AddParticipant(c.Request.Context(), contestID, user, squareLimit); err != nil {
				log.Error("failed to add participant on contest view", "contest_id", contestID, "user", user, "error", err)
			}
		}()
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
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to get Contests for user %s", username), c))
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

// GetParticipatingContests retrieves contests the user is participating in
// @Summary Get participating contests
// @Description Retrieves a paginated list of contests the authenticated user is participating in
// @Tags contests
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Page size (default: 10)"
// @Success 200 {object} model.PaginatedContestResponseSwagger
// @Failure 400 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/participating [get]
func (h *contestHandler) GetParticipatingContests(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// get authenticated user
	user := c.GetString(model.UserKey)
	if user == "" {
		log.Error("user not found in context")
		c.JSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Unauthorized", c))
		return
	}

	// parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	// get participating contests
	contests, total, err := h.contestService.GetParticipatingContestsPaginated(c.Request.Context(), user, page, limit)
	if err != nil {
		log.Error("failed to get participating contests", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to retrieve participating contests", c))
		return
	}

	// build response
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

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

	// get authenticated user
	user := c.GetString(model.UserKey)

	// validate input
	if !isValidContestName(req.Name) {
		log.Warn("invalid contest name", "name", req.Name)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidContestName), c))
		return
	}

	if !isValidTeamName(req.HomeTeam) {
		log.Warn("invalid home team name", "home_team", req.HomeTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidHomeTeamName), c))
		return
	}

	if !isValidTeamName(req.AwayTeam) {
		log.Warn("invalid away team name", "away_team", req.AwayTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidAwayTeamName), c))
		return
	}

	if req.Owner != user {
		log.Warn("user not authorized to create contest", "user", user, "owner", req.Owner)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("User %s is not authorized to create contest for %s", user, req.Owner), c))
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

func isValidContestName(name string) bool {
	// check length
	if len(name) == 0 || len(name) > 20 {
		return false
	}

	// alphanumeric, spaces, hyphens, underscores only
	matches, _ := regexp.MatchString(alphanumericWithSpacesAndSymbols1To20, name)
	return matches
}

func isValidTeamName(name string) bool {
	// empty team names are allowed
	if len(name) == 0 {
		return true
	}

	return isValidContestName(name)
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
		return
	}

	// parse request body
	var req model.UpdateContestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// get authenticated user
	user := c.GetString(model.UserKey)

	// validate input
	if req.Name != nil && !isValidContestName(*req.Name) {
		log.Warn("invalid contest name in update", "name", *req.Name)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidContestName), c))
		return
	}

	if req.HomeTeam != nil && !isValidTeamName(*req.HomeTeam) {
		log.Warn("invalid home team name in update", "home_team", *req.HomeTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidHomeTeamName), c))
		return
	}

	if req.AwayTeam != nil && !isValidTeamName(*req.AwayTeam) {
		log.Warn("invalid away team name in update", "away_team", *req.AwayTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidAwayTeamName), c))
		return
	}

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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
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
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete Contest %s", contestID), c))
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
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
// Invite & Participant Management
// ====================

// GenerateInviteLink generates a shareable invite link for a contest
// @Summary Generate invite link
// @Description Generates a shareable invite link with a specified square limit
// @Tags contests
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param request body model.GenerateInviteLinkRequest true "Invite link parameters"
// @Success 200 {object} model.GenerateInviteLinkResponse
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/invite [post]
func (h *contestHandler) GenerateInviteLink(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse contest id
	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	// parse request body
	var req model.GenerateInviteLinkRequest
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

	// generate invite token
	token, err := h.contestService.GenerateInviteLink(c.Request.Context(), contestID, req.SquareLimit, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found", "contest_id", contestID)
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
			return
		}
		if errors.Is(err, errs.ErrUnauthorizedContestEdit) {
			log.Warn("user not authorized to generate invite", "contest_id", contestID, "user", user)
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, "Only contest owner can generate invite links", c))
			return
		}

		log.Error("failed to generate invite link", "contest_id", contestID, "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to generate invite link", c))
		return
	}

	response := model.GenerateInviteLinkResponse{
		Token: token,
	}

	c.JSON(http.StatusOK, response)
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Square ID: %s", squareIDParam), c))
		return
	}

	// parse request body
	var req model.UpdateSquareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// get authenticated user and normalize value
	user := c.GetString(model.UserKey)
	req.Value = strings.ToUpper(req.Value)

	// validate input
	if !isValidSquareValue(req.Value) {
		log.Warn("invalid square value", "value", req.Value)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidSquareValue), c))
		return
	}

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

func isValidSquareValue(val string) bool {
	// check length (max 3 characters)
	if len(val) == 0 || len(val) > 3 {
		return false
	}

	// uppercase alphanumeric only
	matches, _ := regexp.MatchString(upperAlphanumeric1To3, val)
	return matches
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Contest ID: %s", contestIDParam), c))
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
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, fmt.Sprintf("Invalid Square ID: %s", squareIDParam), c))
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
