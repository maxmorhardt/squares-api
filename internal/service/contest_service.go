package service

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ContestService interface {
	GetAllContestsPaginated(ctx context.Context, page, limit int) ([]model.Contest, int64, error)
	CreateContest(ctx context.Context, req *model.CreateContestRequest) (*model.Contest, error)
	GetContestByID(ctx context.Context, contestID uuid.UUID) (*model.Contest, error)
	UpdateContest(ctx context.Context, contest *model.Contest, req *model.UpdateContestRequest, user string) (*model.Contest, error)
	DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error
	UpdateSquare(ctx context.Context, squareID uuid.UUID, req *model.UpdateSquareRequest) (*model.Square, error)
	GetContestsByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error)
}

type contestService struct {
	repo         repository.ContestRepository
	redisService RedisService
	authService  AuthService
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

func (s *contestService) GetAllContestsPaginated(ctx context.Context, page, limit int) ([]model.Contest, int64, error) {
	log := util.LoggerFromContext(ctx)

	contests, total, err := s.repo.GetAllPaginated(ctx, page, limit)
	if err != nil {
		log.Error("failed to get paginated contests from repository", "error", err)
		return nil, 0, err
	}

	return contests, total, nil
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
		Status:   model.ContestStatusActive,
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

func (s *contestService) UpdateContest(ctx context.Context, contest *model.Contest, req *model.UpdateContestRequest, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	needsUpdate := false
	var contestUpdate *model.ContestWSUpdate = &model.ContestWSUpdate{}
	if req.Name != nil && *req.Name != contest.Name {
		contest.Name = *req.Name
		needsUpdate = true
	}

	if req.HomeTeam != nil && *req.HomeTeam != contest.HomeTeam {
		contest.HomeTeam = *req.HomeTeam
		contestUpdate.HomeTeam = *req.HomeTeam
		needsUpdate = true
	}

	if req.AwayTeam != nil && *req.AwayTeam != contest.AwayTeam {
		contest.AwayTeam = *req.AwayTeam
		contestUpdate.AwayTeam = *req.AwayTeam
		needsUpdate = true
	}

	if req.Status != nil && *req.Status != contest.Status {
		if err := s.handleStatusTransition(ctx, contest, *req.Status, user); err != nil {
			log.Error("failed to handle status transition", "contest_id", contest.ID, "from", contest.Status, "to", *req.Status, "error", err)
			return nil, err
		}

		contest.Status = *req.Status
		contestUpdate.Status = *req.Status
		needsUpdate = true
	}

	if !needsUpdate {
		log.Info("no changes detected for contest update", "contest_id", contest.ID)
		return contest, nil
	}

	contest.UpdatedBy = user
	if err := s.repo.Update(ctx, contest); err != nil {
		log.Error("failed to save updated contest", "contest_id", contest.ID, "error", err)
		return nil, err
	}

	go func() {
		if err := s.redisService.PublishContestUpdate(context.Background(), contest.ID, user, contestUpdate); err != nil {
			log.Error("failed to publish contest update", "contest_id", contest.ID, "error", err)
		}
	}()

	log.Info("contest updated successfully", "contest_id", contest.ID, "user", user)
	return contest, nil
}

func (s *contestService) handleStatusTransition(ctx context.Context, contest *model.Contest, newStatus model.ContestStatus, user string) error {
	log := util.LoggerFromContext(ctx)

	switch newStatus {
	case model.ContestStatusLocked:
		return s.transitionToLocked(ctx, contest, user)
	case model.ContestStatusQ1:
		return s.transitionToQ1(ctx, contest, user)
	case model.ContestStatusQ2:
		return s.transitionToQ2(ctx, contest, user)
	case model.ContestStatusQ3:
		return s.transitionToQ3(ctx, contest, user)
	case model.ContestStatusQ4:
		return s.transitionToQ4(ctx, contest, user)
	case model.ContestStatusFinished:
		return s.transitionToFinished(ctx, contest, user)
	case model.ContestStatusCancelled:
		return s.transitionToCancelled(ctx, contest, user)
	default:
		log.Info("no special handling needed for status transition", "new_status", newStatus)
		return nil
	}
}

func (s *contestService) transitionToLocked(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement locked transition logic
	// - Generate random labels if not already set
	// - Validate contest is ready to be locked
	// - Send notifications about contest being locked

	log.Info("transitionToLocked not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToQ1 handles the transition from LOCKED to Q1
func (s *contestService) transitionToQ1(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement Q1 transition logic
	// - Initialize Q1 scoring
	// - Send game start notifications
	// - Set up Q1 timer if needed

	log.Info("transitionToQ1 not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToQ2 handles the transition from Q1 to Q2
func (s *contestService) transitionToQ2(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement Q2 transition logic
	// - Finalize Q1 scoring
	// - Initialize Q2 scoring
	// - Send quarter change notifications

	log.Info("transitionToQ2 not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToQ3 handles the transition from Q2 to Q3
func (s *contestService) transitionToQ3(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement Q3 transition logic
	// - Finalize Q2 scoring
	// - Initialize Q3 scoring
	// - Send quarter change notifications

	log.Info("transitionToQ3 not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToQ4 handles the transition from Q3 to Q4
func (s *contestService) transitionToQ4(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement Q4 transition logic
	// - Finalize Q3 scoring
	// - Initialize Q4 scoring
	// - Send quarter change notifications

	log.Info("transitionToQ4 not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToFinished handles the transition from Q4 to FINISHED
func (s *contestService) transitionToFinished(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement finished transition logic
	// - Finalize all scoring
	// - Calculate winners
	// - Send completion notifications
	// - Archive contest data

	log.Info("transitionToFinished not fully implemented", "contest_id", contest.ID)
	return nil
}

// transitionToCancelled handles the transition to CANCELLED from any state
func (s *contestService) transitionToCancelled(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// TODO: Implement cancelled transition logic
	// - Stop any active timers
	// - Send cancellation notifications
	// - Handle refunds if applicable
	// - Clean up resources

	log.Info("transitionToCancelled not fully implemented", "contest_id", contest.ID)
	return nil
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

func (s *contestService) DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error {
	log := util.LoggerFromContext(ctx)

	exists, err := s.repo.ExistsByID(ctx, contestID)
	if err != nil {
		log.Error("failed to check if contest exists", "contest_id", contestID, "error", err)
		return err
	}

	if !exists {
		log.Error("contest not found", "contest_id", contestID)
		return gorm.ErrRecordNotFound
	}

	if err := s.repo.Delete(ctx, contestID); err != nil {
		log.Error("failed to delete contest from repository", "contest_id", contestID, "error", err)
		return err
	}

	go func() {
		contestUpdate := &model.ContestWSUpdate{
			Status: model.ContestStatusDeleted,
		}
		if err := s.redisService.PublishContestUpdate(context.Background(), contestID, user, contestUpdate); err != nil {
			log.Error("failed to publish contest update", "contest_id", contestID, "error", err)
		}
	}()

	log.Info("deleted contest successfully", "contest_id", contestID)
	return nil
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

func (c *contestService) GetContestsByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error) {
	log := util.LoggerFromContext(ctx)

	contests, total, err := c.repo.GetAllByUserPaginated(ctx, username, page, limit)
	if err != nil {
		log.Error("failed to get paginated contests by user", "username", username, "error", err)
		return nil, 0, err
	}

	log.Info("retrieved paginated contests by username", "count", len(contests))
	return contests, total, nil
}
