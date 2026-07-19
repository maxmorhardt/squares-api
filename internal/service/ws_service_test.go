package service

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUserService struct {
	UserService
	valid bool
	err   error
}

func (f fakeUserService) IsTokenValid(context.Context, *model.Claims) (bool, error) {
	return f.valid, f.err
}

type fakeParticipantService struct {
	ParticipantService
	authErr error
}

func (f fakeParticipantService) Authorize(context.Context, uuid.UUID, string, Action) error {
	return f.authErr
}

func TestNewWebSocketService(t *testing.T) {
	require.NotNil(t, NewWebSocketService(nil, &fakeUserService{}, &fakeParticipantService{}))
}

func TestShouldCloseOnVisibility(t *testing.T) {
	log := slog.Default()
	contestID := uuid.New()
	claimsCtx := context.WithValue(context.Background(), model.ClaimsKey, &model.Claims{Email: "u"})

	privateUpdate := &model.WSUpdate{
		Type:      model.ContestUpdateType,
		ContestID: contestID,
		Contest:   &model.Contest{ID: contestID, Visibility: model.ContestVisibilityPrivate},
	}
	publicUpdate := &model.WSUpdate{
		Type:      model.ContestUpdateType,
		ContestID: contestID,
		Contest:   &model.Contest{ID: contestID, Visibility: model.ContestVisibilityPublic},
	}

	// not a contest update -> never kicks
	square := &websocketService{participantService: &fakeParticipantService{authErr: errors.New("nope")}}
	assert.False(t, square.shouldCloseOnVisibility(claimsCtx, &model.WSUpdate{Type: model.SquareUpdateType}, log))

	// still public -> never kicks
	assert.False(t, square.shouldCloseOnVisibility(claimsCtx, publicUpdate, log))

	// private + authorized (participant/viewer) -> keep
	authorized := &websocketService{participantService: &fakeParticipantService{}}
	assert.False(t, authorized.shouldCloseOnVisibility(claimsCtx, privateUpdate, log))

	// private + not authorized -> kick
	unauthorized := &websocketService{participantService: &fakeParticipantService{authErr: errors.New("not a participant")}}
	assert.True(t, unauthorized.shouldCloseOnVisibility(claimsCtx, privateUpdate, log))

	// private + no claims -> kick
	assert.True(t, authorized.shouldCloseOnVisibility(context.Background(), privateUpdate, log))
}

func TestShouldCloseConnection(t *testing.T) {
	log := slog.Default()
	ctx := context.Background()

	keep := &websocketService{userService: &fakeUserService{valid: true}}
	assert.False(t, keep.shouldCloseConnection(ctx, log), "valid token -> keep open")

	invalid := &websocketService{userService: &fakeUserService{valid: false}}
	assert.True(t, invalid.shouldCloseConnection(ctx, log), "invalid token -> close")

	dbErr := &websocketService{userService: &fakeUserService{err: errors.New("db down")}}
	assert.False(t, dbErr.shouldCloseConnection(ctx, log), "db error -> keep open, don't drop on transient failure")
}

func TestHandleChatMessage(t *testing.T) {
	s := &websocketService{}
	log := slog.Default()
	contestID := uuid.New()
	ctx := context.WithValue(context.Background(), model.ClaimsKey, &model.Claims{Email: "u", EmailVerified: true})

	s.handleChatMessage(ctx, contestID, "", log)                        // empty
	s.handleChatMessage(ctx, contestID, string(make([]byte, 300)), log) // too long
	s.handleChatMessage(ctx, contestID, "bad<script>", log)             // unsafe characters
	s.handleChatMessage(context.Background(), contestID, "hello", log)  // no claims in context
	s.handleChatMessage(ctx, contestID, "hello", log)                   // nats nil -> not available
}

func TestHandleWebSocketConnection_NATSNil(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	svc := &websocketService{nats: nil}
	contest := &model.Contest{ID: uuid.New()}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		svc.HandleWebSocketConnection(context.Background(), contest, nil, conn)
	}))
	defer ts.Close()

	client := dialWS(t, ts)
	defer client.Close()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, _, err := client.ReadMessage()
	require.Error(t, err)
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		assert.Equal(t, 4503, closeErr.Code)
	}
}

func dialWS(t *testing.T, ts *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

func TestSendWebSocketMessage(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		update := model.NewDisconnectedMessage(uuid.New(), uuid.New())
		err = sendWebSocketMessage(conn, slog.Default(), update)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	client := dialWS(t, ts)
	defer client.Close()

	require.NoError(t, client.SetReadDeadline(time.Now().Add(3*time.Second)))
	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(msg), model.DisconnectType)
}
