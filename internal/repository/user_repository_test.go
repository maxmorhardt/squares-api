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

func userRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "email", "display_name", "created_at", "updated_at"}).
		AddRow(uuid.New().String(), "a@b.com", "Max", time.Now(), time.Now())
}

func TestUserRepository_GetOrCreate_Existing(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows())

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max", "M")

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
	// a fresh sign-in clears any tombstone for the reused email
	mock.ExpectExec(`DELETE FROM deleted_accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows())

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max", "M")

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
	// a fresh sign-in clears any tombstone for the reused email
	mock.ExpectExec(`DELETE FROM deleted_accounts`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows())

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max", "M")

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

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max", "M")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetOrCreate_SelectError(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("select failed"))

	user, err := repo.GetOrCreate(context.Background(), "a@b.com", "Max", "M")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows())

	user, err := repo.GetByEmail(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("record not found"))

	user, err := repo.GetByEmail(context.Background(), "a@b.com")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateProfile_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(userRows())
	mock.ExpectExec(`UPDATE "squares" SET`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(`SELECT .* FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "contest_id", "owner", "value"}).
			AddRow(uuid.New(), uuid.New(), "a@b.com", "MM").
			AddRow(uuid.New(), uuid.New(), "a@b.com", "MM"))
	mock.ExpectCommit()

	user, squares, err := repo.UpdateProfile(context.Background(), "a@b.com", "MM")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
	require.Len(t, squares, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_UpdateProfile_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users" SET`).WillReturnError(errors.New("update failed"))
	mock.ExpectRollback()

	user, squares, err := repo.UpdateProfile(context.Background(), "a@b.com", "MM")

	require.Error(t, err)
	assert.Nil(t, user)
	assert.Nil(t, squares)
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

func TestUserRepository_GetActiveContests_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT c.id, c.name, c.owner`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "owner", "role"}).
			AddRow(uuid.NewString(), "test", "a@b.com", "owner").
			AddRow(uuid.NewString(), "pool", "other@b.com", "participant"))

	contests, err := repo.GetActiveContests(context.Background(), "a@b.com")

	require.NoError(t, err)
	require.Len(t, contests, 2)
	assert.Equal(t, "owner", contests[0].Role)
	assert.Equal(t, "participant", contests[1].Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_GetActiveContests_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT c.id, c.name, c.owner`).
		WillReturnError(errors.New("query failed"))

	contests, err := repo.GetActiveContests(context.Background(), "a@b.com")

	require.Error(t, err)
	assert.Nil(t, contests)
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
	mock.ExpectExec(`INSERT INTO deleted_accounts`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ScrubUserData(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_IsTokenRevoked_True(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	revoked, err := repo.IsTokenRevoked(context.Background(), "a@b.com", 100)

	require.NoError(t, err)
	assert.True(t, revoked)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_IsTokenRevoked_False(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	revoked, err := repo.IsTokenRevoked(context.Background(), "a@b.com", 100)

	require.NoError(t, err)
	assert.False(t, revoked)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_IsTokenRevoked_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewUserRepository(gdb)

	mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(errors.New("query failed"))

	revoked, err := repo.IsTokenRevoked(context.Background(), "a@b.com", 100)

	require.Error(t, err)
	assert.False(t, revoked)
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
