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

type ContactHandler interface {
	SubmitContact(c *gin.Context)
	UpdateContactSubmission(c *gin.Context)
}

type contactHandler struct {
	contactService service.ContactService
}

func NewContactHandler(contactService service.ContactService) ContactHandler {
	return &contactHandler{
		contactService: contactService,
	}
}

// @Summary Submit a contact form
// @Description Submit a contact form message
// @Tags contact
// @Accept json
// @Produce json
// @Param contact body model.ContactRequest true "Contact form data"
// @Success 202 "Contact form submitted successfully"
// @Failure 400 {object} model.APIError
// @Failure 429 {object} model.APIError
// @Router /contact [post]
func (h *contactHandler) SubmitContact(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse request body
	var req model.ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind contact request json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// get client ip address
	ipAddress := c.ClientIP()

	// submit contact via service
	if err := h.contactService.SubmitContact(c.Request.Context(), &req, ipAddress); err != nil {
		log.Error("failed to submit contact", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to submit contact form", c))
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Contact form submitted successfully. We will get back to you soon.",
	})
}

// @Summary Update a contact submission
// @Description Update the status and/or response of a contact submission. Requires squares-admins group.
// @Tags contact
// @Accept json
// @Produce json
// @Param id path string true "Contact Submission ID"
// @Param update body model.UpdateContactSubmissionRequest true "Update data"
// @Success 200 {object} model.ContactSubmission
// @Failure 400 {object} model.APIError
// @Failure 403 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /contact/{id} [patch]
func (h *contactHandler) UpdateContactSubmission(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse submission id from path
	submissionIDParam := c.Param("id")
	if submissionIDParam == "" {
		log.Warn("submission id not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Submission ID is required", c))
		return
	}

	submissionID, err := uuid.Parse(submissionIDParam)
	if err != nil {
		log.Warn("invalid submission id", "param", submissionIDParam, "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid Submission ID", c))
		return
	}

	// parse request body
	var req model.UpdateContactSubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("failed to bind update request json", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, util.CapitalizeFirstLetter(errs.ErrInvalidRequestBody), c))
		return
	}

	// update submission via service
	submission, err := h.contactService.UpdateContactSubmission(c.Request.Context(), submissionID, &req)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contact submission not found", c))
		} else {
			c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to update contact submission", c))
		}
		return
	}

	c.JSON(http.StatusOK, submission)
}
