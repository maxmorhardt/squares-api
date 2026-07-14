package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const systemUser = "system"

type GameService interface {
	GetUpcoming(ctx context.Context) ([]model.Game, error)
	SyncGame(ctx context.Context, gameID uuid.UUID) error
}

type gameService struct {
	gameRepo    repository.GameRepository
	contestRepo repository.ContestRepository
	natsService NatsService
}

func NewGameService(
	gameRepo repository.GameRepository,
	contestRepo repository.ContestRepository,
	natsService NatsService,
) GameService {
	return &gameService{
		gameRepo:    gameRepo,
		contestRepo: contestRepo,
		natsService: natsService,
	}
}

func (s *gameService) GetUpcoming(ctx context.Context) ([]model.Game, error) {
	log := util.LoggerFromContext(ctx)

	games, err := s.gameRepo.GetUpcoming(ctx)
	if err != nil {
		log.Error("failed to get upcoming games", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return games, nil
}

func (s *gameService) SyncGame(ctx context.Context, gameID uuid.UUID) error {
	log := util.LoggerFromContext(ctx)

	game, err := s.gameRepo.GetByID(ctx, gameID)
	if err != nil {
		log.Error("failed to get game for sync", "game_id", gameID, "error", err)
		return err
	}

	contests, err := s.contestRepo.GetByGameID(ctx, gameID)
	if err != nil {
		log.Error("failed to get contests for game", "game_id", gameID, "error", err)
		return err
	}

	for i := range contests {
		if err := s.reconcile(ctx, &contests[i], game); err != nil {
			log.Error("failed to reconcile contest with game", "contest_id", contests[i].ID, "game_id", gameID, "error", err)
		}
	}

	return nil
}

func (s *gameService) reconcile(ctx context.Context, contest *model.Contest, game *model.Game) error {
	log := util.LoggerFromContext(ctx)

	if contest.Status.IsTerminal() {
		return nil
	}

	// an ACTIVE contest hasn't locked its grid yet
	if contest.Status == model.ContestStatusActive {
		switch {
		case game.Status == model.GameStatusFinal:
			// the game ended before the grid ever locked; finalize straight from the final scores
			return s.finalize(ctx, contest, game)
		case game.Status == model.GameStatusInProgress && util.AllSquaresClaimed(contest):
			// full grid at kickoff; lock, randomize, and score live
			if err := s.autoStart(ctx, contest); err != nil {
				return err
			}
		default:
			// grid stays fillable until it fills up or the game ends
			return nil
		}
	}

	currentQuarter, ok := contest.Status.Quarter()
	if !ok {
		return nil
	}

	for i := range game.Scores {
		score := game.Scores[i]
		if score.Quarter < currentQuarter {
			continue
		}

		result, err := util.QuarterResultFor(contest, score.Quarter, score.HomeScore, score.AwayScore)
		if err != nil {
			log.Warn("skipping quarter, winner not determinable", "contest_id", contest.ID, "quarter", score.Quarter, "error", err)
			continue
		}

		next, valid := model.StatusAfterQuarter(score.Quarter)
		if !valid {
			continue
		}

		contest.Status = next
		if err := s.contestRepo.Update(ctx, contest); err != nil {
			log.Error("failed to advance contest after quarter", "contest_id", contest.ID, "quarter", score.Quarter, "error", err)
			return err
		}

		metrics.IncQuarterResult(score.Quarter)

		// publish synchronously and in order so clients apply quarters sequentially
		if err := s.natsService.PublishQuarterResult(contest.ID, systemUser, result); err != nil {
			log.Error("failed to publish quarter result", "contest_id", contest.ID, "quarter", score.Quarter, "error", err)
		}

		currentQuarter = score.Quarter + 1
		log.Info("applied quarter result", "contest_id", contest.ID, "game_id", game.ID, "quarter", score.Quarter, "winner", result.Winner)
	}

	return nil
}

func (s *gameService) autoStart(ctx context.Context, contest *model.Contest) error {
	log := util.LoggerFromContext(ctx)

	xLabels, yLabels, err := util.RandomizedLabels()
	if err != nil {
		return err
	}

	contest.XLabels = xLabels
	contest.YLabels = yLabels
	contest.Status = model.ContestStatusQ1
	contest.UpdatedBy = systemUser

	if err := s.contestRepo.Update(ctx, contest); err != nil {
		return err
	}

	metrics.IncContestStarted()

	// notify clients the grid is locked and randomized; strip heavy relations
	wsContest := *contest
	wsContest.Squares = nil
	wsContest.QuarterResults = nil
	wsContest.Game = nil
	if err := s.natsService.PublishContestUpdate(contest.ID, systemUser, &wsContest); err != nil {
		log.Error("failed to publish auto-start update", "contest_id", contest.ID, "error", err)
	}

	log.Info("auto-started game-linked contest", "contest_id", contest.ID)
	return nil
}

func (s *gameService) finalize(ctx context.Context, contest *model.Contest, game *model.Game) error {
	log := util.LoggerFromContext(ctx)

	// assign labels and empty squares don't win
	xLabels, yLabels, err := util.RandomizedLabels()
	if err != nil {
		return err
	}

	contest.XLabels = xLabels
	contest.YLabels = yLabels
	contest.Status = model.ContestStatusFinished
	contest.UpdatedBy = systemUser

	if err := s.contestRepo.Update(ctx, contest); err != nil {
		return err
	}

	// publish every quarter's outcome so connected clients render the final board
	for i := range game.Scores {
		score := game.Scores[i]
		result, resultErr := util.QuarterResultFor(contest, score.Quarter, score.HomeScore, score.AwayScore)
		if resultErr != nil {
			log.Warn("skipping quarter on finalize, winner not determinable", "contest_id", contest.ID, "quarter", score.Quarter, "error", resultErr)
			continue
		}

		metrics.IncQuarterResult(score.Quarter)
		if err := s.natsService.PublishQuarterResult(contest.ID, systemUser, result); err != nil {
			log.Error("failed to publish quarter result on finalize", "contest_id", contest.ID, "quarter", score.Quarter, "error", err)
		}
	}

	// notify clients the contest resolved; strip heavy relations
	wsContest := *contest
	wsContest.Squares = nil
	wsContest.QuarterResults = nil
	wsContest.Game = nil
	if err := s.natsService.PublishContestUpdate(contest.ID, systemUser, &wsContest); err != nil {
		log.Error("failed to publish finalize update", "contest_id", contest.ID, "error", err)
	}

	log.Info("finalized game-linked contest from final scores", "contest_id", contest.ID, "quarters", len(game.Scores))
	return nil
}
