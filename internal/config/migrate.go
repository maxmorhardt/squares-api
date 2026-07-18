package config

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/maxmorhardt/squares-api/internal/model"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(cfg *model.AppConfig) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to load embedded migrations: %w", err)
	}

	// own connection (not gorm's pool), so the deferred m.Close is safe
	m, err := migrate.NewWithSourceInstance("iofs", src, migrationDatabaseURL(cfg))
	if err != nil {
		_ = src.Close()
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			slog.Warn("failed to close migrator", "source_error", srcErr, "db_error", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			version, _, _ := m.Version()
			slog.Info("database schema already up to date", "version", version)
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	version, _, _ := m.Version()
	slog.Info("database schema migrations applied", "version", version)
	return nil
}

func migrationDatabaseURL(cfg *model.AppConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.DB.User, cfg.DB.Password),
		Host:   net.JoinHostPort(cfg.DB.Host, strconv.Itoa(cfg.DB.Port)),
		Path:   "/" + cfg.DB.Name,
	}
	q := u.Query()
	q.Set("sslmode", cfg.DB.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
