package bootstrap

import (
	"context"
	"log/slog"

	"github.com/maxmorhardt/squares-api/internal/jobs"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
)

func StartScoresWorker(ctx context.Context, deps *Dependencies) {
	cfg := deps.Config.Scores
	if !cfg.Enabled {
		slog.Info("scores worker disabled")
		return
	}

	gameRepo := repository.NewGameRepository(deps.DB)
	contestRepo := repository.NewContestRepository(deps.DB)
	natsService := service.NewNatsService(deps.NATS)
	gameService := service.NewGameService(gameRepo, contestRepo, natsService)

	runner := jobs.NewRunner(deps.DB, gameRepo, gameService, jobs.Config{
		ESPNBaseURL:      cfg.ESPNBaseURL,
		PollInterval:     cfg.PollInterval,
		ScheduleInterval: cfg.ScheduleInterval,
		LockKey:          cfg.LockKey,
	}, slog.Default())
	runner.Start(ctx)

	slog.Info("scores worker started", "poll_interval", cfg.PollInterval, "schedule_interval", cfg.ScheduleInterval)
}
