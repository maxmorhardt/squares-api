package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ParticipantHandler interface {
	GetParticipants(c *gin.Context)
	GetMyContests(c *gin.Context)
	UpdateParticipant(c *gin.Context)
	RemoveParticipant(c *gin.Context)
}

type participantHandler struct {
	participantService service.ParticipantService
}

func NewParticipantHandler(participantService service.ParticipantService) ParticipantHandler {
	return &participantHandler{
		participantService: participantService,
	}
}

// @Summary Get all participants for a contest
// @Description Returns all participants and their roles. Any participant can view. Public contests allow any authenticated user
// @Tags participants
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {array} model.ContestParticipant
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/participants [get]
func (h *participantHandler) GetParticipants(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	user := c.GetString(model.UserKey)
	participants, err := h.participantService.GetParticipants(c.Request.Context(), contestID, user)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		case errors.Is(err, errs.ErrNotParticipant), errors.Is(err, errs.ErrInsufficientRole):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(errs.ErrInsufficientRole), c))
		default:
			log.Error("failed to get participants", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get participants", c))
		}
		return
	}

	c.JSON(http.StatusOK, participants)
}

// @Summary Get all contests the user participates in
// @Description Returns all contests where the authenticated user is a participant
// @Tags participants
// @Produce json
// @Success 200 {array} model.ContestSwagger
// @Security BearerAuth
// @Router /contests/me [get]
func (h *participantHandler) GetMyContests(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	user := c.GetString(model.UserKey)
	contests, err := h.participantService.GetMyContests(c.Request.Context(), user)
	if err != nil {
		log.Error("failed to get user contests", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get contests", c))
		return
	}

	c.JSON(http.StatusOK, contests)
}

// @Summary Update a participant's role or square limit
// @Description Owner updates a participant's role or max squares
// @Tags participants
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param userId path string true "Target user ID"
// @Param participant body model.UpdateParticipantRequest true "Update details"
// @Success 200 {object} model.ContestParticipant
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/participants/{userId} [patch]
func (h *participantHandler) UpdateParticipant(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "User ID is required", c))
		return
	}

	var req model.UpdateParticipantRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		log.Warn("failed to bind update participant json", "error", bindErr)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	user := c.GetString(model.UserKey)
	participant, err := h.participantService.UpdateParticipant(c.Request.Context(), contestID, targetUserID, &req, user)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrNotParticipant):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrContestFinalized):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrInsufficientRole), errors.Is(err, errs.ErrCannotChangeOwner):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrSquareLimitTooLow):
			c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to update participant", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to update participant", c))
		}
		return
	}

	c.JSON(http.StatusOK, participant)
}

// @Summary Remove a participant from a contest
// @Description Owner removes a participant and clears their squares
// @Tags participants
// @Param id path string true "Contest ID"
// @Param userId path string true "Target user ID"
// @Success 204
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/participants/{userId} [delete]
func (h *participantHandler) RemoveParticipant(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "User ID is required", c))
		return
	}

	user := c.GetString(model.UserKey)
	if err := h.participantService.RemoveParticipant(c.Request.Context(), contestID, targetUserID, user); err != nil {
		switch {
		case errors.Is(err, errs.ErrNotParticipant):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrContestNotEditable):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrInsufficientRole), errors.Is(err, errs.ErrCannotRemoveOwner):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to remove participant", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to remove participant", c))
		}
		return
	}

	c.Status(http.StatusNoContent)
}
