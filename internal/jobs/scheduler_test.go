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

func TestScheduler_Sync(t *testing.T) {
	espn := &fakeEspn{games: []model.ESPNGame{{ESPNID: "1"}, {ESPNID: "2"}}}
	gameRepo := mocks.NewGameRepository(t)
	gameRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Times(2)

	s := newScheduler(espn, gameRepo)
	require.NoError(t, s.sync(context.Background()))
}

func TestScheduler_Sync_FetchError(t *testing.T) {
	espn := &fakeEspn{err: errors.New("boom")}
	s := newScheduler(espn, mocks.NewGameRepository(t))
	require.Error(t, s.sync(context.Background()))
}

func TestScheduler_Sync_UpsertErrorContinues(t *testing.T) {
	espn := &fakeEspn{games: []model.ESPNGame{{ESPNID: "1"}, {ESPNID: "2"}}}
	gameRepo := mocks.NewGameRepository(t)
	// a failed upsert on one game doesn't abort the rest of the sync
	gameRepo.EXPECT().Upsert(mock.Anything, mock.Anything).Return(errors.New("db")).Times(2)

	s := newScheduler(espn, gameRepo)
	require.NoError(t, s.sync(context.Background()))
}
