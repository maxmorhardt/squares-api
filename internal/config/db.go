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
	maxOpenConns int = 25
	maxIdleConns int = 5
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

	db.AutoMigrate(&model.Contest{})
	db.AutoMigrate(&model.Square{})

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("failed to get database instance", "error", err)
		panic(err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	slog.Info("database connection configured", "max_open_conns", maxOpenConns, "max_idle_conns", maxIdleConns)

	DB = db
}
