package worker

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func mockRunner(t *testing.T, espn *mocks.ESPNClient, gameSvc *mocks.GameService) (*runner, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	r := &runner{
		db:      gdb,
		worker:  newScoresWorker(espn, gameSvc, time.Minute, time.Hour),
		lockKey: 1,
	}
	return r, dbMock
}

func TestRunner_RunGuarded_LockAcquired(t *testing.T) {
	espn := mocks.NewESPNClient(t)
	espn.EXPECT().FetchScoreboard(mock.Anything, mock.Anything).Return(nil, nil)
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Ingest(mock.Anything, mock.Anything).Return(0, nil)

	r, dbMock := mockRunner(t, espn, gameSvc)
	dbMock.ExpectQuery(`pg_try_advisory_lock`).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	dbMock.ExpectExec(`pg_advisory_unlock`).WillReturnResult(sqlmock.NewResult(0, 1))

	r.runGuarded(context.Background())

	// holding the lock, the worker actually polls (asserted by the FetchScoreboard expectation)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestRunner_RunGuarded_LockNotAcquired(t *testing.T) {
	// no FetchScoreboard expectation: another replica holds the lock, so the worker must not poll
	espn := mocks.NewESPNClient(t)
	r, dbMock := mockRunner(t, espn, mocks.NewGameService(t))
	dbMock.ExpectQuery(`pg_try_advisory_lock`).WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(false))

	r.runGuarded(context.Background())

	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestRunner_Loop_StopsOnContextCancel(t *testing.T) {
	gameSvc := mocks.NewGameService(t)
	gameSvc.EXPECT().Activity(mock.Anything).Return(model.GameActivity{}, nil).Maybe()

	r, _ := mockRunner(t, mocks.NewESPNClient(t), gameSvc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		// cancelled context: the startup run can't acquire a connection and the loop exits immediately
		r.loop(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("loop did not stop on context cancel")
	}
}
