package worker

import (
	"context"
	"time"

	"github.com/maxmorhardt/squares-api/internal/clients"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

// how far ahead each fetch reaches
const scheduleWindow = 10 * 24 * time.Hour

type scoresWorker struct {
	espn           clients.ESPNClient
	gameService    service.GameService
	activeInterval time.Duration
	idleInterval   time.Duration
}

func newScoresWorker(espn clients.ESPNClient, gameService service.GameService, activeInterval, idleInterval time.Duration) *scoresWorker {
	return &scoresWorker{
		espn:           espn,
		gameService:    gameService,
		activeInterval: activeInterval,
		idleInterval:   idleInterval,
	}
}

func (w *scoresWorker) run(ctx context.Context) error {
	// one fetch over the schedule window covers upcoming fixtures and live scores together
	now := time.Now()
	dates := now.Format("20060102") + "-" + now.Add(scheduleWindow).Format("20060102")

	games, err := w.espn.FetchScoreboard(ctx, dates)
	if err != nil {
		return err
	}

	// the service owns all persistence and contest reconciliation
	newScores, err := w.gameService.Ingest(ctx, games)
	if err != nil {
		return err
	}

	// stay silent in steady state; only surface actual scoring changes
	if newScores > 0 {
		util.LoggerFromContext(ctx).Info("recorded new quarter scores", "count", newScores)
		metrics.AddScoresRecorded(newScores)
	}

	return nil
}

func (w *scoresWorker) nextDelay(ctx context.Context) time.Duration {
	// derive pacing from shared DB state so every replica agrees, lock holder or not
	act, err := w.gameService.Activity(ctx)
	if err != nil {
		// couldn't tell, so check back soon rather than sleeping through a game
		return w.activeInterval
	}

	// a live game means poll fast
	if act.Live {
		return w.activeInterval
	}

	if !act.NextKickoff.IsZero() {
		until := time.Until(act.NextKickoff)
		switch {
		case until <= w.activeInterval:
			// kickoff is imminent, so ramp up now
			return w.activeInterval
		case until < w.idleInterval:
			// nothing live yet, but wake right as the game starts
			return until
		}
	}

	// nothing on the horizon, so idle
	return w.idleInterval
}
