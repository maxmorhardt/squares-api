package config

import (
	"log/slog"
	"os"
)

var jwtSecret string

func init() {
	// get JWT secret from environment or use default for development
	jwtSecret = os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		slog.Warn("JWT_SECRET not set, using default (NOT FOR PRODUCTION)")
		jwtSecret = "default-dev-secret-change-in-production"
	}
}

// GetJWTSecret returns the JWT secret for signing tokens
func GetJWTSecret() string {
	return jwtSecret
}
