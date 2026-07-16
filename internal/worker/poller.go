package worker

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/clients"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type poller struct {
	espn        clients.ESPNClient
	gameRepo    repository.GameRepository
	gameService service.GameService
}

func newPoller(espn clients.ESPNClient, gameRepo repository.GameRepository, gameService service.GameService) *poller {
	return &poller{espn: espn, gameRepo: gameRepo, gameService: gameService}
}

func (p *poller) poll(ctx context.Context) error {
	log := util.LoggerFromContext(ctx)

	games, err := p.espn.FetchScoreboard(ctx, "")
	if err != nil {
		return err
	}

	scores := 0
	for i := range games {
		eg := &games[i]
		game := eg.ToGame()
		if err := p.gameRepo.Upsert(ctx, game); err != nil {
			log.Error("failed to upsert game", "espn_id", eg.ESPNID, "error", err)
			continue
		}

		// persist each completed quarter's score, then bring linked contests up to date
		for _, q := range eg.CompletedQuarters() {
			score := &model.GameScore{GameID: game.ID, Quarter: q.Quarter, HomeScore: q.Home, AwayScore: q.Away}
			created, err := p.gameRepo.UpsertScore(ctx, score)
			if err != nil {
				log.Error("failed to upsert game score", "game_id", game.ID, "quarter", q.Quarter, "error", err)
				continue
			}
			if created {
				scores++
			}
		}

		if err := p.gameService.SyncGame(ctx, game.ID); err != nil {
			log.Error("failed to sync game", "game_id", game.ID, "error", err)
		}
	}

	log.Info("score poll complete", "games", len(games), "new_scores", scores)
	return nil
}
