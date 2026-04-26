package config

import (
	"log/slog"

	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/nats-io/nats.go"
)

var (
	natsConn *nats.Conn
)

func InitNATS() {
	var err error
	natsConn, err = nats.Connect(
		Env().NATS.URL,
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("NATS reconnected", "url", nc.ConnectedUrl())
			metrics.IncNATSReconnect()
			metrics.SetNATSConnected(true)
		}),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("NATS disconnected", "error", err)
			metrics.IncNATSDisconnect()
			metrics.SetNATSConnected(false)
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			slog.Info("NATS connection closed")
			metrics.SetNATSConnected(false)
		}),
	)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		panic(err)
	}

	metrics.SetNATSConnected(true)
	slog.Info("NATS connection established successfully", "url", Env().NATS.URL)
}

func NATS() *nats.Conn {
	return natsConn
}

func CloseNATS() {
	if natsConn != nil {
		natsConn.Close()
		slog.Info("NATS connection closed")
	}
}
