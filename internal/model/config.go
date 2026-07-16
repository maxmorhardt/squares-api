package model

import "time"

type AppConfig struct {
	Server    ServerConfig
	DB        DatabaseConfig
	SMTP      SMTPConfig
	OIDC      OIDCConfig
	Turnstile TurnstileConfig
	NATS      NATSConfig
	Worker    WorkerConfig
}

type ServerConfig struct {
	Port             int      `env:"SERVER_PORT" envDefault:"8080"`
	MetricsEnabled   bool     `env:"METRICS_ENABLED" envDefault:"false"`
	AllowedOrigins   []string `env:"ALLOWED_ORIGINS" envDefault:"http://localhost:3000" envSeparator:","`
	ContactRateLimit int      `env:"CONTACT_RATE_LIMIT" envDefault:"10"`
}

type DatabaseConfig struct {
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

type SMTPConfig struct {
	Host         string `env:"SMTP_HOST,required"`
	Port         int    `env:"SMTP_PORT,required"`
	User         string `env:"SMTP_USER,required"`
	Password     string `env:"SMTP_PASSWORD,required"`
	SupportEmail string `env:"SUPPORT_EMAIL,required"`
}

type OIDCConfig struct {
	ClientID string `env:"OIDC_CLIENT_ID,required"`
	Issuer   string `env:"OIDC_ISSUER" envDefault:"https://login.maxstash.io"`
}

type TurnstileConfig struct {
	SecretKey string `env:"TURNSTILE_SECRET_KEY,required"`
	BaseURL   string `env:"TURNSTILE_BASE_URL" envDefault:"https://challenges.cloudflare.com"`
}

type NATSConfig struct {
	URL string `env:"NATS_URL,required"`
}

type WorkerConfig struct {
	Enabled          bool          `env:"SCORES_ENABLED" envDefault:"true"`
	ESPNBaseURL      string        `env:"ESPN_BASE_URL" envDefault:"https://site.api.espn.com"`
	PollInterval     time.Duration `env:"SCORES_POLL_INTERVAL" envDefault:"60s"`
	ScheduleInterval time.Duration `env:"SCORES_SCHEDULE_INTERVAL" envDefault:"6h"`
	LockKey          int64         `env:"SCORES_LOCK_KEY" envDefault:"910011"`
}
