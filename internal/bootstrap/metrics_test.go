package bootstrap

import (
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSetupMetricsServer_Disabled(t *testing.T) {
	cfg := &model.AppConfig{}

	assert.NotPanics(t, func() { setupMetricsServer(cfg) })
}

func TestSetupMetricsServer_Enabled(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.Server.MetricsEnabled = true

	assert.NotPanics(t, func() { setupMetricsServer(cfg) })
}
