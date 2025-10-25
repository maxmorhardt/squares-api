package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

var (
	ErrDatabaseUnavailable     = errors.New("service temporarily unavailable, please try again later")
	errInvalidContestName      = errors.New("contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	errInvalidHomeTeamName     = errors.New("home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	errInvalidAwayTeamName     = errors.New("away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	errContestNotFound         = errors.New("contest not found")
	errUnauthorizedContestEdit = errors.New("only the contest owner or an admin can update this contest")
	errInvalidContestStatus    = errors.New("invalid contest status")
	errInvalidSquareValue      = errors.New("value must be 1-3 uppercase letters or numbers")
)

type ValidationService interface {
	ValidateNewContest(ctx context.Context, req *model.CreateContestRequest, user string) error
	ValidateContestUpdate(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error)
	ValidateSquareUpdate(ctx context.Context, squareID uuid.UUID, req *model.UpdateSquareRequest) error
	ValidateWSRequest(ctx context.Context, contestID uuid.UUID) error
}

type validationService struct {
	contestRepo repository.ContestRepository
	authService AuthService
}

func NewValidationService(contestRepo repository.ContestRepository, authService AuthService) ValidationService {
	return &validationService{
		contestRepo: contestRepo,
		authService: authService,
	}
}

func (s *validationService) ValidateNewContest(ctx context.Context, req *model.CreateContestRequest, user string) error {
	log := util.LoggerFromContext(ctx)

	if !s.isValidContestName(req.Name) {
		log.Warn("invalid contest name", "name", req.Name)
		return errInvalidContestName
	}

	if !s.isValidTeamName(req.HomeTeam) {
		log.Warn("invalid home team name", "home_team", req.HomeTeam)
		return errInvalidHomeTeamName
	}

	if !s.isValidTeamName(req.AwayTeam) {
		log.Warn("invalid away team name", "away_team", req.AwayTeam)
		return errInvalidAwayTeamName
	}

	if !s.authService.IsContestOwner(ctx, req.Owner, user) {
		log.Warn("user not authorized to create contest", "user", user, "owner", req.Owner)
		return fmt.Errorf("user %s is not authorized to create contest for %s", user, req.Owner)
	}

	exists, err := s.contestRepo.ExistsByUserAndName(ctx, req.Owner, req.Name)
	if err != nil {
		log.Error("failed to check if contest exists", "owner", req.Owner, "name", req.Name, "error", err)
		return ErrDatabaseUnavailable
	}

	if exists {
		log.Warn("contest already exists", "owner", req.Owner, "name", req.Name)
		return fmt.Errorf("contest already exists with name %s for user %s", req.Name, req.Owner)
	}

	return nil
}

func (s *validationService) isValidContestName(name string) bool {
	if len(name) == 0 || len(name) > 20 {
		return false
	}

	matches, _ := regexp.MatchString(`^[A-Za-z0-9\s\-_]{1,20}$`, name)
	return matches
}

func (s *validationService) isValidTeamName(name string) bool {
	if len(name) == 0 {
		return true
	}

	return s.isValidContestName(name)
}

func (s *validationService) ValidateContestUpdate(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found for update", "contest_id", contestID)
			return nil, errContestNotFound
		}

		log.Error("failed to get contest for ownership validation", "contest_id", contestID, "error", err)
		return nil, ErrDatabaseUnavailable
	}

	if !s.authService.IsContestOwner(ctx, contest.Owner, user) {
		log.Warn("user is not authorized to update contest", "contest_id", contestID, "owner", contest.Owner, "user", user)
		return nil, errUnauthorizedContestEdit
	}

	if req.Name != nil {
		if !s.isValidContestName(*req.Name) {
			log.Warn("invalid contest name in update", "name", *req.Name)
			return nil, errInvalidContestName
		}

		if *req.Name != contest.Name {
			exists, err := s.contestRepo.ExistsByUserAndName(ctx, contest.Owner, *req.Name)
			if err != nil {
				log.Error("failed to check contest name uniqueness", "user", user, "name", *req.Name, "error", err)
				return nil, ErrDatabaseUnavailable
			}

			if exists {
				log.Warn("contest name already exists for user", "user", contest.Owner, "name", *req.Name)
				return nil, fmt.Errorf("contest with name '%s' already exists for user %s", *req.Name, contest.Owner)
			}
		}
	}

	if req.HomeTeam != nil && !s.isValidTeamName(*req.HomeTeam) {
		log.Warn("invalid home team name in update", "home_team", *req.HomeTeam)
		return nil, errInvalidHomeTeamName
	}

	if req.AwayTeam != nil && !s.isValidTeamName(*req.AwayTeam) {
		log.Warn("invalid away team name in update", "away_team", *req.AwayTeam)
		return nil, errInvalidAwayTeamName
	}

	if req.Status != nil {
		if !req.Status.IsValid() {
			log.Warn("invalid contest status in update", "status", *req.Status)
			return nil, errInvalidContestStatus
		}

		if !contest.Status.CanTransitionTo(*req.Status) {
			log.Warn("invalid contest status transition", "contest_id", contestID, "current_status", contest.Status, "target_status", *req.Status)
			return nil, fmt.Errorf("cannot transition from %s to %s. Status must progress sequentially: ACTIVE → LOCKED → Q1 → Q2 → Q3 → Q4 → FINISHED, or to CANCELLED from any active state", contest.Status, *req.Status)
		}
	}

	log.Info("contest update validation passed", "contest_id", contestID, "user", user)
	return contest, nil
}

func (s *validationService) ValidateSquareUpdate(ctx context.Context, squareID uuid.UUID, req *model.UpdateSquareRequest) error {
	log := util.LoggerFromContext(ctx)

	exists, err := s.contestRepo.SquareExistsByID(ctx, squareID)
	if err != nil {
		log.Error("failed to check if square exists", "square_id", squareID, "error", err)
		return ErrDatabaseUnavailable
	}

	if !exists {
		log.Warn("square not found", "square_id", squareID)
		return gorm.ErrRecordNotFound
	}

	if !isValidSquareValue(req.Value) {
		log.Warn("invalid square value", "value", req.Value)
		return errInvalidSquareValue
	}

	return nil
}

func isValidSquareValue(val string) bool {
	if val == "" {
		return true
	}

	if len(val) > 3 {
		return false
	}

	matches, _ := regexp.MatchString(`^[A-Z0-9]{1,3}$`, val)
	return matches
}

func (s *validationService) ValidateWSRequest(ctx context.Context, contestID uuid.UUID) error {
	log := util.LoggerFromContext(ctx)

	exists, err := s.contestRepo.ExistsByID(ctx, contestID)

	if err != nil || !exists {
		log.Warn("contest not found")
		return gorm.ErrRecordNotFound
	}

	return nil
}
