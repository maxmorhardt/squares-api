package service

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type UserService interface {
	GetProfile(ctx context.Context, email, defaultDisplayName string) (*model.User, error)
	UpdateProfile(ctx context.Context, email, initials string) (*model.User, error)
	GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error)
	GetActiveContests(ctx context.Context, email string) ([]model.UserActiveContest, error)
	DeleteAccount(ctx context.Context, email string) error
}

type userService struct {
	repo        repository.UserRepository
	natsService NatsService
}

func NewUserService(repo repository.UserRepository, natsService NatsService) UserService {
	return &userService{
		repo:        repo,
		natsService: natsService,
	}
}

func (s *userService) GetProfile(ctx context.Context, email, defaultDisplayName string) (*model.User, error) {
	log := util.LoggerFromContext(ctx)

	user, err := s.repo.GetOrCreate(ctx, email, defaultDisplayName, util.InitialsFromName(defaultDisplayName))
	if err != nil {
		log.Error("failed to get or create user", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	log.Info("retrieved user profile")
	return user, nil
}

func (s *userService) UpdateProfile(ctx context.Context, email, initials string) (*model.User, error) {
	log := util.LoggerFromContext(ctx)

	user, squares, err := s.repo.UpdateProfile(ctx, email, initials)
	if err != nil {
		log.Error("failed to update user profile", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	// broadcast the new initials so live contest views update without a refresh
	for i := range squares {
		square := squares[i]
		go func() {
			if err := s.natsService.PublishSquareUpdate(square.ContestID, email, &square); err != nil {
				log.Error("failed to publish square update after initials change", "contestId", square.ContestID, "squareId", square.ID, "error", err)
			}
		}()
	}

	log.Info("updated user profile", "cascaded_squares", len(squares))
	return user, nil
}

func (s *userService) GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error) {
	log := util.LoggerFromContext(ctx)

	stats, err := s.repo.GetStats(ctx, email)
	if err != nil {
		log.Error("failed to get user stats", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	log.Info("retrieved user stats")
	return stats, nil
}

func (s *userService) GetActiveContests(ctx context.Context, email string) ([]model.UserActiveContest, error) {
	log := util.LoggerFromContext(ctx)

	active, err := s.repo.GetActiveContests(ctx, email)
	if err != nil {
		log.Error("failed to list active contests", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	log.Info("retrieved active contests", "count", len(active))
	return active, nil
}

func (s *userService) DeleteAccount(ctx context.Context, email string) error {
	log := util.LoggerFromContext(ctx)

	// deletion is blocked while the user owns or participates in live contests
	active, err := s.repo.GetActiveContests(ctx, email)
	if err != nil {
		log.Error("failed to list active contests for account deletion", "error", err)
		return errs.ErrDatabaseUnavailable
	}

	if len(active) > 0 {
		log.Info("account deletion blocked by active contests", "count", len(active))
		return errs.ErrAccountActiveContests
	}

	if err := s.repo.ScrubUserData(ctx, email); err != nil {
		log.Error("failed to scrub user data", "error", err)
		return errs.ErrDatabaseUnavailable
	}

	log.Info("account deleted")
	return nil
}
