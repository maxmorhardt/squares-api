package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsRepository_GetStats_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewStatsRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "squares" JOIN contests c ON c\.id = squares\.contest_id AND c\.status <> \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	stats, err := repo.GetStats(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.ContestsCreatedToday)
	assert.Equal(t, int64(12), stats.SquaresClaimedToday)
	assert.Equal(t, int64(5), stats.TotalActiveContests)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatsRepository_GetStats_ContestsCreatedTodayError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewStatsRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnError(errors.New("query failed"))

	stats, err := repo.GetStats(context.Background())

	require.Error(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatsRepository_GetStats_SquaresClaimedTodayError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewStatsRepository(gdb)

	// first contests query succeeds
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	// squares query fails
	mock.ExpectQuery(`SELECT count\(\*\) FROM "squares"`).
		WillReturnError(errors.New("squares query failed"))

	stats, err := repo.GetStats(context.Background())

	require.Error(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStatsRepository_GetStats_TotalActiveContestsError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewStatsRepository(gdb)

	// first two queries succeed
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))
	// third (active contests) query fails
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnError(errors.New("active contests query failed"))

	stats, err := repo.GetStats(context.Background())

	require.Error(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, mock.ExpectationsWereMet())
}
