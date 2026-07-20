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

func TestLeaderboardService_GetLeaderboard_AssignsRanks(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).Return([]model.LeaderboardEntry{
		{DisplayName: "Max", QuarterWins: 12, SquaresClaimed: 48},
		{DisplayName: "Jordan", QuarterWins: 9, SquaresClaimed: 40},
		{DisplayName: "Sam", QuarterWins: 7, SquaresClaimed: 52},
	}, nil)

	got, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 25)

	require.NoError(t, err)
	require.Len(t, got.Entries, 3)
	assert.Equal(t, 1, got.Entries[0].Rank)
	assert.Equal(t, 2, got.Entries[1].Rank)
	assert.Equal(t, 3, got.Entries[2].Rank)
}

func TestLeaderboardService_GetLeaderboard_TiedPlayersShareRank(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).Return([]model.LeaderboardEntry{
		{DisplayName: "Max", QuarterWins: 10},
		{DisplayName: "Jordan", QuarterWins: 7},
		{DisplayName: "Sam", QuarterWins: 7},
		{DisplayName: "Riley", QuarterWins: 3},
	}, nil)

	got, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 25)

	require.NoError(t, err)
	// ties share a rank and the next player skips to the position they actually hold
	assert.Equal(t, 1, got.Entries[0].Rank)
	assert.Equal(t, 2, got.Entries[1].Rank)
	assert.Equal(t, 2, got.Entries[2].Rank)
	assert.Equal(t, 4, got.Entries[3].Rank)
}

func TestLeaderboardService_GetLeaderboard_PublishesFirstNameLastInitial(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).Return([]model.LeaderboardEntry{
		{DisplayName: "Max Morhardt", QuarterWins: 20},
		{DisplayName: "Mary Jane Watson", QuarterWins: 18},
		{DisplayName: "Jordan", QuarterWins: 16},
		{DisplayName: "max.morhardt@example.com", QuarterWins: 14},
		{DisplayName: "ada lovelace", QuarterWins: 12},
		{DisplayName: "   ", QuarterWins: 10},
		{DisplayName: "@handle", QuarterWins: 8},
	}, nil)

	got, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 25)

	require.NoError(t, err)
	assert.Equal(t, "Max M.", got.Entries[0].DisplayName)
	// only the final token is treated as the surname
	assert.Equal(t, "Mary W.", got.Entries[1].DisplayName)
	// a single name has no surname to abbreviate
	assert.Equal(t, "Jordan", got.Entries[2].DisplayName)
	// an email-shaped name is never published whole
	assert.Equal(t, "max.morhardt", got.Entries[3].DisplayName)
	assert.Equal(t, "ada L.", got.Entries[4].DisplayName)
	assert.Equal(t, "Player", got.Entries[5].DisplayName)
	assert.Equal(t, "Player", got.Entries[6].DisplayName)
}

func TestLeaderboardService_GetLeaderboard_NeverPublishesAFullSurname(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).Return([]model.LeaderboardEntry{
		{DisplayName: "Max Morhardt", QuarterWins: 12},
	}, nil)

	got, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 25)

	require.NoError(t, err)
	assert.NotContains(t, got.Entries[0].DisplayName, "Morhardt")
}

func TestLeaderboardService_GetLeaderboard_DefaultsLimit(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, service.DefaultLeaderboardLimit).Return(nil, nil)

	_, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 0)

	require.NoError(t, err)
}

func TestLeaderboardService_GetLeaderboard_ClampsLimit(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, service.MaxLeaderboardLimit).Return(nil, nil)

	_, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 5000)

	require.NoError(t, err)
}

func TestLeaderboardService_GetLeaderboard_CachesResult(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).
		Return([]model.LeaderboardEntry{{DisplayName: "Max", QuarterWins: 12}}, nil).Once()

	svc := service.NewLeaderboardService(repo)
	first, err := svc.GetLeaderboard(context.Background(), 25)
	require.NoError(t, err)
	second, err := svc.GetLeaderboard(context.Background(), 25)
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

func TestLeaderboardService_GetLeaderboard_Error(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetTopWinners(mock.Anything, 25).Return(nil, errors.New("query failed"))

	got, err := service.NewLeaderboardService(repo).GetLeaderboard(context.Background(), 25)

	require.Error(t, err)
	assert.Nil(t, got)
}

func TestLeaderboardService_GetUserRank_Success(t *testing.T) {
	want := &model.LeaderboardRankResponse{Rank: 7, TotalRanked: 143, QuarterWins: 5, Ranked: true}
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetUserRank(mock.Anything, "user@example.com").Return(want, nil)

	got, err := service.NewLeaderboardService(repo).GetUserRank(context.Background(), "user@example.com")

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLeaderboardService_GetUserRank_Error(t *testing.T) {
	repo := mocks.NewLeaderboardRepository(t)
	repo.EXPECT().GetUserRank(mock.Anything, "user@example.com").Return(nil, errors.New("query failed"))

	got, err := service.NewLeaderboardService(repo).GetUserRank(context.Background(), "user@example.com")

	require.Error(t, err)
	assert.Nil(t, got)
}
