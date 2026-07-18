package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestScoresWorker_Run(t *testing.T) {
	games := []model.ESPNGame{{ESPNID: "1", State: "in"}}
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).Return(games, nil)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, games).Return(2, nil)

	require.NoError(t, newWorker(t, espn, gameSvc).run(context.Background()))
}

func newWorker(t *testing.T, espn *mocks.ESPNClient, gameSvc *mocks.GameService) *scoresWorker {
	t.Helper()
	return newScoresWorker(espn, gameSvc, time.Minute, time.Hour)
}

func TestScoresWorker_Run_FetchError(t *testing.T) {
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))
	// must not run when the fetch fails
	require.Error(t, newWorker(t, espn, mocks.NewGameService(t)).run(context.Background()))
}

func TestScoresWorker_Run_IngestError(t *testing.T) {
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).
		Return([]model.ESPNGame{{ESPNID: "1"}}, nil)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, errors.New("db"))

	require.Error(t, newWorker(t, espn, gameSvc).run(context.Background()))
}

func TestScoresWorker_NextDelay_Live(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).Return(model.GameActivity{Live: true}, nil)

	w := newWorker(t, mocks.NewESPNClient(t), gameSvc)
	assert.Equal(t, w.activeInterval, w.nextDelay(context.Background()))
}

func TestScoresWorker_NextDelay_Idle(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).Return(model.GameActivity{}, nil)

	w := newWorker(t, mocks.NewESPNClient(t), gameSvc)
	assert.Equal(t, w.idleInterval, w.nextDelay(context.Background()))
}

func TestScoresWorker_NextDelay_KickoffImminent(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).
		Return(model.GameActivity{NextKickoff: time.Now().Add(10 * time.Second)}, nil)

	w := newWorker(t, mocks.NewESPNClient(t), gameSvc)
	// kickoff sooner than the active interval collapses to the active interval
	assert.Equal(t, w.activeInterval, w.nextDelay(context.Background()))
}

func TestScoresWorker_NextDelay_WakesAtKickoff(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).
		Return(model.GameActivity{NextKickoff: time.Now().Add(30 * time.Minute)}, nil)

	w := newWorker(t, mocks.NewESPNClient(t), gameSvc)
	delay := w.nextDelay(context.Background())
	// between active and idle: sleep until roughly the kickoff
	assert.Greater(t, delay, w.activeInterval)
	assert.LessOrEqual(t, delay, 30*time.Minute)
}

func TestScoresWorker_NextDelay_ActivityError(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).Return(model.GameActivity{}, errors.New("db"))

	w := newWorker(t, mocks.NewESPNClient(t), gameSvc)
	// on error, fall back to the active interval rather than sleeping through a game
	assert.Equal(t, w.activeInterval, w.nextDelay(context.Background()))
}
