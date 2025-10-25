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

type ValidationService interface {
	ValidateNewContest(ctx context.Context, req *model.CreateContestRequest, user string) error
	ValidateContestUpdate(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) error
	ValidateSquareUpdate(ctx context.Context, req *model.UpdateSquareRequest) error
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

	if !isValidContestName(req.Name) {
		log.Warn("invalid contest name", "name", req.Name)
		return errors.New("contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if !isValidTeamName(req.HomeTeam) {
		log.Warn("invalid home team name", "home_team", req.HomeTeam)
		return errors.New("home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if !isValidTeamName(req.AwayTeam) {
		log.Warn("invalid away team name", "away_team", req.AwayTeam)
		return errors.New("away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	exists, err := s.contestRepo.ExistsByUserAndName(ctx, req.Owner, req.Name)
	if err != nil {
		log.Error("failed to check if contest exists", "owner", req.Owner, "name", req.Name, "error", err)
		return err
	}

	if exists {
		log.Warn("contest already exists", "owner", req.Owner, "name", req.Name)
		return fmt.Errorf("contest already exists with name %s for user %s", req.Name, req.Owner)
	}

	return nil
}

func (s *validationService) ValidateContestUpdate(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) error {
	log := util.LoggerFromContext(ctx)

	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found for update", "contest_id", contestID)
			return errors.New("contest not found")
		}

		log.Error("failed to get contest for ownership validation", "contest_id", contestID, "error", err)
		return err
	}

	if !s.authService.IsContestOwner(ctx, contest.Owner, user) {
		log.Warn("user is not authorized to update contest", "contest_id", contestID, "owner", contest.Owner, "user", user)
		return errors.New("only the contest owner or an admin can update this contest")
	}

	if req.Name != nil {
		if !isValidContestName(*req.Name) {
			log.Warn("invalid contest name in update", "name", *req.Name)
			return errors.New("contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
		}

		if *req.Name != contest.Name {
			exists, err := s.contestRepo.ExistsByUserAndName(ctx, contest.Owner, *req.Name)
			if err != nil {
				log.Error("failed to check contest name uniqueness", "user", user, "name", *req.Name, "error", err)
				return err
			}

			if exists {
				log.Warn("contest name already exists for user", "user", contest.Owner, "name", *req.Name)
				return fmt.Errorf("contest with name '%s' already exists for user %s", *req.Name, contest.Owner)
			}
		}
	}

	if req.HomeTeam != nil && !isValidTeamName(*req.HomeTeam) {
		log.Warn("invalid home team name in update", "home_team", *req.HomeTeam)
		return errors.New("home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if req.AwayTeam != nil && !isValidTeamName(*req.AwayTeam) {
		log.Warn("invalid away team name in update", "away_team", *req.AwayTeam)
		return errors.New("away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if req.Status != nil {
		if !req.Status.IsValid() {
			log.Warn("invalid contest status in update", "status", *req.Status)
			return errors.New("invalid contest status")
		}

		if !contest.Status.CanTransitionTo(*req.Status) {
			log.Warn("invalid contest status transition", "contest_id", contestID, "current_status", contest.Status, "target_status", *req.Status)
			return fmt.Errorf("cannot transition from %s to %s. Status must progress sequentially: ACTIVE → LOCKED → Q1 → Q2 → Q3 → Q4 → FINISHED, or to CANCELLED from any active state", contest.Status, *req.Status)
		}
	}

	log.Info("contest update validation passed", "contest_id", contestID, "user", user)
	return nil
}

func isValidContestName(name string) bool {
	if len(name) == 0 || len(name) > 20 {
		return false
	}

	matches, _ := regexp.MatchString(`^[A-Za-z0-9\s\-_]{1,20}$`, name)
	return matches
}

func isValidTeamName(name string) bool {
	if len(name) == 0 {
		return true
	}

	return isValidContestName(name)
}

func (s *validationService) ValidateSquareUpdate(ctx context.Context, req *model.UpdateSquareRequest) error {
	log := util.LoggerFromContext(ctx)

	if !isValidSquareValue(req.Value) {
		log.Warn("invalid square value", "value", req.Value)
		return errors.New("value must be 1-3 uppercase letters or numbers")
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
