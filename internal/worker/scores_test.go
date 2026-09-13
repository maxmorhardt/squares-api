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

func TestScoresWorker_ScoreboardDates(t *testing.T) {
	// 00:20 UTC on the 22nd is still the evening of the 21st in ET, mid-game
	now := time.Date(2026, 8, 22, 0, 20, 0, 0, time.UTC)
	assert.Equal(t, "20260821-20260901", scoreboardDates(now))
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

func TestScoresWorker_LiveDates(t *testing.T) {
	now := time.Date(2026, 8, 22, 0, 20, 0, 0, time.UTC)
	assert.Equal(t, "20260821-20260823", liveDates(now))
}

func TestScoresWorker_Run_FirstRunFetchesFullSchedule(t *testing.T) {
	var got string
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, dates string) ([]model.ESPNGame, error) {
			got = dates
			return nil, nil
		})
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, nil)

	w := newWorker(t, espn, gameSvc)
	require.NoError(t, w.run(context.Background()))
	assert.Equal(t, scoreboardDates(time.Now()), got)
	assert.False(t, w.lastScheduleSync.IsZero())
}

func TestScoresWorker_Run_NarrowsWindowUntilIdleIntervalElapses(t *testing.T) {
	var got []string
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, dates string) ([]model.ESPNGame, error) {
			got = append(got, dates)
			return nil, nil
		}).Times(3)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, nil).Times(3)

	w := newWorker(t, espn, gameSvc)
	require.NoError(t, w.run(context.Background()))
	require.NoError(t, w.run(context.Background()))

	// age the last sync past the idle interval so the schedule refreshes again
	w.lastScheduleSync = time.Now().Add(-2 * w.idleInterval)
	require.NoError(t, w.run(context.Background()))

	assert.Equal(t, scoreboardDates(time.Now()), got[0])
	assert.Equal(t, liveDates(time.Now()), got[1])
	assert.Equal(t, scoreboardDates(time.Now()), got[2])
}

func TestScoresWorker_Run_FetchErrorLeavesScheduleUnsynced(t *testing.T) {
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	w := newWorker(t, espn, mocks.NewGameService(t))
	require.Error(t, w.run(context.Background()))

	// a failed wide fetch must not count as a schedule sync, or the window stays narrow
	assert.True(t, w.lastScheduleSync.IsZero())
}

func TestScoresWorker_Run_IngestErrorLeavesScheduleUnsynced(t *testing.T) {
	var got []string
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, dates string) ([]model.ESPNGame, error) {
			got = append(got, dates)
			return nil, nil
		}).Times(2)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, errors.New("db")).Once()
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, nil).Once()

	w := newWorker(t, espn, gameSvc)
	require.Error(t, w.run(context.Background()))

	// the wide window was never persisted, so the next run must retry it instead of narrowing
	assert.True(t, w.lastScheduleSync.IsZero())

	require.NoError(t, w.run(context.Background()))
	assert.Equal(t, scoreboardDates(time.Now()), got[1])
	assert.False(t, w.lastScheduleSync.IsZero())
}
