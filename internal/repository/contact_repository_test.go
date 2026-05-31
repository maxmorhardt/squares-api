package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	// gorm preloads run in non-deterministic order, so don't require ordering
	mock.MatchExpectationsInOrder(false)

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return gdb, mock
}

func TestContactRepository_Create_Success(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContactRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contact_submissions"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.Create(context.Background(), &model.ContactSubmission{
		Name: "Jane", Email: "jane@test.com", Subject: "Hi", Message: "Hello",
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContactRepository_Create_Error(t *testing.T) {
	gdb, mock := newMockDB(t)
	repo := NewContactRepository(gdb)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "contact_submissions"`).WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err := repo.Create(context.Background(), &model.ContactSubmission{Name: "Jane"})

	require.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
