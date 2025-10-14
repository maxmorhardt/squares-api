package service

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type ValidationService interface	{
	ValidateNewContest(req model.CreateContestRequest, user string) bool
	ValidateSquareUpdate(req model.UpdateSquareRequest) bool
	ValidateWebSocketRequest() uuid.UUID
}

type validationService struct{}

func NewValidationService() ValidationService {
	return &validationService{}
}

func (s *validationService) ValidateNewContest(req model.CreateContestRequest, user string) bool {
	log := util.LoggerFromContext(c)
	
	if !isValidContestName(req.Name) {
		log.Error("invalid contest name", "name", req.Name)
		c.JSON(http.StatusBadRequest, model.NewAPIError(
			http.StatusBadRequest, 
			"Contest name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores",
			c,
		))

		return false
	}

	if !isValidTeamName(req.HomeTeam) {
		log.Error("invalid home team name", "homeTeam", req.HomeTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(
			http.StatusBadRequest,
			"Home team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores",
			c,
		))

		return false
	}

	if !isValidTeamName(req.AwayTeam) {
		log.Error("invalid away team name", "awayTeam", req.AwayTeam)
		c.JSON(http.StatusBadRequest, model.NewAPIError(
			http.StatusBadRequest,
			"Away team name must be 1-20 characters and contain only letters, numbers, spaces, hyphens, and underscores",
			c,
		))
		
		return false
	}

	return true
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

func (s *validationService) ValidateSquareUpdate(c *gin.Context, req model.UpdateSquareRequest) bool {
	log := util.LoggerFromContext(c)

	if !isValidSquareValue(req.Value) {
		log.Error("invalid square value", "value", req.Value)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Value must be 1-3 uppercase letters or numbers", c))

		return false
	}

	return true
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

func (s *validationService) ValidateWebSocketRequest(c *gin.Context) uuid.UUID {
	log := util.LoggerFromContext(c)

	contestId, err := uuid.Parse(c.Param("contestId"))
	if err != nil || contestId == uuid.Nil {
		log.Error("invalid or missing contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid or missing Contest ID", c))
		return uuid.Nil
	}

	repo := repository.NewContestRepository()
	_, err = repo.GetByID(c.Request.Context(), contestId)

	if err != nil {
		log.Error("contest not found", "contestId", contestId)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
		return uuid.Nil
	}

	return contestId
}
