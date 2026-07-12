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
	GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error)
	DeleteAccount(ctx context.Context, email string) error
}

type userService struct {
	repo           repository.UserRepository
	contestService ContestService
}

func NewUserService(repo repository.UserRepository, contestService ContestService) UserService {
	return &userService{
		repo:           repo,
		contestService: contestService,
	}
}

func (s *userService) GetProfile(ctx context.Context, email, defaultDisplayName string) (*model.User, error) {
	log := util.LoggerFromContext(ctx)

	user, err := s.repo.GetOrCreate(ctx, email, defaultDisplayName)
	if err != nil {
		log.Error("failed to get or create user", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return user, nil
}

func (s *userService) GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error) {
	log := util.LoggerFromContext(ctx)

	stats, err := s.repo.GetStats(ctx, email)
	if err != nil {
		log.Error("failed to get user stats", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return stats, nil
}

func (s *userService) DeleteAccount(ctx context.Context, email string) error {
	log := util.LoggerFromContext(ctx)

	// soft-delete owned contests first so participants get the usual ws notifications
	contestIDs, err := s.repo.GetOwnedActiveContestIDs(ctx, email)
	if err != nil {
		log.Error("failed to list owned contests for account deletion", "error", err)
		return errs.ErrDatabaseUnavailable
	}

	for _, id := range contestIDs {
		if err := s.contestService.DeleteContest(ctx, id, email); err != nil {
			log.Error("failed to delete owned contest during account deletion", "contest_id", id, "error", err)
			return errs.ErrDatabaseUnavailable
		}
	}

	if err := s.repo.ScrubUserData(ctx, email); err != nil {
		log.Error("failed to scrub user data", "error", err)
		return errs.ErrDatabaseUnavailable
	}

	log.Info("account deleted")
	return nil
}
