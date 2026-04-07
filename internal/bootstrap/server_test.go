package bootstrap

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/stretchr/testify/assert"
)

func loadTestConfig(t *testing.T) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "test")
	t.Setenv("DB_PASSWORD", "test")
	t.Setenv("DB_NAME", "test")
	t.Setenv("DB_SSL_MODE", "disable")
	t.Setenv("SMTP_HOST", "localhost")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "test")
	t.Setenv("SMTP_PASSWORD", "test")
	t.Setenv("SUPPORT_EMAIL", "test@test.com")
	t.Setenv("OIDC_CLIENT_ID", "test-client")
	t.Setenv("NATS_URL", "nats://localhost:4222")
	t.Setenv("TURNSTILE_SECRET_KEY", "test-secret")
	config.LoadEnv()
}

func TestSetupMiddleware_MetricsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("METRICS_ENABLED", "false")
	loadTestConfig(t)

	r := gin.New()
	setupMiddleware(r)

	// Recovery, RequestSizeLimit, CORS, Logger
	assert.Len(t, r.Handlers, 4)
}

func TestSetupMiddleware_MetricsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("METRICS_ENABLED", "true")
	loadTestConfig(t)

	r := gin.New()
	setupMiddleware(r)

	// Recovery, RequestSizeLimit, CORS, Logger, Prometheus
	assert.Len(t, r.Handlers, 5)
}
