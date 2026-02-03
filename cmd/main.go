package main

import (
	"log/slog"
	"os"

	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/bootstrap"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	logger.Info("initialized logger")
}

// @title           Squares API
// @version         1.0.0
// @description     API for squares.maxstash.io
// @securityDefinitions.apikey BearerAuth
// @type apiKey
// @in header
// @name Authorization
func main() {
	if err := bootstrap.NewServer().Run(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
		panic(err)
	}
}
