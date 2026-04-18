package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testOrigin = "http://test-origin"

func loadWSTestConfig(t *testing.T) {
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
	t.Setenv("ALLOWED_ORIGINS", testOrigin)
	config.LoadEnv()
}

// mockContestRepo implements repository.ContestRepository for WS handler tests.
// Only GetByOwnerAndName is used by the WS handler; other methods are satisfied
// by embedding the interface (panics if called, but they won't be).
type mockContestRepo struct {
	repository.ContestRepository
	getByOwnerAndNameFn func(ctx context.Context, owner, name string) (*model.Contest, error)
}

func (m *mockContestRepo) GetByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error) {
	return m.getByOwnerAndNameFn(ctx, owner, name)
}

// mockWSService implements service.WebSocketService.
type mockWSService struct {
	service.WebSocketService
	handleFn func(ctx context.Context, contestID uuid.UUID, conn *websocket.Conn)
}

func (m *mockWSService) HandleWebSocketConnection(ctx context.Context, contestID uuid.UUID, conn *websocket.Conn) {
	if m.handleFn != nil {
		m.handleFn(ctx, contestID, conn)
	}
}

func newWSHandler(t *testing.T, repo *mockContestRepo, pSvc *mockParticipantService) WebSocketHandler {
	t.Helper()
	loadWSTestConfig(t)
	return NewWebSocketHandler(&mockWSService{}, repo, pSvc)
}

func dialWS(t *testing.T, server *httptest.Server, path string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + path
	header := http.Header{}
	header.Set("Origin", testOrigin)
	return websocket.DefaultDialer.Dial(wsURL, header)
}

// TestWSHandler_UpgradeFails sends a non-WS request; the upgrader rejects it.
func TestWSHandler_UpgradeFails(t *testing.T) {
	loadWSTestConfig(t)

	repo := &mockContestRepo{}
	pSvc := defaultMockParticipantService()
	h := NewWebSocketHandler(&mockWSService{}, repo, pSvc)

	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/owner/:owner/name/:name", h.ContestWSConnection)

	req, _ := http.NewRequest(http.MethodGet, "/ws/contests/owner/o1/name/n1", nil)
	w := doRequest(r, req)

	// Upgrader writes 400, handler may also attempt 500
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)
}

// TestWSHandler_ContestNotFound upgrades successfully then gets 4404 close.
func TestWSHandler_ContestNotFound(t *testing.T) {
	repo := &mockContestRepo{
		getByOwnerAndNameFn: func(_ context.Context, _, _ string) (*model.Contest, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	pSvc := defaultMockParticipantService()
	h := newWSHandler(t, repo, pSvc)

	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/owner/:owner/name/:name", h.ContestWSConnection)

	server := httptest.NewServer(r)
	defer server.Close()

	conn, resp, err := dialWS(t, server, "/ws/contests/owner/o1/name/missing")
	require.NoError(t, err)
	defer conn.Close()
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok)
	assert.Equal(t, 4404, closeErr.Code)
}

// TestWSHandler_ContestRepoError covers the non-404 repo error path.
func TestWSHandler_ContestRepoError(t *testing.T) {
	repo := &mockContestRepo{
		getByOwnerAndNameFn: func(_ context.Context, _, _ string) (*model.Contest, error) {
			return nil, assert.AnError
		},
	}
	pSvc := defaultMockParticipantService()
	h := newWSHandler(t, repo, pSvc)

	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/owner/:owner/name/:name", h.ContestWSConnection)

	server := httptest.NewServer(r)
	defer server.Close()

	conn, resp, err := dialWS(t, server, "/ws/contests/owner/o1/name/err")
	require.NoError(t, err)
	defer conn.Close()
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok)
	assert.Equal(t, 4500, closeErr.Code)
}

// TestWSHandler_Unauthorized covers the authorization failure path.
func TestWSHandler_Unauthorized(t *testing.T) {
	contestID := uuid.New()
	repo := &mockContestRepo{
		getByOwnerAndNameFn: func(_ context.Context, _, _ string) (*model.Contest, error) {
			return &model.Contest{ID: contestID, Owner: "owner1", Name: "test"}, nil
		},
	}
	pSvc := defaultMockParticipantService()
	pSvc.authorizeFn = func(_ context.Context, _ uuid.UUID, _ string, _ service.Action) error {
		return assert.AnError
	}
	h := newWSHandler(t, repo, pSvc)

	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.GET("/ws/contests/owner/:owner/name/:name", h.ContestWSConnection)

	server := httptest.NewServer(r)
	defer server.Close()

	conn, resp, err := dialWS(t, server, "/ws/contests/owner/owner1/name/test")
	require.NoError(t, err)
	defer conn.Close()
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok)
	assert.Equal(t, 4403, closeErr.Code)
}

// TestWSHandler_NATSUnavailable covers the NATS nil check path.
func TestWSHandler_NATSUnavailable(t *testing.T) {
	contestID := uuid.New()
	repo := &mockContestRepo{
		getByOwnerAndNameFn: func(_ context.Context, _, _ string) (*model.Contest, error) {
			return &model.Contest{ID: contestID, Owner: "owner1", Name: "test"}, nil
		},
	}
	pSvc := defaultMockParticipantService()
	pSvc.authorizeFn = func(_ context.Context, _ uuid.UUID, _ string, _ service.Action) error {
		return nil
	}
	h := newWSHandler(t, repo, pSvc)

	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/owner/:owner/name/:name", h.ContestWSConnection)

	server := httptest.NewServer(r)
	defer server.Close()

	conn, _, err := dialWS(t, server, "/ws/contests/owner/owner1/name/test")
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	closeErr, ok := err.(*websocket.CloseError)
	require.True(t, ok)
	assert.Equal(t, 4503, closeErr.Code)
}
