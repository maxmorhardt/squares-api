package routes

import (
	"net/http"
	"squares-api/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(metrics.PrometheusMiddleware)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
		})
	})

	go http.ListenAndServe(":2112", promhttp.Handler())

	return r
}