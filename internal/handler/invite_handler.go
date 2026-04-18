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

type InviteHandler interface {
	CreateInvite(c *gin.Context)
	GetInvitePreview(c *gin.Context)
	RedeemInvite(c *gin.Context)
	GetInvites(c *gin.Context)
	DeleteInvite(c *gin.Context)
}

type inviteHandler struct {
	inviteService service.InviteService
}

func NewInviteHandler(inviteService service.InviteService) InviteHandler {
	return &inviteHandler{
		inviteService: inviteService,
	}
}

// @Summary Create an invite link for a contest
// @Description Owner creates an invite link with specified role and square limit
// @Tags invites
// @Accept json
// @Produce json
// @Param id path string true "Contest ID"
// @Param invite body model.CreateInviteRequest true "Invite details"
// @Success 201 {object} model.InviteResponse
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/invites [post]
func (h *inviteHandler) CreateInvite(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	var req model.CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind create invite json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	user := c.GetString(model.UserKey)
	invite, err := h.inviteService.CreateInvite(c.Request.Context(), contestID, &req, user)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		case errors.Is(err, errs.ErrInsufficientRole):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		default:
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to create invite", c))
		}
		return
	}

	c.JSON(http.StatusCreated, model.InviteResponse{
		Token: invite.Token,
	})
}

// @Summary Preview an invite link
// @Description Returns contest name and invite details without authentication
// @Tags invites
// @Produce json
// @Param token path string true "Invite token"
// @Success 200 {object} model.InvitePreviewResponse
// @Failure 404 {object} model.APIError
// @Failure 410 {object} model.APIError
// @Router /invites/{token} [get]
func (h *inviteHandler) GetInvitePreview(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Token is required", c))
		return
	}

	preview, err := h.inviteService.GetInvitePreview(c.Request.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInviteNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrInviteExpired), errors.Is(err, errs.ErrInviteMaxUsesReached):
			c.JSON(http.StatusGone, model.NewAPIError(http.StatusGone, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to get invite preview", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get invite", c))
		}
		return
	}

	c.JSON(http.StatusOK, preview)
}

// @Summary Redeem an invite link
// @Description Authenticated user joins a contest via invite token
// @Tags invites
// @Produce json
// @Param token path string true "Invite token"
// @Success 201 {object} model.ContestParticipant
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 409 {object} model.APIError
// @Failure 410 {object} model.APIError
// @Security BearerAuth
// @Router /invites/{token}/redeem [post]
func (h *inviteHandler) RedeemInvite(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Token is required", c))
		return
	}

	user := c.GetString(model.UserKey)
	participant, err := h.inviteService.RedeemInvite(c.Request.Context(), token, user)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInviteNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrInviteExpired), errors.Is(err, errs.ErrInviteMaxUsesReached):
			c.JSON(http.StatusGone, model.NewAPIError(http.StatusGone, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrAlreadyParticipant):
			c.JSON(http.StatusConflict, model.NewAPIError(http.StatusConflict, util.CapitalizeFirstLetter(err), c))
		case errors.Is(err, errs.ErrNotEnoughSquares):
			c.JSON(http.StatusConflict, model.NewAPIError(http.StatusConflict, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to redeem invite", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to redeem invite", c))
		}
		return
	}

	c.JSON(http.StatusCreated, participant)
}

// @Summary Get all invites for a contest
// @Description Owner gets all invite links for a contest
// @Tags invites
// @Produce json
// @Param id path string true "Contest ID"
// @Success 200 {array} model.ContestInvite
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/invites [get]
func (h *inviteHandler) GetInvites(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	user := c.GetString(model.UserKey)
	invites, err := h.inviteService.GetInvitesByContestID(c.Request.Context(), contestID, user)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		case errors.Is(err, errs.ErrInsufficientRole):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to get invites", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get invites", c))
		}
		return
	}

	c.JSON(http.StatusOK, invites)
}

// @Summary Delete an invite link
// @Description Owner deletes an invite link
// @Tags invites
// @Param id path string true "Contest ID"
// @Param inviteId path string true "Invite ID"
// @Success 204
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Security BearerAuth
// @Router /contests/{id}/invites/{inviteId} [delete]
func (h *inviteHandler) DeleteInvite(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	contestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Warn("invalid contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid contest ID", c))
		return
	}

	inviteID, err := uuid.Parse(c.Param("inviteId"))
	if err != nil {
		log.Warn("invalid invite id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid invite ID", c))
		return
	}

	user := c.GetString(model.UserKey)
	if err := h.inviteService.DeleteInvite(c.Request.Context(), contestID, inviteID, user); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, util.CapitalizeFirstLetter(errs.ErrContestNotFound), c))
		case errors.Is(err, errs.ErrInsufficientRole):
			c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, util.CapitalizeFirstLetter(err), c))
		default:
			log.Error("failed to delete invite", "error", err)
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to delete invite", c))
		}
		return
	}

	c.Status(http.StatusNoContent)
}
