package bootstrap

import (
	"testing"

	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestSetupMetricsServer_Disabled(t *testing.T) {
	cfg := &config.Config{}

	assert.NotPanics(t, func() { setupMetricsServer(cfg) })
}

func TestSetupMetricsServer_Enabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Server.MetricsEnabled = true

	assert.NotPanics(t, func() { setupMetricsServer(cfg) })
}
