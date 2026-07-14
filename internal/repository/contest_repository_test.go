package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContestRepository_GetVisibilityByID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"visibility"}).AddRow("public"))

	vis, err := repo.GetVisibilityByID(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Equal(t, model.ContestVisibilityPublic, vis)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_ExistsByOwnerAndName(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.ExistsByOwnerAndName(context.Background(), "owner", "name")

	require.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_GetAllByOwnerPaginated(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`SELECT \* FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("C1").AddRow("C2"))

	contests, total, err := repo.GetAllByOwnerPaginated(context.Background(), "owner", 1, 10, "foo")

	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, contests, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_Delete(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "contests" SET "status"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Delete(context.Background(), uuid.New()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_Update(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "contests"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), &model.Contest{ID: uuid.New(), Name: "x"}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_GetByGameID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	gameID := uuid.New()
	contestID := uuid.New()
	mock.ExpectQuery(`SELECT \* FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "game_id"}).AddRow(contestID, gameID))
	mock.ExpectQuery(`SELECT \* FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"contest_id"}).AddRow(contestID))

	contests, err := repo.GetByGameID(context.Background(), gameID)
	require.NoError(t, err)
	assert.Len(t, contests, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_UpdateSquare(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "squares"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	sq, err := repo.UpdateSquare(context.Background(), &model.Square{ID: uuid.New()}, "AB", "owner", "Owner Name")

	require.NoError(t, err)
	assert.Equal(t, "AB", sq.Value)
	assert.Equal(t, "owner", sq.Owner)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_ClearSquare(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "squares"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	sq, err := repo.ClearSquare(context.Background(), &model.Square{ID: uuid.New(), Value: "AB", Owner: "o"})

	require.NoError(t, err)
	assert.Empty(t, sq.Owner)
	assert.Empty(t, sq.Value)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_GetByID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	id := uuid.New()
	mock.ExpectQuery(`SELECT \* FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "C1"))
	mock.ExpectQuery(`SELECT \* FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "contest_id"}).AddRow(uuid.New(), id))
	mock.ExpectQuery(`SELECT \* FROM "quarter_results"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "contest_id"}))

	contest, err := repo.GetByID(context.Background(), id)

	require.NoError(t, err)
	assert.Equal(t, id, contest.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_Create(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contests"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO "squares"`).WillReturnResult(sqlmock.NewResult(1, 100))
	mock.ExpectExec(`INSERT INTO "contest_participants"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), &model.Contest{ID: uuid.New(), Name: "C1"}, &model.ContestParticipant{UserID: "owner"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_Create_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contests"`).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), &model.Contest{ID: uuid.New(), Name: "C1"}, &model.ContestParticipant{UserID: "owner"})
	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_GetAllByParticipantUserID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	id := uuid.New()
	mock.ExpectQuery(`SELECT contests\.\* FROM "contests"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(id, "C1"))
	mock.ExpectQuery(`SELECT \* FROM "squares"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "contest_id"}))
	mock.ExpectQuery(`SELECT \* FROM "quarter_results"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "contest_id"}))

	contests, err := repo.GetAllByParticipantUserID(context.Background(), "user1", "foo")

	require.NoError(t, err)
	assert.Len(t, contests, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContestRepository_CreateQuarterResult(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContestRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "quarter_results"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.CreateQuarterResult(context.Background(), &model.QuarterResult{ContestID: uuid.New(), Quarter: 1})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
