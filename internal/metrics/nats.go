package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	natsConnected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "nats_connected",
			Help: "Whether the NATS client is currently connected (1 = connected, 0 = disconnected)",
		},
	)

	natsMessagesPublishedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nats_messages_published_total",
			Help: "Total number of NATS messages published by subject prefix",
		},
		[]string{"subject"},
	)

	natsReconnectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_reconnects_total",
			Help: "Total number of NATS client reconnect events",
		},
	)

	natsDisconnectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_disconnects_total",
			Help: "Total number of NATS client disconnect events",
		},
	)
)

func init() {
	prometheus.MustRegister(
		natsConnected,
		natsMessagesPublishedTotal,
		natsReconnectsTotal,
		natsDisconnectsTotal,
	)
}

func SetNATSConnected(connected bool) {
	if connected {
		natsConnected.Set(1)
	} else {
		natsConnected.Set(0)
	}
}

func IncNATSMessagePublished(subject string) {
	natsMessagesPublishedTotal.WithLabelValues(subject).Inc()
}

func IncNATSReconnect() { 
	natsReconnectsTotal.Inc() 
}

func IncNATSDisconnect() { 
	natsDisconnectsTotal.Inc() 
}
