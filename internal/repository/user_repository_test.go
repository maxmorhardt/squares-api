package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userRows(email, displayName string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "email", "display_name", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), email, displayName, time.Now(), time.Now())
}

func TestUserRepository_GetOrCreate_Existing(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows("a@b.com", "Max"))

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
	assert.Equal(t, "Max", user.DisplayName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreate_CreatesWithFirstActivity(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	first := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT MIN`).WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(first))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows("a@b.com", "Max"))

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreate_CreatesWithoutActivity(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT MIN`).WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows("a@b.com", "Max"))

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max")

	require.NoError(t, err)
	assert.Equal(t, "Max", user.DisplayName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreate_InsertError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT MIN`).WillReturnRows(sqlmock.NewRows([]string{"min"}).AddRow(nil))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users"`).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreate_SelectError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("select failed"))

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetStats_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contest_participants"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "quarter_results"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	stats, err := repo.GetStats(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, int64(3), stats.ContestsCreated)
	assert.Equal(t, int64(7), stats.ContestsJoined)
	assert.Equal(t, int64(42), stats.SquaresClaimed)
	assert.Equal(t, int64(5), stats.QuarterWins)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetStats_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnError(errors.New("query failed"))

	stats, err := repo.GetStats(context.Background(), "a@b.com")

	require.Error(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOwnedActiveContestIDs_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	id := uuid.New()
	mock.ExpectQuery(`SELECT "id" FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

	ids, err := repo.GetOwnedActiveContestIDs(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{id}, ids)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOwnedActiveContestIDs_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT "id" FROM "contests"`).
		WillReturnError(errors.New("query failed"))

	ids, err := repo.GetOwnedActiveContestIDs(context.Background(), "a@b.com")

	require.Error(t, err)
	assert.Nil(t, ids)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ScrubUserData_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectBegin()
	// free squares in live contests, then anonymize owner/created_by/updated_by
	for i := 0; i < 4; i++ {
		mock.ExpectExec(`UPDATE "squares"`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < 3; i++ {
		mock.ExpectExec(`UPDATE "quarter_results"`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	for i := 0; i < 3; i++ {
		mock.ExpectExec(`UPDATE "contests"`).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectExec(`UPDATE "contest_invites"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "contest_participants"`).WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`DELETE FROM "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ScrubUserData(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_ScrubUserData_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "squares"`).WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	err := repo.ScrubUserData(context.Background(), "a@b.com")

	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
