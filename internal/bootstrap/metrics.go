package bootstrap

import (
	"log/slog"
	"net/http"

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
		if err := http.ListenAndServe(":9090", promhttp.Handler()); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()
}
