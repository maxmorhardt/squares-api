package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestLoadEnv_Success(t *testing.T) {
	setRequiredEnv(t)

	LoadEnv()

	require.NotNil(t, Env())
	assert.Equal(t, "localhost", Env().DB.Host)
	assert.Equal(t, 5432, Env().DB.Port)
	assert.Equal(t, "testuser", Env().DB.User)
	assert.Equal(t, "testpass", Env().DB.Password)
	assert.Equal(t, "testdb", Env().DB.Name)
	assert.Equal(t, "disable", Env().DB.SSLMode)
	assert.Equal(t, "smtp.test.com", Env().SMTP.Host)
	assert.Equal(t, 587, Env().SMTP.Port)
	assert.Equal(t, "smtpuser", Env().SMTP.User)
	assert.Equal(t, "smtppass", Env().SMTP.Password)
	assert.Equal(t, "support@test.com", Env().SMTP.SupportEmail)
	assert.Equal(t, "test-client", Env().OIDC.ClientID)
	assert.Equal(t, "nats://localhost:4222", Env().NATS.URL)
	assert.Equal(t, "secret", Env().Turnstile.SecretKey)
}

func TestLoadEnv_Defaults(t *testing.T) {
	setRequiredEnv(t)

	LoadEnv()

	assert.False(t, Env().Server.MetricsEnabled)
	assert.Equal(t, "https://login.maxstash.io/application/o/squares/", Env().OIDC.Issuer)
}

func TestLoadEnv_MetricsEnabled(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("METRICS_ENABLED", "true")

	LoadEnv()

	assert.True(t, Env().Server.MetricsEnabled)
}

func TestLoadEnv_OptionalReadReplica(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("DB_READ_HOST", "replica.host")
	t.Setenv("DB_READ_PORT", "5433")
	t.Setenv("DB_READ_USER", "replicauser")
	t.Setenv("DB_READ_PASSWORD", "replicapass")
	t.Setenv("DB_READ_NAME", "replicadb")
	t.Setenv("DB_READ_SSL_MODE", "require")

	LoadEnv()

	assert.Equal(t, "replica.host", Env().DB.ReadHost)
	assert.Equal(t, 5433, Env().DB.ReadPort)
	assert.Equal(t, "replicauser", Env().DB.ReadUser)
	assert.Equal(t, "replicapass", Env().DB.ReadPassword)
	assert.Equal(t, "replicadb", Env().DB.ReadName)
	assert.Equal(t, "require", Env().DB.ReadSSLMode)
}

func TestLoadEnv_MissingRequired_Panics(t *testing.T) {
	// Clear all env vars that might be set
	os.Clearenv()

	assert.Panics(t, func() {
		LoadEnv()
	})
}
