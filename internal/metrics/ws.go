package metrics

import (
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	wsConnectionsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Number of currently active WebSocket connections",
		},
	)

	wsConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_connections_total",
			Help: "Total number of WebSocket connection attempts by result",
		},
		[]string{"result"},
	)

	wsDisconnectsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_disconnects_total",
			Help: "Total number of WebSocket disconnections by reason",
		},
		[]string{"reason"},
	)

	wsConnectionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "websocket_connection_duration_seconds",
			Help:    "Duration of WebSocket connections from upgrade to close",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600, 7200},
		},
	)

	wsMessagesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent to clients by type",
		},
		[]string{"type"},
	)

	wsMessagesReceived = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "websocket_messages_received_total",
			Help: "Total number of WebSocket messages received from clients",
		},
	)
)

func init() {
	prometheus.MustRegister(
		wsConnectionsActive,
		wsConnectionsTotal,
		wsDisconnectsTotal,
		wsConnectionDuration,
		wsMessagesSent,
		wsMessagesReceived,
	)
}

func RecordWSConnectionResult(result model.WSConnectionResult) {
	wsConnectionsTotal.WithLabelValues(string(result)).Inc()
}

func RecordWSDisconnect(reason model.WSDisconnectReason) {
	wsDisconnectsTotal.WithLabelValues(string(reason)).Inc()
}

func IncWSActiveConnections() { wsConnectionsActive.Inc() }

func DecWSActiveConnections() { wsConnectionsActive.Dec() }

func ObserveWSConnectionDuration(d time.Duration) {
	wsConnectionDuration.Observe(d.Seconds())
}

func IncWSMessageSent(msgType string) {
	wsMessagesSent.WithLabelValues(msgType).Inc()
}

func IncWSMessageReceived() { wsMessagesReceived.Inc() }
