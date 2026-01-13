package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "Duration of HTTP requests",
			Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
		},
		[]string{"method", "path", "status"},
	)

	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, activeConnections)
}

func PrometheusMiddleware(c *gin.Context) {
	path := c.Request.URL.Path
	method := c.Request.Method

	// increment active connections
	activeConnections.Inc()

	// start timer for request duration
	timer := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
		status := strconv.Itoa(c.Writer.Status())
		httpRequestDuration.WithLabelValues(method, path, status).Observe(v)
	}))

	c.Next()

	// record metrics after request completes
	status := strconv.Itoa(c.Writer.Status())
	httpRequestsTotal.WithLabelValues(method, path, status).Inc()
	timer.ObserveDuration()
	activeConnections.Dec()
}
