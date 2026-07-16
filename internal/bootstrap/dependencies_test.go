package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDependencies_FailsWithoutEnv(t *testing.T) {
	deps, err := BuildDependencies()

	require.Error(t, err)
	assert.Nil(t, deps)
}

func TestBuildDependencies_FailsOnDBConnect(t *testing.T) {
	for k, v := range map[string]string{
		"DB_HOST":              "127.0.0.1",
		"DB_PORT":              "1",
		"DB_USER":              "u",
		"DB_PASSWORD":          "p",
		"DB_NAME":              "squares",
		"DB_SSL_MODE":          "disable",
		"SMTP_HOST":            "127.0.0.1",
		"SMTP_PORT":            "1",
		"SMTP_USER":            "u@example.com",
		"SMTP_PASSWORD":        "p",
		"SUPPORT_EMAIL":        "support@example.com",
		"OIDC_CLIENT_ID":       "client",
		"NATS_URL":             "nats://127.0.0.1:1",
		"TURNSTILE_SECRET_KEY": "key",
	} {
		t.Setenv(k, v)
	}

	deps, err := BuildDependencies()

	require.Error(t, err)
	assert.Nil(t, deps)
	assert.Contains(t, err.Error(), "database")
}
