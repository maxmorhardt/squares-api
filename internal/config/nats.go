package config

import (
	"fmt"
	"log/slog"

	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/nats-io/nats.go"
)

func InitNATS(cfg *model.AppConfig) (*nats.Conn, error) {
	natsConn, err := nats.Connect(
		cfg.NATS.URL,
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
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	metrics.SetNATSConnected(true)
	slog.Info("NATS connection established successfully", "url", cfg.NATS.URL)
	return natsConn, nil
}
