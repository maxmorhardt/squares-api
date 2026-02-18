package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ContestService interface {
	GetContestByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error)
	GetContestsByOwnerPaginated(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error)

	CreateContest(ctx context.Context, req *model.CreateContestRequest, user string) (*model.Contest, error)
	UpdateContest(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error)
	StartContest(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error)
	RecordQuarterResult(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error)
	DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error

	UpdateSquare(ctx context.Context, contestID uuid.UUID, squareID uuid.UUID, req *model.UpdateSquareRequest, user string) (*model.Square, error)
	ClearSquare(ctx context.Context, contestID uuid.UUID, squareID uuid.UUID, user string) (*model.Square, error)
}

type contestService struct {
	repo        repository.ContestRepository
	natsService NatsService
	authService AuthService
}

func NewContestService(
	repo repository.ContestRepository,
	natsService NatsService,
	authService AuthService,
) ContestService {
	return &contestService{
		repo:        repo,
		natsService: natsService,
		authService: authService,
	}
}

// ====================
// Getters
// ====================

func (s *contestService) GetContestByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	// get contest from repository
	contest, err := s.repo.GetByOwnerAndName(ctx, owner, name)
	if err != nil {
		log.Error("failed to get contest by owner and name", "owner", owner, "name", name, "error", err)
		return nil, err
	}

	log.Info("contest retrieved successfully", "owner", owner, "name", name, "contest_id", contest.ID)
	return contest, nil
}

func (s *contestService) GetContestsByOwnerPaginated(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error) {
	log := util.LoggerFromContext(ctx)

	// get paginated contests from repository
	contests, total, err := s.repo.GetAllByOwnerPaginated(ctx, owner, page, limit)
	if err != nil {
		log.Error("failed to get paginated contests by user", "owner", owner, "error", err)
		return nil, 0, err
	}

	log.Info("retrieved paginated contests by owner", "count", len(contests))
	return contests, total, nil
}

// ====================
// Contest Lifecycle Actions
// ====================

func (s *contestService) CreateContest(ctx context.Context, req *model.CreateContestRequest, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	// check if contest name already exists for user
	exists, err := s.repo.ExistsByOwnerAndName(ctx, req.Owner, req.Name)
	if err != nil {
		log.Error("failed to check if contest exists", "owner", req.Owner, "name", req.Name, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if exists {
		log.Warn("contest already exists", "owner", req.Owner, "name", req.Name)
		return nil, errs.ErrContestAlreadyExists
	}

	// build contest with initial labels
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

	// create contest in repository
	if err := s.repo.Create(ctx, &contest); err != nil {
		log.Error("failed to create contest in repository", "error", err)
		return nil, err
	}

	log.Info("created contest", "name", req.Name, "contest_id", contest.ID, "owner", req.Owner)
	return &contest, nil
}

func initLabels() (xLabelsJSON, yLabelsJSON []byte) {
	// initialize labels to -1 (unset)
	labels := make([]int8, 10)
	for i := range int8(10) {
		labels[i] = -1
	}

	xLabelsJSON, _ = json.Marshal(labels)
	yLabelsJSON, _ = json.Marshal(labels)

	return xLabelsJSON, yLabelsJSON
}

func (s *contestService) UpdateContest(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	// get contest and check authorization
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found for update", "contest_id", contestID)
			return nil, err
		}

		log.Error("failed to get contest for ownership validation", "contest_id", contestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if contest.Owner != user {
		log.Warn("user is not authorized to update contest", "contest_id", contestID, "owner", contest.Owner)
		return nil, errs.ErrUnauthorizedContestEdit
	}

	// check for changes and build update
	needsUpdate := false
	if req.HomeTeam != nil && *req.HomeTeam != contest.HomeTeam {
		contest.HomeTeam = *req.HomeTeam
		needsUpdate = true
	}

	if req.AwayTeam != nil && *req.AwayTeam != contest.AwayTeam {
		contest.AwayTeam = *req.AwayTeam
		needsUpdate = true
	}

	if !needsUpdate {
		log.Info("no changes detected for contest update", "contest_id", contest.ID)
		return contest, nil
	}

	// save updated contest
	contest.UpdatedBy = user
	if err := s.repo.Update(ctx, contest); err != nil {
		log.Error("failed to save updated contest", "contest_id", contest.ID, "error", err)
		return nil, err
	}

	// publish update to websocket clients
	go func() {
		if err := s.natsService.PublishContestUpdate(contest.ID, user, contest); err != nil {
			log.Error("failed to publish contest update", "contest_id", contest.ID, "error", err)
		}
	}()

	log.Info("contest updated successfully", "contest_id", contest.ID, "user", user)
	return contest, nil
}

func (s *contestService) StartContest(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	// get contest and validate status
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return nil, err
	}

	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot start contest - not in ACTIVE status", "contest_id", contestID, "current_status", contest.Status)
		return nil, errors.New("contest must be in ACTIVE status to start")
	}

	// transition to q1 and randomize labels
	if err := s.transitionToQ1(ctx, contest, user); err != nil {
		log.Error("failed to transition to Q1", "contest_id", contestID, "error", err)
		return nil, err
	}

	log.Info("contest started successfully", "contest_id", contestID)
	return contest, nil
}

