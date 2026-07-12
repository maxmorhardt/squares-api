package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server    serverConfig
	DB        databaseConfig
	SMTP      smtpConfig
	OIDC      oidcConfig
	Turnstile turnstileConfig
	NATS      natsConfig
}

type serverConfig struct {
	Port             int      `env:"SERVER_PORT" envDefault:"8080"`
	MetricsEnabled   bool     `env:"METRICS_ENABLED" envDefault:"false"`
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:3000" envSeparator:","`
	ContactRateLimit int      `env:"CONTACT_RATE_LIMIT" envDefault:"10"`
}

type databaseConfig struct {
	Host         string `env:"DB_HOST,required"`
	Port         int    `env:"DB_PORT,required"`
	User         string `env:"DB_USER,required"`
	Password     string `env:"DB_PASSWORD,required"`
	Name         string `env:"DB_NAME,required"`
	SSLMode      string `env:"DB_SSL_MODE,required"`
	ReadHost     string `env:"DB_READ_HOST"`
	ReadPort     int    `env:"DB_READ_PORT"`
	ReadUser     string `env:"DB_READ_USER"`
	ReadPassword string `env:"DB_READ_PASSWORD"`
	ReadName     string `env:"DB_READ_NAME"`
	ReadSSLMode  string `env:"DB_READ_SSL_MODE"`
}

type smtpConfig struct {
	Host         string `env:"SMTP_HOST,required"`
	Port         int    `env:"SMTP_PORT,required"`
	User         string `env:"SMTP_USER,required"`
	Password     string `env:"SMTP_PASSWORD,required"`
	SupportEmail string `env:"SUPPORT_EMAIL,required"`
}

type oidcConfig struct {
	ClientID string `env:"OIDC_CLIENT_ID,required"`
	Issuer   string `env:"OIDC_ISSUER" envDefault:"https://login.maxstash.io"`
}

type natsConfig struct {
	URL string `env:"NATS_URL,required"`
}

type turnstileConfig struct {
	SecretKey string `env:"TURNSTILE_SECRET_KEY,required"`
	BaseURL   string `env:"TURNSTILE_BASE_URL" envDefault:"https://challenges.cloudflare.com"`
}

func LoadEnv() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return cfg, nil
}
