package config

import (
	"log/slog"

	"github.com/nats-io/nats.go"
)

var (
	natsConn *nats.Conn
)

func InitNATS() {
	var err error
	natsConn, err = nats.Connect(Env().NATS.URL)
	if err != nil {
		slog.Error("failed to connect to NATS", "error", err)
		panic(err)
	}

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
