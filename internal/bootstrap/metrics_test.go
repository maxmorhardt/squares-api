package bootstrap

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupMetricsServer_Disabled(t *testing.T) {
	t.Setenv("METRICS_ENABLED", "false")
	loadTestConfig(t)

	assert.NotPanics(t, func() {
		setupMetricsServer()
	})
}

func TestSetupMetricsServer_Enabled(t *testing.T) {
	t.Setenv("METRICS_ENABLED", "true")
	loadTestConfig(t)

	setupMetricsServer()

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://localhost:9090/metrics")
		if err != nil {
			return false
		}
		_ = resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 100*time.Millisecond)
}
