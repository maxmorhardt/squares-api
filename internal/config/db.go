package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const (
	maxOpenConns    int           = 20
	maxIdleConns    int           = 5
	maxConnLifetime time.Duration = time.Hour
)

func InitDB(cfg *Config) (*gorm.DB, error) {
	db, err := setupPrimary(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DB.ReadHost != "" {
		if err := validateReadReplicaConfig(cfg); err != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			return nil, err
		}
		if err := setupReadReplica(cfg, db); err != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			return nil, err
		}
	}

	return db, nil
}

func validateReadReplicaConfig(cfg *Config) error {
	var missing []string

	if cfg.DB.ReadPort == 0 {
		missing = append(missing, "DB_READ_PORT")
	}
	if cfg.DB.ReadUser == "" {
		missing = append(missing, "DB_READ_USER")
	}
	if cfg.DB.ReadPassword == "" {
		missing = append(missing, "DB_READ_PASSWORD")
	}
	if cfg.DB.ReadName == "" {
		missing = append(missing, "DB_READ_NAME")
	}
	if cfg.DB.ReadSSLMode == "" {
		missing = append(missing, "DB_READ_SSL_MODE")
	}

	if len(missing) > 0 {
		return fmt.Errorf("DB_READ_HOST is set but required read replica config is missing: %v", missing)
	}

	return nil
}

func setupPrimary(cfg *Config) (*gorm.DB, error) {
	dsn := formatDSN(
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Name,
		cfg.DB.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}

	// auto-migrate models
	models := []any{
		&model.Contest{},
		&model.Square{},
		&model.QuarterResult{},
		&model.ContactSubmission{},
		&model.ContestParticipant{},
		&model.ContestInvite{},
	}

	for _, m := range models {
		if migErr := db.AutoMigrate(m); migErr != nil {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			return nil, fmt.Errorf("failed to migrate model: %w", migErr)
		}
	}

	// configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(maxConnLifetime)

	// expose database/sql pool stats via the official prometheus collector
	if err := prometheus.Register(collectors.NewDBStatsCollector(sqlDB, cfg.DB.Name)); err != nil {
		slog.Warn("failed to register db stats collector", "error", err)
	}

	slog.Info("primary database configured")
	return db, nil
}

func setupReadReplica(cfg *Config, db *gorm.DB) error {
	dsn := formatDSN(
		cfg.DB.ReadHost,
		cfg.DB.ReadPort,
		cfg.DB.ReadUser,
		cfg.DB.ReadPassword,
		cfg.DB.ReadName,
		cfg.DB.ReadSSLMode,
	)

	// register read replica with dbresolver
	resolver := dbresolver.Register(dbresolver.Config{
		Replicas:          []gorm.Dialector{postgres.Open(dsn)},
		Policy:            dbresolver.RandomPolicy{},
		TraceResolverMode: true,
	})

	// configure replica connection pool
	resolver.SetConnMaxIdleTime(maxConnLifetime)
	resolver.SetConnMaxLifetime(maxConnLifetime)
	resolver.SetMaxIdleConns(maxIdleConns)
	resolver.SetMaxOpenConns(maxOpenConns)

	if err := db.Use(resolver); err != nil {
		return fmt.Errorf("failed to register read replica: %w", err)
	}

	slog.Info("read replica configured")
	return nil
}

func formatDSN(host string, port int, user, password, dbname, sslmode string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}
