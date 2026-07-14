package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockery mock would live in internal/mocks, which imports jobs, creating an import cycle with this white-box test package
type fakeEspn struct {
	games []model.ESPNGame
	err   error
}

func (f *fakeEspn) FetchScoreboard(context.Context, string) ([]model.ESPNGame, error) {
	return f.games, f.err
}

func TestPoller_Poll(t *testing.T) {
	// in Q2, so Q1 is complete and gets ingested
	espn := &fakeEspn{games: []model.ESPNGame{
		{ESPNID: "1", State: "in", Period: 2, HomeLine: []int{7}, AwayLine: []int{3}},
	}}
	gameRepo := mocks.NewGameRepository(t)
	gameRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil)
	gameRepo.EXPECT().UpsertScore(mock.Anything, mock.Anything).Return(true, nil)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().SyncGame(mock.Anything, mock.Anything).Return(nil)

	p := newPoller(espn, gameRepo, gameSvc)
	require.NoError(t, p.poll(context.Background()))
}

func TestPoller_Poll_FetchError(t *testing.T) {
	espn := &fakeEspn{err: errors.New("boom")}
	p := newPoller(espn, mocks.NewGameRepository(t), mocks.NewGameService(t))
	require.Error(t, p.poll(context.Background()))
}

func TestPoller_Poll_UpsertErrorSkipsGame(t *testing.T) {
	espn := &fakeEspn{games: []model.ESPNGame{{ESPNID: "1", State: "in"}}}
	gameRepo := mocks.NewGameRepository(t)
	gameRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(errors.New("db"))
	// UpsertScore / SyncGame must not run when the game upsert fails

	p := newPoller(espn, gameRepo, mocks.NewGameService(t))
	require.NoError(t, p.poll(context.Background()))
}
