package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnv_Success(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadEnv()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "testuser", cfg.DB.User)
	assert.Equal(t, "testpass", cfg.DB.Password)
	assert.Equal(t, "testdb", cfg.DB.Name)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, "smtp.test.com", cfg.SMTP.Host)
	assert.Equal(t, 587, cfg.SMTP.Port)
	assert.Equal(t, "smtpuser", cfg.SMTP.User)
	assert.Equal(t, "smtppass", cfg.SMTP.Password)
	assert.Equal(t, "support@test.com", cfg.SMTP.SupportEmail)
	assert.Equal(t, "test-client", cfg.OIDC.ClientID)
	assert.Equal(t, "nats://localhost:4222", cfg.NATS.URL)
	assert.Equal(t, "secret", cfg.Turnstile.SecretKey)
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	envVars := map[string]string{
		"DB_HOST":              "localhost",
		"DB_PORT":              "5432",
		"DB_USER":              "testuser",
		"DB_PASSWORD":          "testpass",
		"DB_NAME":              "testdb",
		"DB_SSL_MODE":          "disable",
		"SMTP_HOST":            "smtp.test.com",
		"SMTP_PORT":            "587",
		"SMTP_USER":            "smtpuser",
		"SMTP_PASSWORD":        "smtppass",
		"SUPPORT_EMAIL":        "support@test.com",
		"OIDC_CLIENT_ID":       "test-client",
		"NATS_URL":             "nats://localhost:4222",
		"TURNSTILE_SECRET_KEY": "secret",
	}

	for k, v := range envVars {
		t.Setenv(k, v)
	}
}

func TestLoadEnv_Defaults(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadEnv()

	require.NoError(t, err)
	assert.False(t, cfg.Server.MetricsEnabled)
	assert.Equal(t, "https://login.maxstash.io", cfg.OIDC.Issuer)
}

func TestLoadEnv_MetricsEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("METRICS_ENABLED", "true")

	cfg, err := LoadEnv()

	require.NoError(t, err)
	assert.True(t, cfg.Server.MetricsEnabled)
}

func TestLoadEnv_OptionalReadReplica(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_READ_HOST", "replica.host")
	t.Setenv("DB_READ_PORT", "5433")
	t.Setenv("DB_READ_USER", "replicauser")
	t.Setenv("DB_READ_PASSWORD", "replicapass")
	t.Setenv("DB_READ_NAME", "replicadb")
	t.Setenv("DB_READ_SSL_MODE", "require")

	cfg, err := LoadEnv()

	require.NoError(t, err)
	assert.Equal(t, "replica.host", cfg.DB.ReadHost)
	assert.Equal(t, 5433, cfg.DB.ReadPort)
	assert.Equal(t, "replicauser", cfg.DB.ReadUser)
	assert.Equal(t, "replicapass", cfg.DB.ReadPassword)
	assert.Equal(t, "replicadb", cfg.DB.ReadName)
	assert.Equal(t, "require", cfg.DB.ReadSSLMode)
}

func TestLoadEnv_AllowedOrigins_Default(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := LoadEnv()

	require.NoError(t, err)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.AllowedOrigins)
}

func TestLoadEnv_AllowedOrigins_Custom(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")

	cfg, err := LoadEnv()

	require.NoError(t, err)
	assert.Equal(t, []string{"https://app.example.com", "https://admin.example.com"}, cfg.Server.AllowedOrigins)
}

func TestLoadEnv_MissingRequired_Errors(t *testing.T) {
	for _, key := range []string{
		"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSL_MODE",
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD", "SUPPORT_EMAIL",
		"OIDC_CLIENT_ID", "NATS_URL", "TURNSTILE_SECRET_KEY",
	} {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}

	cfg, err := LoadEnv()

	require.Error(t, err)
	assert.Nil(t, cfg)
}
