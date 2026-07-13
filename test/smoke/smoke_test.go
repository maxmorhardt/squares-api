package smoke

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmoke(t *testing.T) {
	client := newClient(baseURL(t))

	endpoints := []string{
		"/health/live",
		"/health/ready",
		"/stats",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			resp, err := client.R().Get(endpoint)
			require.NoError(t, err, "GET %s failed", endpoint)
			assert.Equal(t, http.StatusOK, resp.StatusCode(), "unexpected status for %s", endpoint)
		})
	}
}

func baseURL(t *testing.T) string {
	url := os.Getenv("SMOKE_BASE_URL")
	if url == "" {
		t.Skip("SMOKE_BASE_URL not set; skipping smoke test")
	}
	return url
}

func newClient(base string) *resty.Client {
	return resty.New().
		SetBaseURL(base).
		SetTimeout(15 * time.Second).
		SetRetryCount(5).
		SetRetryWaitTime(5 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			return err != nil || r.StatusCode() != http.StatusOK
		})
}