func (s *contestService) transitionToQ1(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// randomize the x and y labels
	xLabels := generateRandomizedLabels()
	yLabels := generateRandomizedLabels()

	xLabelsJSON, err := json.Marshal(xLabels)
	if err != nil {
		log.Error("failed to marshal X labels", "contest_id", contest.ID, "error", err)
		return err
	}

	yLabelsJSON, err := json.Marshal(yLabels)
	if err != nil {
		log.Error("failed to marshal Y labels", "contest_id", contest.ID, "error", err)
		return err
	}

	contest.XLabels = xLabelsJSON
	contest.YLabels = yLabelsJSON
	contest.Status = model.ContestStatusQ1
	contest.UpdatedBy = user

	// save contest with randomized labels and updated status
	if err := s.repo.Update(ctx, contest); err != nil {
		log.Error("failed to save contest with randomized labels", "contest_id", contest.ID, "error", err)
		return err
	}

	// publish status change to websocket clients
	go func() {
		if err := s.natsService.PublishContestUpdate(contest.ID, user, contest); err != nil {
			log.Error("failed to publish contest update for Q1 transition", "contest_id", contest.ID, "error", err)
		}
	}()

	log.Info("transitioned to Q1, labels randomized, squares now immutable", "contest_id", contest.ID)
	return nil
}

func generateRandomizedLabels() []int8 {
	// create labels 0-9
	labels := make([]int8, 10)
	for i := range int8(10) {
		labels[i] = i
	}

	// fisher-yates shuffle
	for i := len(labels) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		labels[i], labels[j] = labels[j], labels[i]
	}

	return labels
}

