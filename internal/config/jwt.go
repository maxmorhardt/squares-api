package config

import (
	"log/slog"
	"os"
)

var jwtSecret string

func InitJWT() {
	jwtSecret = os.Getenv("JWT_SECRET")
	slog.Info("initialized JWT secret")
	if jwtSecret == "" {
		slog.Warn("JWT_SECRET not set, using default (NOT FOR PRODUCTION)")
		jwtSecret = "default-dev-secret-change-in-production"
	}
}

func GetJWTSecret() string {
	return jwtSecret
}
