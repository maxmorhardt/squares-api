package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestStartScoresWorker_Disabled(t *testing.T) {
	deps := &Dependencies{Config: &model.AppConfig{}}
	deps.Config.Worker.Enabled = false

	// disabled: returns immediately without touching the (nil) DB or NATS
	assert.NotPanics(t, func() {
		StartScoresWorker(context.Background(), deps)
	})
}

func TestStartScoresWorker_Enabled(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	deps := &Dependencies{Config: &model.AppConfig{}, DB: gdb}
	deps.Config.Worker.Enabled = true
	deps.Config.Worker.PollInterval = time.Hour
	deps.Config.Worker.ScheduleInterval = time.Hour

	// a cancelled context makes the loops start and then exit before polling ESPN
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() {
		StartScoresWorker(ctx, deps)
	})

	// let the background goroutines observe cancellation and return
	time.Sleep(100 * time.Millisecond)
}
