package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
)

// HealthCheck godoc
// @Summary Health check
// @Description Returns UP if service is running
// @Tags health
// @Produce json
// @Success 200 {object} model.HealthResponse
// @Router /health [get]
func HealthCheck(c *gin.Context) {
  c.JSON(200, &model.HealthResponse{ Status: "UP" })
}