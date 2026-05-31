package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParticipantRepository_GetByContestAndUser(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewParticipantRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contest_participants"`).
		WillReturnRows(sqlmock.NewRows([]string{"contest_id", "user_id", "role", "max_squares"}).
			AddRow(uuid.New(), "u1", "owner", 10))

	p, err := repo.GetByContestAndUser(context.Background(), uuid.New(), "u1")

	require.NoError(t, err)
	assert.Equal(t, "u1", p.UserID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParticipantRepository_GetAllByContestID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewParticipantRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contest_participants"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "role"}).
			AddRow("u1", "owner").
			AddRow("u2", "participant"))

	ps, err := repo.GetAllByContestID(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Len(t, ps, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParticipantRepository_GetTotalAllocatedSquares(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewParticipantRepository(gdb)

	mock.ExpectQuery(`SUM\(max_squares\)`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(25))

	total, err := repo.GetTotalAllocatedSquares(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParticipantRepository_CountSquaresByUser(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewParticipantRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountSquaresByUser(context.Background(), uuid.New(), "u1")

	require.NoError(t, err)
	assert.Equal(t, 3, count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestParticipantRepository_Delete(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewParticipantRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "contest_participants"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Delete(context.Background(), uuid.New(), "u1")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
