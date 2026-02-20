package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/bootstrap"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				source := a.Value.Any().(*slog.Source)
				a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line))
			}
			return a
		},
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
