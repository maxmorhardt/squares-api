package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/maxmorhardt/squares-api/docs"
	"github.com/maxmorhardt/squares-api/internal/bootstrap"
	"github.com/maxmorhardt/squares-api/internal/config"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok && source != nil {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", filepath.Base(source.File), source.Line))
				}
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
	// initialize server with all routes and middleware
	router := bootstrap.NewServer()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// start server in a goroutine so it doesn't block signal handling
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start server", "error", err)
			panic(err)
		}
	}()

	slog.Info("server started", "addr", srv.Addr)

	// wait for interrupt or termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// give active connections time to finish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	// cleanup external connections
	config.CloseNATS()

	slog.Info("server exited")
}
