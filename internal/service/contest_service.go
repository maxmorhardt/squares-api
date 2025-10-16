package service

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type ContestService interface {
	GetAllContests(ctx context.Context) ([]model.Contest, error)
	CreateContest(ctx context.Context, req *model.CreateContestRequest) (*model.Contest, error)
	GetContestByID(ctx context.Context, contestID uuid.UUID) (*model.Contest, error)
	RandomizeLabels(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error)
	UpdateSquare(ctx context.Context, squareID uuid.UUID, req *model.UpdateSquareRequest) (*model.Square, error)
	GetContestsByUser(ctx context.Context, username string) ([]model.Contest, error)
}

type contestService struct {
	repo              repository.ContestRepository
	redisService      RedisService
	authService       AuthService
}

func NewContestService(
	repo repository.ContestRepository, 
	redisService RedisService, 
	authService AuthService, 
) ContestService {
	return &contestService{
		repo:         repo,
		redisService: redisService,
		authService:  authService,
	}
}

func (s *contestService) GetAllContests(ctx context.Context) ([]model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	contests, err := s.repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get all contests from repository", "error", err)
		return nil, err
	}

	return contests, nil
}

func (s *contestService) CreateContest(ctx context.Context, req *model.CreateContestRequest) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)
	
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

	if err := s.repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		return nil, err
	}

	log.Info("created contest", "name", req.Name, "contest_id", contest.ID, "owner", req.Owner)
	return &contest, nil
}

func initLabels() ([]byte, []byte) {
	xLabels := make([]int8, 10)
	yLabels := make([]int8, 10)
	for i := range 10 {
		xLabels[i] = -1
		yLabels[i] = -1
	}
		
	xLabelsJSON, _ := json.Marshal(xLabels)
	yLabelsJSON, _ := json.Marshal(yLabels)

	return xLabelsJSON, yLabelsJSON
}

func (s *contestService) GetContestByID(ctx context.Context, contestID uuid.UUID) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest by id", "contest_id", contestID, "error", err)
		return nil, err
	}

	log.Info("contest retrieved successfully", "contest_id", contest.ID)
	return contest, nil
}

func (s *contestService) RandomizeLabels(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	xLabels := generateRandomizedLabels()
	yLabels := generateRandomizedLabels()

	updatedContest, err := s.repo.UpdateLabels(ctx, contestID, xLabels, yLabels, user)
	if err != nil {
		log.Error("failed to update contest labels", "contest_id", contestID, "error", err)
		return nil, err
	}

	go func() {
		if err := s.redisService.PublishLabelsUpdate(context.Background(), updatedContest.ID, user, xLabels, yLabels); err != nil {
			log.Error("failed to publish contest update", "contestId", updatedContest.ID, "error", err)
		}
	}()

	log.Info("contest labels randomized successfully", "contest_id", contestID, "x_labels", xLabels, "y_labels", yLabels)
	return updatedContest, nil
}

func generateRandomizedLabels() []int8 {
	labels := make([]int8, 10)
	for i := range int8(10) {
		labels[i] = i
	}

	for i := len(labels) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		labels[i], labels[j] = labels[j], labels[i]
	}

	return labels
}

func (s *contestService) UpdateSquare(ctx context.Context, squareID uuid.UUID, req *model.UpdateSquareRequest) (*model.Square, error) {
	log := util.LoggerFromContext(ctx)

	user := ctx.Value(model.UserKey).(string)

	updatedSquare, err := s.repo.UpdateSquare(ctx, squareID, req.Value, user)
	if err != nil {
		log.Error("failed to update square", "square_id", squareID, "value", req.Value, "error", err)
		return nil, err
	}

	go func() {
		if err := s.redisService.PublishSquareUpdate(context.Background(), updatedSquare.ContestID, user, updatedSquare.ID, updatedSquare.Value); err != nil {
			log.Error("failed to publish square update", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID, "error", err)
		}
	}()

	log.Info("square updated successfully", "square_id", squareID, "value", req.Value)
	return updatedSquare, nil
}

func (c *contestService) GetContestsByUser(ctx context.Context, username string) ([]model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	contests, err := c.repo.GetAllByUser(ctx, username)
	if err != nil {
		log.Error("failed to get contests by user", "username", username, "error", err)
		return nil, err
	}

	log.Info("retrieved contests by username", "count", len(contests))
	return contests, nil
}