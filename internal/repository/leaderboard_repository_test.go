package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderboardRepository_GetTopWinners_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	mock.ExpectQuery(`WITH wins AS`).
		WillReturnRows(
			sqlmock.NewRows([]string{"display_name", "quarter_wins", "squares_claimed", "quarters_played"}).
				AddRow("Max", 12, 48, 40).
				AddRow("Jordan", 9, 40, 36))

	entries, err := repo.GetTopWinners(context.Background(), 25)

	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "Max", entries[0].DisplayName)
	assert.Equal(t, int64(12), entries[0].QuarterWins)
	assert.Equal(t, int64(48), entries[0].SquaresClaimed)
	assert.Equal(t, int64(40), entries[0].QuartersPlayed)
	assert.Equal(t, "Jordan", entries[1].DisplayName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardRepository_GetTopWinners_Empty(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	mock.ExpectQuery(`WITH wins AS`).
		WillReturnRows(
			sqlmock.NewRows([]string{"display_name", "quarter_wins", "squares_claimed", "quarters_played"}))

	entries, err := repo.GetTopWinners(context.Background(), 25)

	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardRepository_GetTopWinners_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	mock.ExpectQuery(`WITH wins AS`).WillReturnError(errors.New("query failed"))

	entries, err := repo.GetTopWinners(context.Background(), 25)

	require.Error(t, err)
	assert.Nil(t, entries)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardRepository_GetUserRank_Ranked(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	mock.ExpectQuery(`WITH wins AS`).
		WillReturnRows(sqlmock.NewRows([]string{"total_ranked", "quarter_wins", "rank"}).
			AddRow(143, 5, 7))

	rank, err := repo.GetUserRank(context.Background(), "user@example.com")

	require.NoError(t, err)
	assert.Equal(t, 7, rank.Rank)
	assert.Equal(t, int64(143), rank.TotalRanked)
	assert.Equal(t, int64(5), rank.QuarterWins)
	assert.True(t, rank.Ranked)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardRepository_GetUserRank_Unranked(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	// a user with no wins comes back at rank 0
	mock.ExpectQuery(`WITH wins AS`).
		WillReturnRows(sqlmock.NewRows([]string{"total_ranked", "quarter_wins", "rank"}).
			AddRow(143, 0, 0))

	rank, err := repo.GetUserRank(context.Background(), "nobody@example.com")

	require.NoError(t, err)
	assert.Equal(t, 0, rank.Rank)
	assert.False(t, rank.Ranked)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLeaderboardRepository_GetUserRank_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewLeaderboardRepository(gdb)

	mock.ExpectQuery(`WITH wins AS`).WillReturnError(errors.New("query failed"))

	rank, err := repo.GetUserRank(context.Background(), "user@example.com")

	require.Error(t, err)
	assert.Nil(t, rank)
	assert.NoError(t, mock.ExpectationsWereMet())
}
