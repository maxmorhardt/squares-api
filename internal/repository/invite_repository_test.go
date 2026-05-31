package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestInviteRepository_GetByToken_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contest_invites"`).
		WillReturnRows(sqlmock.NewRows([]string{"token", "role", "max_squares"}).AddRow("tok123", "participant", 5))

	invite, err := repo.GetByToken(context.Background(), "tok123")

	require.NoError(t, err)
	assert.Equal(t, "tok123", invite.Token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_GetByToken_NotFound(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contest_invites"`).
		WillReturnRows(sqlmock.NewRows([]string{"token"}))

	_, err := repo.GetByToken(context.Background(), "missing")

	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_GetAllByContestID(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectQuery(`SELECT .* FROM "contest_invites"`).
		WillReturnRows(sqlmock.NewRows([]string{"token"}).AddRow("t1").AddRow("t2"))

	invites, err := repo.GetAllByContestID(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Len(t, invites, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_Create(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contest_invites"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Create(context.Background(), &model.ContestInvite{ContestID: uuid.New()}))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_RedeemInvite(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contest_participants"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`UPDATE "contest_invites"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.RedeemInvite(context.Background(), uuid.New(), &model.ContestParticipant{UserID: "u"})
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInviteRepository_Delete(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewInviteRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "contest_invites"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.Delete(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
