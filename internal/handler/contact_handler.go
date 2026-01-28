package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type ContactHandler interface {
	SubmitContact(c *gin.Context)
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

	// sanitize inputs to prevent email injection
	req.Name = util.SanitizeInput(req.Name)
	req.Email = util.SanitizeInput(req.Email)
	req.Subject = util.SanitizeInput(req.Subject)
	req.Message = util.SanitizeInput(req.Message)

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
