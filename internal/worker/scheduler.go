package worker

import (
	"context"
	"time"

	"github.com/maxmorhardt/squares-api/internal/clients"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const scheduleWindow = 10 * 24 * time.Hour

type scheduler struct {
	espn     clients.ESPNClient
	gameRepo repository.GameRepository
}

func newScheduler(espn clients.ESPNClient, gameRepo repository.GameRepository) *scheduler {
	return &scheduler{espn: espn, gameRepo: gameRepo}
}

func (s *scheduler) sync(ctx context.Context) error {
	log := util.LoggerFromContext(ctx)

	now := time.Now()
	dates := now.Format("20060102") + "-" + now.Add(scheduleWindow).Format("20060102")

	games, err := s.espn.FetchScoreboard(ctx, dates)
	if err != nil {
		return err
	}

	for i := range games {
		if err := s.gameRepo.Upsert(ctx, games[i].ToGame()); err != nil {
			log.Error("failed to upsert scheduled game", "espn_id", games[i].ESPNID, "error", err)
		}
	}

	log.Info("schedule sync complete", "games", len(games))
	return nil
}
