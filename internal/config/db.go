package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var DB *gorm.DB

const (
	maxOpenConns    int           = 20
	maxIdleConns    int           = 5
	maxConnLifetime time.Duration = time.Hour
)

func InitDB() {
	setupPrimary()
	setupReadReplica()
}

func setupPrimary() {
	dsn := formatDSN(
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSL_MODE"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to primary database", "error", err)
		panic(err)
	}

	// auto-migrate models
	models := []any{
		&model.Contest{},
		&model.Square{},
		&model.QuarterResult{},
		&model.ContactSubmission{},
	}

	for _, m := range models {
		if err := db.AutoMigrate(m); err != nil {
			slog.Error("failed to migrate model", "error", err)
			panic(err)
		}
	}

	// configure connection pool
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(maxConnLifetime)

	DB = db
	slog.Info("primary database configured")
}

func setupReadReplica() {
	readHost := os.Getenv("DB_READ_HOST")
	if readHost == "" {
		slog.Info("no read replica configured")
		return
	}

	// build read replica dsn
	dsn := formatDSN(
		readHost,
		os.Getenv("DB_READ_PORT"),
		os.Getenv("DB_READ_USER"),
		os.Getenv("DB_READ_PASSWORD"),
		os.Getenv("DB_READ_NAME"),
		os.Getenv("DB_READ_SSL_MODE"),
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

	err := DB.Use(resolver)
	if err != nil {
		slog.Warn("failed to register read replica", "error", err)
		return
	}

	slog.Info("read replica configured")
}

func formatDSN(host, port, user, password, dbname, sslmode string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}
