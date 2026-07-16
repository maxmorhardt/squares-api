package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const testOrigin = "http://test-origin"

func wsTestConfig(t *testing.T) *model.AppConfig {
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
	cfg, err := config.LoadEnv()
	require.NoError(t, err)
	return cfg
}

// builds a WS handler from mockery mocks; mocks are not t-bound because the
// handler runs in the httptest server goroutine and may call them after the
// test body returns
func newWSHandler(t *testing.T, repo *mocks.ContestRepository, wsSvc *mocks.WebSocketService, pSvc *mocks.ParticipantService, natsUp bool) WebSocketHandler {
	t.Helper()
	cfg := wsTestConfig(t)
	h := NewWebSocketHandler(wsSvc, repo, pSvc, cfg.Server.AllowedOrigins, nil)
	h.(*websocketHandler).natsAvailable = func() bool { return natsUp }
	return h
}

func serveWS(t *testing.T, h WebSocketHandler) *httptest.Server {
	t.Helper()
	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/:id", h.ContestWSConnection)
	return httptest.NewServer(r)
}

func dialWS(t *testing.T, server *httptest.Server, path string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + path
	header := http.Header{}
	header.Set("Origin", testOrigin)
	return websocket.DefaultDialer.Dial(wsURL, header)
}

func expectCloseCode(t *testing.T, server *httptest.Server, code int) {
	t.Helper()
	conn, _, err := dialWS(t, server, "/ws/contests/"+uuid.New().String())
	require.NoError(t, err)
	defer conn.Close()

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	var closeErr *websocket.CloseError
	require.True(t, errors.As(err, &closeErr))
	assert.Equal(t, code, closeErr.Code)
}

func TestWSHandler_UpgradeFails(t *testing.T) {
	h := newWSHandler(t, &mocks.ContestRepository{}, &mocks.WebSocketService{}, &mocks.ParticipantService{}, false)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/ws/contests/:id", h.ContestWSConnection)

	req, _ := http.NewRequest(http.MethodGet, "/ws/contests/"+uuid.New().String(), http.NoBody)
	w := doRequest(r, req)

	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)
}

func TestWSHandler_ContestNotFound(t *testing.T) {
	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	h := newWSHandler(t, repo, &mocks.WebSocketService{}, &mocks.ParticipantService{}, false)

	server := serveWS(t, h)
	defer server.Close()
	expectCloseCode(t, server, 4404)
}

func TestWSHandler_ContestRepoError(t *testing.T) {
	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	h := newWSHandler(t, repo, &mocks.WebSocketService{}, &mocks.ParticipantService{}, false)

	server := serveWS(t, h)
	defer server.Close()
	expectCloseCode(t, server, 4500)
}

func TestWSHandler_Unauthorized(t *testing.T) {
	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&model.Contest{ID: uuid.New(), Owner: "owner1", Name: "test"}, nil)
	pSvc := &mocks.ParticipantService{}
	pSvc.On("Authorize", mock.Anything, mock.Anything, mock.Anything, service.ActionView).Return(assert.AnError)
	h := newWSHandler(t, repo, &mocks.WebSocketService{}, pSvc, false)

	server := serveWS(t, h)
	defer server.Close()
	expectCloseCode(t, server, 4403)
}

func TestWSHandler_NATSUnavailable(t *testing.T) {
	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&model.Contest{ID: uuid.New(), Owner: "owner1", Name: "test"}, nil)
	pSvc := &mocks.ParticipantService{}
	pSvc.On("Authorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	h := newWSHandler(t, repo, &mocks.WebSocketService{}, pSvc, false)

	server := serveWS(t, h)
	defer server.Close()
	expectCloseCode(t, server, 4503)
}

func TestWSHandler_ParticipantsFetchFails(t *testing.T) {
	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).
		Return(&model.Contest{ID: uuid.New(), Owner: "owner1", Name: "test"}, nil)
	pSvc := &mocks.ParticipantService{}
	pSvc.On("Authorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	pSvc.On("GetParticipantsInternal", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	h := newWSHandler(t, repo, &mocks.WebSocketService{}, pSvc, true)

	server := serveWS(t, h)
	defer server.Close()
	expectCloseCode(t, server, 4500)
}

func TestWSHandler_HandoffToService(t *testing.T) {
	contestID := uuid.New()
	contest := &model.Contest{ID: contestID, Owner: "owner1", Name: "test"}
	participants := []model.ContestParticipant{{ContestID: contestID, UserID: "user1", Role: model.ParticipantRoleParticipant}}

	repo := &mocks.ContestRepository{}
	repo.On("GetByID", mock.Anything, mock.Anything).Return(contest, nil)
	pSvc := &mocks.ParticipantService{}
	pSvc.On("Authorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	pSvc.On("GetParticipantsInternal", mock.Anything, mock.Anything).Return(participants, nil)

	called := make(chan struct{}, 1)
	wsSvc := &mocks.WebSocketService{}
	wsSvc.On("HandleWebSocketConnection", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			conn := args.Get(3).(*websocket.Conn)
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			_ = conn.Close()
			called <- struct{}{}
		}).Return()

	h := newWSHandler(t, repo, wsSvc, pSvc, true)
	server := serveWS(t, h)
	defer server.Close()

	conn, _, err := dialWS(t, server, "/ws/contests/"+contestID.String())
	require.NoError(t, err)
	defer conn.Close()
	conn.ReadMessage() //nolint:errcheck // draining until server closes

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("HandleWebSocketConnection was not called")
	}
}
