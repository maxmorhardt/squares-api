package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

var database *gorm.DB

const (
	maxOpenConns    int           = 20
	maxIdleConns    int           = 5
	maxConnLifetime time.Duration = time.Hour
)

func InitDB() {
	setupPrimary()
	
	if (Env().DB.ReadHost != "") {
		setupReadReplica()
	}
}

func setupPrimary() {
	dsn := formatDSN(
		Env().DB.Host,
		Env().DB.Port,
		Env().DB.User,
		Env().DB.Password,
		Env().DB.Name,
		Env().DB.SSLMode,
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

	database = db
	slog.Info("primary database configured")
}

func setupReadReplica() {
	dsn := formatDSN(
		Env().DB.ReadHost,
		Env().DB.ReadPort,
		Env().DB.ReadUser,
		Env().DB.ReadPassword,
		Env().DB.ReadName,
		Env().DB.ReadSSLMode,
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

	err := database.Use(resolver)
	if err != nil {
		slog.Warn("failed to register read replica", "error", err)
		return
	}

	slog.Info("read replica configured")
}

func formatDSN(host string, port int, user, password, dbname, sslmode string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}

func DB() *gorm.DB {
	return database
}