package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

const (
	maxOpenConns    int           = 25
	maxIdleConns    int           = 5
	maxConnLifetime time.Duration = time.Hour
)

func InitDB() {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSL_MODE")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		panic(err)
	}

	if err := db.AutoMigrate(&model.Contest{}); err != nil {
		slog.Error("failed to migrate Contest", "error", err)
		panic(err)
	}
	if err := db.AutoMigrate(&model.Square{}); err != nil {
		slog.Error("failed to migrate Square", "error", err)
		panic(err)
	}
	if err := db.AutoMigrate(&model.QuarterResult{}); err != nil {
		slog.Error("failed to migrate QuarterResult", "error", err)
		panic(err)
	}
	if err := db.AutoMigrate(&model.ContactSubmission{}); err != nil {
		slog.Error("failed to migrate ContactSubmission", "error", err)
		panic(err)
	}
	if err := db.AutoMigrate(&model.ContestParticipant{}); err != nil {
		slog.Error("failed to migrate ContestParticipant", "error", err)
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get database instance", "error", err)
		panic(err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(maxConnLifetime)

	slog.Info("database connection configured", "max_open_conns", maxOpenConns, "max_idle_conns", maxIdleConns, "max_conn_lifetime", maxConnLifetime)

	DB = db
}
