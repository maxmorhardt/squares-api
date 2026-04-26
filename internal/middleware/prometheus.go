package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func PrometheusMiddleware(c *gin.Context) {
	method := c.Request.Method

	metrics.HTTPActiveConnections.Inc()

	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(v)
	}))

	c.Next()

	status := strconv.Itoa(c.Writer.Status())
	path := c.FullPath()
	if path == "" {
		path = "unmatched"
	}
	metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	timer.ObserveDuration()
	metrics.HTTPActiveConnections.Dec()
}
