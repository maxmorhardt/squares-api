package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStatsService_GetStats_Success(t *testing.T) {
	want := &model.StatsResponse{ContestsCreatedToday: 3, SquaresClaimedToday: 12, TotalActiveContests: 5}
	repo := mocks.NewStatsRepository(t)
	repo.EXPECT().GetStats(mock.Anything).Return(want, nil)

	got, err := service.NewStatsService(repo).GetStats(context.Background())

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestStatsService_GetStats_RepoError(t *testing.T) {
	repo := mocks.NewStatsRepository(t)
	repo.EXPECT().GetStats(mock.Anything).Return(nil, errors.New("db down"))

	got, err := service.NewStatsService(repo).GetStats(context.Background())

	require.Error(t, err)
	assert.Nil(t, got)
}
