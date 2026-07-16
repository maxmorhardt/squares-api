package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer_WiresRoutesWithoutInfra(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &model.AppConfig{}
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
		"GET /ws/contests/:id",
		"GET /users/me",
		"DELETE /users/me",
		"GET /users/me/stats",
		"GET /users/me/active-contests",
	}

	for _, route := range expected {
		assert.True(t, registered[route], "expected route %q to be registered", route)
	}
}

func TestNewServer_MetricsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &model.AppConfig{}
	cfg.Server.ContactRateLimit = 10
	cfg.Server.MetricsEnabled = true
	deps := &Dependencies{Config: cfg, DB: nil, NATS: nil}

	r := NewServer(deps)

	require.NotNil(t, r)
	assert.NotEmpty(t, r.Routes())
}

func TestSetupMiddleware_MetricsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &model.AppConfig{}
	cfg.Server.MetricsEnabled = false

	assert.NotPanics(t, func() { setupMiddleware(r, cfg) })
}

func TestSetupMiddleware_MetricsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &model.AppConfig{}
	cfg.Server.MetricsEnabled = true

	assert.NotPanics(t, func() { setupMiddleware(r, cfg) })
}
