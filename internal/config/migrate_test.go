package config

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationsEmbedded(t *testing.T) {
	src, err := iofs.New(migrationsFS, "migrations")
	require.NoError(t, err)
	t.Cleanup(func() { _ = src.Close() })

	v, err := src.First()
	require.NoError(t, err)
	assert.Equal(t, uint(1), v, "initial migration should be version 1")
}

func TestRunMigrations_BadConnection(t *testing.T) {
	// an unprepared mock connection fails the migration driver's setup queries
	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	assert.Error(t, runMigrations(sqlDB))
}
