package config

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/maxmorhardt/squares-api/internal/model"
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

func TestMigrationDatabaseURL(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.DB.Host = "db.example.com"
	cfg.DB.Port = 5432
	cfg.DB.User = "user"
	cfg.DB.Password = "p@ss word"
	cfg.DB.Name = "squares"
	cfg.DB.SSLMode = "require"

	url := migrationDatabaseURL(cfg)

	assert.Contains(t, url, "postgres://")
	assert.Contains(t, url, "db.example.com:5432")
	assert.Contains(t, url, "/squares")
	assert.Contains(t, url, "sslmode=require")
	assert.NotContains(t, url, "p@ss word", "credentials must be percent-encoded")
}

func TestRunMigrations_BadConnection(t *testing.T) {
	cfg := &model.AppConfig{}
	cfg.DB.Host = "127.0.0.1"
	cfg.DB.Port = 1
	cfg.DB.User = "u"
	cfg.DB.Password = "p"
	cfg.DB.Name = "n"
	cfg.DB.SSLMode = "disable"

	assert.Error(t, runMigrations(cfg))
}
