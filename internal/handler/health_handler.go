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
// @Success 200 {object} model.Health
// @Router /health [get]
func HealthCheck(c *gin.Context) {
  c.JSON(200, &model.Health{ Status: "UP" })
}