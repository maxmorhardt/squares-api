package metrics

import "github.com/prometheus/client_golang/prometheus"

var requestSizeRejectedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "http_request_size_rejected_total",
		Help: "Total number of HTTP requests rejected for exceeding the maximum body size",
	},
)

func init() {
	prometheus.MustRegister(requestSizeRejectedTotal)
}

func IncRequestSizeRejected() { 
	requestSizeRejectedTotal.Inc()
}
