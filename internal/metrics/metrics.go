package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path"},
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

 timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(path))

 httpRequestsTotal.WithLabelValues(path).Inc()

 activeConnections.Inc()

 c.Next()

 timer.ObserveDuration()

 activeConnections.Dec()
}