func (s *contestService) RecordQuarterResult(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error) {
	log := util.LoggerFromContext(ctx)

	// get the contest to access labels and status
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return nil, err
	}

	// determine quarter and next status from current contest status
	var quarter int
	var nextStatus model.ContestStatus
	switch contest.Status {
	case model.ContestStatusQ1:
		quarter = 1
		nextStatus = model.ContestStatusQ2
	case model.ContestStatusQ2:
		quarter = 2
		nextStatus = model.ContestStatusQ3
	case model.ContestStatusQ3:
		quarter = 3
		nextStatus = model.ContestStatusQ4
	case model.ContestStatusQ4:
		quarter = 4
		nextStatus = model.ContestStatusFinished
	default:
		log.Warn("invalid contest status for recording quarter result", "status", contest.Status)
		return nil, errors.New("contest must be in Q1, Q2, Q3, or Q4 status to record quarter result")
	}

	// no duplicate quarter results
	for _, qr := range contest.QuarterResults {
		if qr.Quarter == quarter {
			log.Warn("quarter result already exists for given quarter", "quarter", quarter)
			return nil, errs.ErrQuarterResultAlreadyExists
		}
	}

	// parse labels
	var xLabels, yLabels []int8
	if err := json.Unmarshal(contest.XLabels, &xLabels); err != nil {
		log.Error("failed to unmarshal X labels", "contest_id", contestID, "error", err)
		return nil, err
	}
	if err := json.Unmarshal(contest.YLabels, &yLabels); err != nil {
		log.Error("failed to unmarshal Y labels", "contest_id", contestID, "error", err)
		return nil, err
	}

	// calculate winner coordinates
	winnerRow, winnerCol, err := calculateWinnerCoordinates(homeScore, awayScore, xLabels, yLabels)
	if err != nil {
		log.Error("failed to calculate winner coordinates", "contest_id", contestID, "error", err)
		return nil, err
	}

	// find the winning square and get owner details
	var winner, winnerName string
	for _, square := range contest.Squares {
		if square.Row == winnerRow && square.Col == winnerCol {
			winner = square.Owner
			winnerName = square.OwnerName
			break
		}
	}

	// create quarter result
	result := &model.QuarterResult{
		ContestID:     contestID,
		Quarter:       quarter,
		HomeTeamScore: homeScore,
		AwayTeamScore: awayScore,
		WinnerRow:     winnerRow,
		WinnerCol:     winnerCol,
		Winner:        winner,
		WinnerName:    winnerName,
	}

	if err := s.repo.CreateQuarterResult(ctx, result); err != nil {
		log.Error("failed to create quarter result", "contest_id", contestID, "quarter", quarter, "error", err)
		return nil, err
	}

	// transition contest status and publish update
	if err := s.transitionContestAfterQuarter(ctx, contest, nextStatus, result, user); err != nil {
		log.Error("failed to transition contest after quarter", "contest_id", contestID, "quarter", quarter, "error", err)
		return nil, err
	}

	log.Info("quarter result recorded and status transitioned", "contest_id", contestID, "quarter", quarter, "winner", winner, "new_status", nextStatus)
	return result, nil
}

func (s *contestService) transitionContestAfterQuarter(ctx context.Context, contest *model.Contest, newStatus model.ContestStatus, result *model.QuarterResult, user string) error {
	log := util.LoggerFromContext(ctx)

	// update contest status
	contest.Status = newStatus
	if err := s.repo.Update(ctx, contest); err != nil {
		log.Error("failed to update contest status", "contest_id", contest.ID, "new_status", newStatus, "error", err)
		return err
	}

	// publish quarter result to websocket clients
	go func() {
		if err := s.natsService.PublishQuarterResult(contest.ID, user, result); err != nil {
			log.Error("failed to publish quarter result", "contest_id", contest.ID, "quarter", result.Quarter, "error", err)
		}
	}()

	log.Info("contest transitioned after quarter", "contest_id", contest.ID, "quarter", result.Quarter, "new_status", newStatus)
	return nil
}

func calculateWinnerCoordinates(homeScore, awayScore int, xLabels, yLabels []int8) (row, col int, err error) {
	// get last digit of each score
	homeLastDigit := homeScore % 10
	awayLastDigit := awayScore % 10

	// find row (away team - y axis)
	row = -1
	for i, label := range yLabels {
		if int(label) == awayLastDigit {
			row = i
			break
		}
	}

	// find col (home team - x axis)
	col = -1
	for i, label := range xLabels {
		if int(label) == homeLastDigit {
			col = i
			break
		}
	}

	// validate both coordinates were found
	if row == -1 || col == -1 {
		return 0, 0, gorm.ErrInvalidData
	}

	return row, col, nil
}

func (s *contestService) DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error {
	log := util.LoggerFromContext(ctx)

	// get contest and verify ownership
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return err
	}

	if contest.Owner != user {
		log.Warn("unauthorized delete attempt", "contest_owner", contest.Owner)
		return errs.ErrUnauthorizedContestDelete
	}

	// delete contest from repository
	if err := s.repo.Delete(ctx, contestID); err != nil {
		log.Error("failed to delete contest from repository", "contest_id", contestID, "error", err)
		return err
	}

	go func() {
		if err := s.natsService.PublishContestDeleted(contestID, user); err != nil {
			log.Error("failed to publish contest deleted", "contest_id", contestID, "error", err)
		}
	}()

	log.Info("deleted contest successfully", "contest_id", contestID)
	return nil
}

