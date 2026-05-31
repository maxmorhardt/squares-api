package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_WiresRoutesWithoutInfra(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Server.ContactRateLimit = 10
	deps := &Dependencies{Config: cfg, DB: nil, NATS: nil}

	r := NewServer(deps)
	require.NotNil(t, r)

	registered := make(map[string]bool)
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /health/live",
		"GET /health/ready",
		"GET /stats",
		"POST /contact",
		"PUT /contests",
		"GET /contests/owner/:owner",
		"GET /contests/me",
		"GET /contests/:id/participants",
		"POST /contests/:id/invites",
		"GET /invites/:token",
		"GET /ws/contests/owner/:owner/name/:name",
	}

	for _, route := range expected {
		assert.True(t, registered[route], "expected route %q to be registered", route)
	}
}

func TestNewServer_MetricsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Server.ContactRateLimit = 10
	cfg.Server.MetricsEnabled = true
	deps := &Dependencies{Config: cfg, DB: nil, NATS: nil}

	r := NewServer(deps)

	require.NotNil(t, r)
	assert.NotEmpty(t, r.Routes())
}

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

func TestSetupMiddleware_MetricsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.Server.MetricsEnabled = false

	assert.NotPanics(t, func() { setupMiddleware(r, cfg) })
}

func TestSetupMiddleware_MetricsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{}
	cfg.Server.MetricsEnabled = true

	assert.NotPanics(t, func() { setupMiddleware(r, cfg) })
}
