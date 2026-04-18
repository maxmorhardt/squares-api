package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/glebarez/sqlite"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func startTestNATS(t *testing.T) (string, func()) {
	t.Helper()
	opts := &natsserver.Options{Port: -1}
	server, err := natsserver.NewServer(opts)
	require.NoError(t, err)
	server.Start()
	require.True(t, server.ReadyForConnections(5_000_000_000)) // 5s timeout
	return server.ClientURL(), func() { server.Shutdown() }
}

func TestLiveness(t *testing.T) {
	h := NewHealthHandler(nil, func() *nats.Conn { return nil }, func() *oidc.IDTokenVerifier { return nil })
	r := newTestRouter()
	r.GET("/health/live", h.Liveness)

	req, _ := http.NewRequest(http.MethodGet, "/health/live", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "UP", resp["status"])
}

func TestReadiness_AllUp(t *testing.T) {
	db := setupTestDB(t)
	verifier := &oidc.IDTokenVerifier{}

	// Start an in-process NATS server for a real connected client
	natsURL, stopNATS := startTestNATS(t)
	defer stopNATS()

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	h := NewHealthHandler(db, func() *nats.Conn { return nc }, func() *oidc.IDTokenVerifier { return verifier })
	r := newTestRouter()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "UP", resp["status"])
	assert.Equal(t, "UP", resp["database"])
	assert.Equal(t, "UP", resp["nats"])
	assert.Equal(t, "UP", resp["oidc"])
}

func TestReadiness_NilDB(t *testing.T) {
	verifier := &oidc.IDTokenVerifier{}

	natsURL, stopNATS := startTestNATS(t)
	defer stopNATS()

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	h := NewHealthHandler(nil, func() *nats.Conn { return nc }, func() *oidc.IDTokenVerifier { return verifier })
	r := newTestRouter()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DOWN", resp["status"])
	assert.Equal(t, "DOWN", resp["database"])
}

func TestReadiness_NATSDown(t *testing.T) {
	db := setupTestDB(t)
	verifier := &oidc.IDTokenVerifier{}

	h := NewHealthHandler(db, func() *nats.Conn { return nil }, func() *oidc.IDTokenVerifier { return verifier })
	r := newTestRouter()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DOWN", resp["nats"])
}

func TestReadiness_OIDCDown(t *testing.T) {
	db := setupTestDB(t)

	natsURL, stopNATS := startTestNATS(t)
	defer stopNATS()

	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	defer nc.Close()

	h := NewHealthHandler(db, func() *nats.Conn { return nc }, func() *oidc.IDTokenVerifier { return nil })
	r := newTestRouter()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DOWN", resp["oidc"])
}

func TestReadiness_AllDown(t *testing.T) {
	h := NewHealthHandler(nil, func() *nats.Conn { return nil }, func() *oidc.IDTokenVerifier { return nil })
	r := newTestRouter()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "DOWN", resp["status"])
	assert.Equal(t, "DOWN", resp["database"])
	assert.Equal(t, "DOWN", resp["nats"])
	assert.Equal(t, "DOWN", resp["oidc"])
}
