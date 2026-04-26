package metrics

import (
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/prometheus/client_golang/prometheus"
)

var authFailuresTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_failures_total",
		Help: "Total number of authentication failures by reason",
	},
	[]string{"reason"},
)

func init() {
	prometheus.MustRegister(authFailuresTotal)
}

func RecordAuthFailure(reason model.AuthFailureReason) {
	authFailuresTotal.WithLabelValues(string(reason)).Inc()
}
