package handler

import "github.com/gin-gonic/gin"

// HealthCheck godoc
// @Summary Health check
// @Description Returns UP if service is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheck(c *gin.Context) {
    c.JSON(200, gin.H{"status": "UP"})
}