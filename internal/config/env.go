package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

var cfg *config

type config struct {
	Server    serverConfig
	DB        databaseConfig
	SMTP      smtpConfig
	OIDC      oidcConfig
	Turnstile turnstileConfig
	NATS      natsConfig
}

type serverConfig struct {
	MetricsEnabled bool `env:"METRICS_ENABLED" envDefault:"false"`
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
	Issuer   string `env:"OIDC_ISSUER" envDefault:"https://login.maxstash.io/application/o/squares/"`
}

type natsConfig struct {
	URL string `env:"NATS_URL,required"`
}

type turnstileConfig struct {
	SecretKey string `env:"TURNSTILE_SECRET_KEY,required"`
}

func LoadEnv() {
	_ = godotenv.Load()

	cfg = &config{}
	if err := env.Parse(cfg); err != nil {
		slog.Error("failed to handle configuration", "error", err)
		panic(err)
	}

	slog.Info("configuration loaded successfully")
}

func Env() *config {
	return cfg
}
