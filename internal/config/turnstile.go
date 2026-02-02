package config

import (
	"log/slog"
	"os"
)

var (
	TurnstileSecretKey string
)

func InitTurnstile() {
	TurnstileSecretKey = os.Getenv("TURNSTILE_SECRET_KEY")
	if TurnstileSecretKey == "" {
		// gotta be careful with this in prod
		TurnstileSecretKey = "1x0000000000000000000000000000000AA"
	}

	slog.Info("turnstile initialized")
}
