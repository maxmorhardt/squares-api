package jobs

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func mockRunner(t *testing.T) (*Runner, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	return &Runner{db: gdb, log: slog.Default()}, mock
}

func TestRunner_RunGuarded_LockAcquired(t *testing.T) {
	r, mock := mockRunner(t)
	mock.ExpectQuery(`pg_try_advisory_lock`).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	mock.ExpectExec(`pg_advisory_unlock`).WillReturnResult(sqlmock.NewResult(0, 1))

	called := false
	r.runGuarded(context.Background(), "test", 1, func(context.Context) error {
		called = true
		return nil
	})

	assert.True(t, called)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunner_RunGuarded_LockNotAcquired(t *testing.T) {
	r, mock := mockRunner(t)
	mock.ExpectQuery(`pg_try_advisory_lock`).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(false))

	called := false
	r.runGuarded(context.Background(), "test", 1, func(context.Context) error {
		called = true
		return nil
	})

	// another replica holds the lock, so the job body must not run
	assert.False(t, called)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunner_Loop_StopsOnContextCancel(t *testing.T) {
	r, _ := mockRunner(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		// cancelled context: the startup tick can't acquire a connection and the loop exits immediately
		r.loop(ctx, "test", time.Hour, 1, func(context.Context) error { return nil })
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop on context cancel")
	}
}
