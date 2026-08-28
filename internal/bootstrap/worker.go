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
	participantRepo := repository.NewParticipantRepository(deps.DB)
	userRepo := repository.NewUserRepository(deps.DB)

	natsService := service.NewNatsService(deps.NATS)
	participantService := service.NewParticipantService(participantRepo, contestRepo, natsService)
	contestService := service.NewContestService(contestRepo, participantRepo, gameRepo, userRepo, natsService, participantService)
	gameService := service.NewGameService(gameRepo, contestRepo, contestService, natsService)

	runner := worker.NewRunner(deps.DB, gameService, cfg)

	ctx = util.ContextWithLogger(ctx, slog.Default().With("component", "scores-worker"))
	runner.Start(ctx)

	slog.Info("scores worker started", "active_interval", cfg.ActiveInterval, "idle_interval", cfg.IdleInterval)
}