// ====================
// Square Actions
// ====================

func (s *contestService) UpdateSquare(ctx context.Context, contestID uuid.UUID, squareID uuid.UUID, req *model.UpdateSquareRequest, user string) (*model.Square, error) {
	log := util.LoggerFromContext(ctx)

	// get contest to check status and find square
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest for square update", "contest_id", contestID, "error", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, errs.ErrDatabaseUnavailable
	}

	// check if contest is in editable state
	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot update square when contest is not active", "square_id", squareID, "contest_status", contest.Status)
		return nil, errs.ErrSquareNotEditable
	}

	// find square in contest
	var square *model.Square
	for i := range contest.Squares {
		if contest.Squares[i].ID == squareID {
			square = &contest.Squares[i]
			break
		}
	}

	if square == nil {
		log.Warn("square not found in contest", "square_id", squareID, "contest_id", contestID)
		return nil, gorm.ErrRecordNotFound
	}

	// check authorization
	if req.Owner != user {
		log.Warn("user not authorized to update square", "square_id", squareID, "requested_owner", req.Owner)
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	if square.Owner != "" && square.Owner != user {
		log.Warn("user not authorized to update square", "square_id", squareID, "owner", square.Owner, "requested_owner", req.Owner)
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	// get claims for first and last name
	claims := util.ClaimsFromContext(ctx)
	if claims == nil {
		log.Error("claims not found in context")
		return nil, errs.ErrClaimsNotFound
	}

	updatedSquare, err := s.repo.UpdateSquare(ctx, square, req.Value, req.Owner, claims.Name)
	if err != nil {
		log.Error("failed to update square", "square_id", square.ID, "value", req.Value, "owner", req.Owner, "error", err)
		return nil, err
	}

	go func() {
		if err := s.natsService.PublishSquareUpdate(contest.ID, user, updatedSquare); err != nil {
			log.Error("failed to publish square update", "contestId", updatedSquare.ContestID, "squareId", updatedSquare.ID, "error", err)
		}
	}()

	log.Info("square updated successfully", "square_id", square.ID, "value", req.Value, "owner", req.Owner)
	return updatedSquare, nil
}

func (s *contestService) ClearSquare(ctx context.Context, contestID uuid.UUID, squareID uuid.UUID, user string) (*model.Square, error) {
	log := util.LoggerFromContext(ctx)

	// get contest to check status and find square
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest for square clear", "contest_id", contestID, "error", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		return nil, errs.ErrDatabaseUnavailable
	}

	// check if contest is in editable state
	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot clear square when contest is not active", "square_id", squareID, "contest_status", contest.Status)
		return nil, errs.ErrSquareNotEditable
	}

	// find square in contest
	var square *model.Square
	for i := range contest.Squares {
		if contest.Squares[i].ID == squareID {
			square = &contest.Squares[i]
			break
		}
	}

	if square == nil {
		log.Warn("square not found in contest", "square_id", squareID, "contest_id", contestID)
		return nil, gorm.ErrRecordNotFound
	}

	// check authorization - allow if user is contest owner or square owner
	isContestOwner := contest.Owner == user
	isSquareOwner := square.Owner == user
	if !isContestOwner && !isSquareOwner {
		log.Warn("user not authorized to clear square", "square_id", squareID, "square_owner", square.Owner, "contest_owner", contest.Owner, "user", user)
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	clearedSquare, err := s.repo.ClearSquare(ctx, square)
	if err != nil {
		log.Error("failed to clear square", "square_id", square.ID, "error", err)
		return nil, err
	}

	go func() {
		if err := s.natsService.PublishSquareUpdate(contest.ID, user, clearedSquare); err != nil {
			log.Error("failed to publish square clear", "contestId", clearedSquare.ContestID, "squareId", clearedSquare.ID, "error", err)
		}
	}()

	log.Info("square cleared successfully", "square_id", square.ID)
	return clearedSquare, nil
}
