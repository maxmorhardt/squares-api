package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Test godoc
// @Summary      Health check
// @Description  Returns service status
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Security     BearerAuth
// @Router       /squares [get]
func Test(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}