package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
)

type ContestService interface {
	CreateContest(ctx context.Context, req model.CreateContestRequest, user string) (*model.Contest, error)
	GetContestByID(ctx context.Context, contestID uuid.UUID) (*model.Contest, error)
	GetContestsByUser(ctx context.Context, username string) ([]model.Contest, error)
	UpdateSquare(ctx context.Context, squareID uuid.UUID, value string, user string) (*model.Square, error)
	RandomizeLabels(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error)
}

type contestService struct {
	repo              repository.ContestRepository
	redisService      RedisService
	authService       AuthService
	validationService ValidationService
	log               *slog.Logger
}

func NewContestService(
	repo repository.ContestRepository, 
	redisService RedisService, 
	authService AuthService, 
	validationService ValidationService,
	log *slog.Logger,
) ContestService {
	return &contestService{
		repo:              repo,
		redisService:      redisService,
		authService:       authService,
		validationService: validationService,
		log:               log,
	}
}

func (s *contestService) CreateContest(ctx context.Context, req model.CreateContestRequest, user string) (*model.Contest, error) {
	if !s.validationService.ValidateNewContest(c, req, user) {
		return
	}
	
	ctx := context.WithValue(c.Request.Context(), model.UserKey, user)
	xLabelsJSON, yLabelsJSON := initLabels()

	contest := model.Contest{
		Name:     req.Name,
		XLabels:  xLabelsJSON,
		YLabels:  yLabelsJSON,
		HomeTeam: req.HomeTeam,
		AwayTeam: req.AwayTeam,
		Owner:    req.Owner,
		Status:   "ACTIVE",
	}

	if err := repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, fmt.Sprintf("Failed to create new contest: %s", err), c))
		return
	}

	log.Info("created contest", "name", req.Name, "contest_id", contest.ID, "owner", req.Owner)
	c.JSON(http.StatusOK, contest)
}

// GetContestByID implements ContestService.
func (c *contestService) GetContestByID(ctx context.Context, contestID uuid.UUID) (*model.Contest, error) {
	panic("unimplemented")
}

// GetContestsByUser implements ContestService.
func (c *contestService) GetContestsByUser(ctx context.Context, username string) ([]model.Contest, error) {
	panic("unimplemented")
}

// RandomizeLabels implements ContestService.
func (c *contestService) RandomizeLabels(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error) {
	panic("unimplemented")
}

// UpdateSquare implements ContestService.
func (c *contestService) UpdateSquare(ctx context.Context, squareID uuid.UUID, value string, user string) (*model.Square, error) {
	panic("unimplemented")
}