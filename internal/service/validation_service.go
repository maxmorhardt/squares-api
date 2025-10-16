package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type ValidationService interface {
	ValidateNewContest(ctx context.Context, req *model.CreateContestRequest, user string) error
	ValidateSquareUpdate(ctx context.Context, req *model.UpdateSquareRequest) error
	ValidateWSRequest(ctx context.Context, contestID uuid.UUID) error
}

type validationService struct{}

func NewValidationService() ValidationService {
	return &validationService{}
}

func (s *validationService) ValidateNewContest(ctx context.Context, req *model.CreateContestRequest, user string) error {
	log := util.LoggerFromContext(ctx)

	if !isValidContestName(req.Name) {
		log.Error("invalid contest name", "name", req.Name)
		return errors.New("Contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if !isValidTeamName(req.HomeTeam) {
		log.Error("invalid home team name", "home_team", req.HomeTeam)
		return errors.New("Home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

	if !isValidTeamName(req.AwayTeam) {
		log.Error("invalid away team name", "away_team", req.AwayTeam)
		return errors.New("Away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores")
	}

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
	if (len(name) == 0) {
		return true
	}

	return isValidContestName(name)
}

func (s *validationService) ValidateSquareUpdate(ctx context.Context, req *model.UpdateSquareRequest) error {
	log := util.LoggerFromContext(ctx)

	if !isValidSquareValue(req.Value) {
		log.Error("invalid square value", "value", req.Value)
		return errors.New("Value must be 1-3 uppercase letters or numbers")
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

	repo := repository.NewContestRepository()
	_, err := repo.GetByID(ctx, contestID)

	if err != nil {
		log.Error("contest not found")
		return err
	}

	return nil
}
