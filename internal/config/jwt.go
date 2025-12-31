package config

import (
	"log/slog"
	"os"
)

var jwtSecret string

func InitJWT() {
	jwtSecret = os.Getenv("JWT_SECRET")
	
	if jwtSecret == "" {
		slog.Error("JWT_SECRET not set")
		panic("JWT_SECRET environment variable is required")
	}

	slog.Info("jwt configuration initialized")
}

func GetJWTSecret() string {
	return jwtSecret
}
