package service

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
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
	VerifyToken(ctx context.Context, token string) (*model.Claims, error)
	IsTokenValid(ctx context.Context, claims *model.Claims) (bool, error)
}

type userService struct {
	repo        repository.UserRepository
	natsService NatsService
	oidc        *oidc.IDTokenVerifier
}

func NewUserService(repo repository.UserRepository, natsService NatsService, oidcVerifier *oidc.IDTokenVerifier) UserService {
	return &userService{
		repo:        repo,
		natsService: natsService,
		oidc:        oidcVerifier,
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

func (s *userService) VerifyToken(ctx context.Context, token string) (*model.Claims, error) {
	if s.oidc == nil {
		return nil, fmt.Errorf("oidc verifier not configured")
	}

	idToken, err := s.oidc.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &model.Claims{}
	if parseErr := idToken.Claims(claims); parseErr != nil {
		return nil, fmt.Errorf("%w: %w", errs.ErrClaimsParse, parseErr)
	}

	valid, err := s.IsTokenValid(ctx, claims)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errs.ErrTokenInvalid
	}

	return claims, nil
}

func (s *userService) IsTokenValid(ctx context.Context, claims *model.Claims) (bool, error) {
	// iat is required: revocation compares it against the deletion instant
	if claims == nil || claims.Email == "" || !claims.EmailVerified ||
		claims.IssuedAt <= 0 || claims.Expire <= time.Now().Unix() {
		return false, nil
	}

	revoked, err := s.repo.IsTokenRevoked(ctx, claims.Email, claims.IssuedAt)
	if err != nil {
		return false, err
	}

	return !revoked, nil
}
