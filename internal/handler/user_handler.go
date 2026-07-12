package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type UserHandler interface {
	GetMe(c *gin.Context)
	GetMyStats(c *gin.Context)
	DeleteMe(c *gin.Context)
}

type userHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) UserHandler {
	return &userHandler{
		userService: userService,
	}
}

func toProfileResponse(user *model.User) model.UserProfileResponse {
	return model.UserProfileResponse{
		Email:       user.Email,
		DisplayName: user.DisplayName,
		CreatedAt:   user.CreatedAt.Format(time.RFC3339),
	}
}

// GetMe godoc
// @Summary Get the current user's profile
// @Description Returns the profile of the authenticated user, creating it on first access
// @Tags users
// @Produce json
// @Success 200 {object} model.UserProfileResponse
// @Failure 401 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /users/me [get]
func (h *userHandler) GetMe(c *gin.Context) {
	user := c.GetString(model.UserKey)

	// provider full name seeds the display name on first visit
	defaultDisplayName := user
	if claims := util.ClaimsFromContext(c.Request.Context()); claims != nil && claims.Name != "" {
		defaultDisplayName = claims.Name
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), user, defaultDisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		return
	}

	c.JSON(http.StatusOK, toProfileResponse(profile))
}

// GetMyStats godoc
// @Summary Get the current user's stats
// @Description Returns contest and square stats for the authenticated user
// @Tags users
// @Produce json
// @Success 200 {object} model.UserStatsResponse
// @Failure 401 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /users/me/stats [get]
func (h *userHandler) GetMyStats(c *gin.Context) {
	user := c.GetString(model.UserKey)

	stats, err := h.userService.GetStats(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		return
	}

	c.JSON(http.StatusOK, stats)
}

// DeleteMe godoc
// @Summary Delete the current user's account
// @Description Deletes owned contests, releases claimed squares, and removes all personal data
// @Tags users
// @Success 204
// @Failure 401 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /users/me [delete]
func (h *userHandler) DeleteMe(c *gin.Context) {
	user := c.GetString(model.UserKey)

	if err := h.userService.DeleteAccount(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, util.CapitalizeFirstLetter(err), c))
		return
	}

	c.Status(http.StatusNoContent)
}
