package bootstrap

import (
	"context"
	"log/slog"

	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"github.com/maxmorhardt/squares-api/internal/worker"
)

func StartScoresWorker(ctx context.Context, deps *Dependencies) {
	cfg := deps.Config.Worker
	if !cfg.Enabled {
		slog.Info("scores worker disabled")
		return
	}

	gameRepo := repository.NewGameRepository(deps.DB)
	contestRepo := repository.NewContestRepository(deps.DB)
	natsService := service.NewNatsService(deps.NATS)
	gameService := service.NewGameService(gameRepo, contestRepo, natsService)

	runner := worker.NewRunner(deps.DB, gameRepo, gameService, cfg)

	// seed a logger the runner picks up from context, rather than passing it down as a parameter
	ctx = util.ContextWithLogger(ctx, slog.Default().With("component", "scores-worker"))
	runner.Start(ctx)

	slog.Info("scores worker started", "poll_interval", cfg.PollInterval, "schedule_interval", cfg.ScheduleInterval)
}
