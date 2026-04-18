package bootstrap

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func setupMetricsServer() {
	if !config.Env().Server.MetricsEnabled {
		slog.Info("metrics disabled")
		return
	}

	slog.Info("starting metrics server on port 9090")
	go func() {
		srv := &http.Server{
			Addr:              ":9090",
			Handler:           promhttp.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()
}
