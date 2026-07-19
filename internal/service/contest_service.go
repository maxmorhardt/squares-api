package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ContestService interface {
	GetContestsByOwnerPaginated(ctx context.Context, owner string, page, limit int, search string) ([]model.Contest, int64, error)

	CreateContest(ctx context.Context, req *model.CreateContestRequest, user string) (*model.Contest, error)
	UpdateContest(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error)
	StartContest(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error)
	RecordQuarterResult(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error)
	RollbackLastQuarterResult(ctx context.Context, contestID uuid.UUID, user string) (*model.QuarterResult, error)
	DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error

	ClaimSquare(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error)
	ClearSquare(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error)
	ClearUserSquares(ctx context.Context, contestID uuid.UUID, user string) ([]model.Square, error)
}

type contestService struct {
	repo               repository.ContestRepository
	participantRepo    repository.ParticipantRepository
	gameRepo           repository.GameRepository
	userRepo           repository.UserRepository
	natsService        NatsService
	participantService ParticipantService
}

func NewContestService(
	repo repository.ContestRepository,
	participantRepo repository.ParticipantRepository,
	gameRepo repository.GameRepository,
	userRepo repository.UserRepository,
	natsService NatsService,
	participantService ParticipantService,
) ContestService {
	return &contestService{
		repo:               repo,
		participantRepo:    participantRepo,
		gameRepo:           gameRepo,
		userRepo:           userRepo,
		natsService:        natsService,
		participantService: participantService,
	}
}

// ====================
// Getters
// ====================

func (s *contestService) GetContestsByOwnerPaginated(ctx context.Context, owner string, page, limit int, search string) ([]model.Contest, int64, error) {
	log := util.LoggerFromContext(ctx)

	contests, total, err := s.repo.GetAllByOwnerPaginated(ctx, owner, page, limit, search)
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
	xLabelsJSON, yLabelsJSON := util.InitialLabels()
	visibility := model.ContestVisibilityPrivate
	if req.Visibility == "public" {
		visibility = model.ContestVisibilityPublic
	}

	contest := model.Contest{
		Name:       req.Name,
		XLabels:    xLabelsJSON,
		YLabels:    yLabelsJSON,
		HomeTeam:   req.HomeTeam,
		AwayTeam:   req.AwayTeam,
		Owner:      req.Owner,
		Visibility: visibility,
		Status:     model.ContestStatusActive,
	}

	// game-linked contest scores automatically and takes its teams from the game
	if req.GameID != "" {
		gameID, parseErr := uuid.Parse(req.GameID)
		if parseErr != nil {
			return nil, errs.ErrGameNotFound
		}
		game, gameErr := s.gameRepo.GetByID(ctx, gameID)
		if gameErr != nil {
			if errors.Is(gameErr, gorm.ErrRecordNotFound) {
				return nil, errs.ErrGameNotFound
			}

			log.Error("failed to get game for contest link", "game_id", gameID, "error", gameErr)
			return nil, errs.ErrDatabaseUnavailable
		}

		// set the foreign key and team names
		contest.GameID = &game.ID
		contest.HomeTeam = game.HomeTeam
		contest.AwayTeam = game.AwayTeam
	}

	// atomically create contest, squares, and owner participant
	ownerParticipant := &model.ContestParticipant{
		UserID:     user,
		Role:       model.ParticipantRoleOwner,
		MaxSquares: req.MaxSquares,
	}
	if err := s.repo.Create(ctx, &contest, ownerParticipant); err != nil {
		log.Error("failed to create contest with owner participant", "error", err)
		return nil, err
	}

	metrics.IncContestCreated()
	metrics.IncParticipantJoined(string(model.ParticipantRoleOwner))
	log.Info("created contest", "name", req.Name, "contest_id", contest.ID, "owner", req.Owner)
	return &contest, nil
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

	// game-linked contests read their quarter results from the shared game record
	util.SynthesizeFromGame(contest)

	if contest.Status.IsTerminal() {
		log.Warn("cannot update contest in terminal state", "contest_id", contestID, "status", contest.Status)
		return nil, errs.ErrContestFinalized
	}

	if err := s.participantService.Authorize(ctx, contestID, user, ActionEditContest); err != nil {
		log.Warn("user is not authorized to update contest", "contest_id", contestID, "user", user)
		return nil, errs.ErrUnauthorizedContestEdit
	}

	// game-linked contests take their team names from the game record
	canEditTeams := contest.GameID == nil

	// check for changes and build update
	needsUpdate := false
	if canEditTeams && req.HomeTeam != nil && *req.HomeTeam != contest.HomeTeam {
		contest.HomeTeam = *req.HomeTeam
		needsUpdate = true
	}

	if canEditTeams && req.AwayTeam != nil && *req.AwayTeam != contest.AwayTeam {
		contest.AwayTeam = *req.AwayTeam
		needsUpdate = true
	}

	// visibility is editable regardless of game link, as long as the contest isn't terminal
	if req.Visibility != nil && model.ContestVisibility(*req.Visibility) != contest.Visibility {
		contest.Visibility = model.ContestVisibility(*req.Visibility)
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
		// create a lightweight copy of the contest to avoid sending large preloaded relations
		wsContest := *contest
		wsContest.Squares = nil
		wsContest.QuarterResults = nil

		if err := s.natsService.PublishContestUpdate(contest.ID, user, &wsContest); err != nil {
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

	// game-linked contests start automatically at kickoff; owners can't start them
	if contest.GameID != nil {
		log.Warn("cannot manually start a game-linked contest", "contest_id", contestID)
		return nil, errs.ErrContestIsGameLinked
	}

	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot start contest, not in ACTIVE status", "contest_id", contestID, "current_status", contest.Status)
		return nil, errors.New("contest must be in ACTIVE status to start")
	}

	if !util.AllSquaresClaimed(contest) {
		log.Warn("cannot start contest - unclaimed squares remain", "contest_id", contestID)
		return nil, errs.ErrContestNotReady
	}

	// transition to q1 and randomize labels
	if err := s.transitionToQ1(ctx, contest, user); err != nil {
		log.Error("failed to transition to Q1", "contest_id", contestID, "error", err)
		return nil, err
	}

	metrics.IncContestStarted()
	log.Info("contest started successfully", "contest_id", contestID)
	return contest, nil
}

func (s *contestService) transitionToQ1(ctx context.Context, contest *model.Contest, user string) error {
	log := util.LoggerFromContext(ctx)

	// randomize the x and y labels
	xLabels, yLabels, err := util.RandomizedLabels()
	if err != nil {
		log.Error("failed to generate randomized labels", "contest_id", contest.ID, "error", err)
		return err
	}

	contest.XLabels = xLabels
	contest.YLabels = yLabels
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

func (s *contestService) RecordQuarterResult(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error) {
	log := util.LoggerFromContext(ctx)

	// get the contest to access labels and status
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return nil, err
	}

	// game-linked contests are scored automatically; they exit before the auth check so they're unaffected by it
	if contest.GameID != nil {
		log.Warn("cannot manually record quarter result for a game-linked contest", "contest_id", contestID)
		return nil, errs.ErrContestIsGameLinked
	}

	// only the contest owner may record scores
	if err = s.participantService.Authorize(ctx, contestID, user, ActionEditContest); err != nil {
		log.Warn("user is not authorized to record quarter result", "contest_id", contestID, "user", user)
		return nil, errs.ErrUnauthorizedContestEdit
	}

	// determine quarter and next status from current contest status
	quarter, ok := contest.Status.Quarter()
	if !ok {
		log.Warn("invalid contest status for recording quarter result", "contest_id", contestID, "status", contest.Status)
		return nil, errors.New("contest must be in Q1, Q2, Q3, or Q4 status to record quarter result")
	}
	nextStatus, _ := model.StatusAfterQuarter(quarter)

	// no duplicate quarter results
	for i := range contest.QuarterResults {
		if contest.QuarterResults[i].Quarter == quarter {
			log.Warn("quarter result already exists for given quarter", "quarter", quarter)
			return nil, errs.ErrQuarterResultAlreadyExists
		}
	}

	// compute the winning square from this contest's labels
	result, err := util.QuarterResultFor(contest, quarter, homeScore, awayScore)
	if err != nil {
		log.Error("failed to compute quarter result", "contest_id", contestID, "error", err)
		return nil, err
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

	metrics.IncQuarterResult(quarter)
	log.Info("quarter result recorded and status transitioned", "contest_id", contestID, "quarter", quarter, "winner", result.Winner, "new_status", nextStatus)
	return result, nil
}

func (s *contestService) transitionContestAfterQuarter(ctx context.Context, contest *model.Contest, newStatus model.ContestStatus, result *model.QuarterResult, user string) error {
	log := util.LoggerFromContext(ctx)

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

func (s *contestService) RollbackLastQuarterResult(ctx context.Context, contestID uuid.UUID, user string) (*model.QuarterResult, error) {
	log := util.LoggerFromContext(ctx)

	// get the contest to access status and recorded results
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return nil, err
	}

	// game-linked contests are scored automatically; they exit before the auth check so they're unaffected by it
	if contest.GameID != nil {
		log.Warn("cannot roll back quarter result for a game-linked contest", "contest_id", contestID)
		return nil, errs.ErrContestIsGameLinked
	}

	// only the contest owner may roll back scores
	if err = s.participantService.Authorize(ctx, contestID, user, ActionEditContest); err != nil {
		log.Warn("user is not authorized to roll back quarter result", "contest_id", contestID, "user", user)
		return nil, errs.ErrUnauthorizedContestEdit
	}

	// determine the most recently recorded quarter and the status to revert to
	revertStatus, quarter, ok := model.PreviousQuarterStatus(contest.Status)
	if !ok {
		log.Warn("no quarter result to roll back for current status", "contest_id", contestID, "status", contest.Status)
		return nil, errs.ErrNoQuarterResultToRollback
	}

	// find the recorded result for that quarter
	var result *model.QuarterResult
	for i := range contest.QuarterResults {
		if contest.QuarterResults[i].Quarter == quarter {
			result = &contest.QuarterResults[i]
			break
		}
	}
	if result == nil {
		log.Warn("quarter result missing for rollback", "contest_id", contestID, "quarter", quarter)
		return nil, errs.ErrNoQuarterResultToRollback
	}

	// delete the result and revert the contest status
	if err := s.repo.DeleteQuarterResult(ctx, result.ID); err != nil {
		log.Error("failed to delete quarter result", "contest_id", contestID, "quarter", quarter, "error", err)
		return nil, err
	}

	contest.Status = revertStatus
	contest.UpdatedBy = user
	if err := s.repo.Update(ctx, contest); err != nil {
		log.Error("failed to revert contest status after rollback", "contest_id", contestID, "quarter", quarter, "error", err)
		return nil, err
	}

	// publish the rollback so connected clients drop the quarter and revert status
	go func() {
		// send a lightweight contest copy without large preloaded relations
		wsContest := *contest
		wsContest.Squares = nil
		wsContest.QuarterResults = nil

		if err := s.natsService.PublishQuarterResultRollback(contest.ID, user, result, &wsContest); err != nil {
			log.Error("failed to publish quarter result rollback", "contest_id", contest.ID, "quarter", quarter, "error", err)
		}
	}()

	metrics.IncQuarterResultRolledBack(quarter)
	log.Info("quarter result rolled back and status reverted", "contest_id", contestID, "quarter", quarter, "new_status", revertStatus)
	return result, nil
}

func (s *contestService) DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error {
	log := util.LoggerFromContext(ctx)

	// check contest is not in a terminal state
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		log.Error("failed to get contest for delete", "contest_id", contestID, "error", err)
		return errs.ErrDatabaseUnavailable
	}

	if contest.Status.IsTerminal() {
		log.Warn("cannot delete contest in terminal state", "contest_id", contestID, "status", contest.Status)
		return errs.ErrContestFinalized
	}

	// verify authorization
	if err := s.participantService.Authorize(ctx, contestID, user, ActionDeleteContest); err != nil {
		log.Warn("unauthorized delete attempt", "contest_id", contestID, "user", user)
		return errs.ErrUnauthorizedContestDelete
	}

	if err := s.repo.Delete(ctx, contestID); err != nil {
		log.Error("failed to delete contest from repository", "contest_id", contestID, "error", err)
		return err
	}

	go func() {
		if err := s.natsService.PublishContestDeleted(contestID, user); err != nil {
			log.Error("failed to publish contest deleted", "contest_id", contestID, "error", err)
		}
	}()

	metrics.IncContestDeleted()
	log.Info("deleted contest successfully", "contest_id", contestID)
	return nil
}

// ====================
// Square Actions
// ====================

func (s *contestService) ClaimSquare(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error) {
	log := util.LoggerFromContext(ctx)

	// get contest to check status and find square
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest for square claim", "contest_id", contestID, "error", err)
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

	// check role-based permission to claim squares
	if err = s.participantService.Authorize(ctx, contestID, user, ActionClaimSquare); err != nil {
		log.Warn("user not authorized to claim squares", "contest_id", contestID, "user", user)
		return nil, errs.ErrUnauthorizedSquareEdit
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

	// a claimed square can only be re-claimed by its owner
	if square.Owner != "" && square.Owner != user {
		log.Warn("user not authorized to update square", "square_id", squareID, "owner", square.Owner, "user", user)
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	// enforce square limit for the participant
	participant, err := s.participantRepo.GetByContestAndUser(ctx, contestID, user)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("user is not a participant", "contest_id", contestID, "user", user)
			return nil, errs.ErrNotParticipant
		}
		log.Error("failed to get participant for square limit check", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	// capture whether the square was unclaimed before the update mutates square.Owner
	wasUnclaimed := square.Owner == ""

	// only check limit if claiming a new square (not re-editing own square)
	if wasUnclaimed {
		claimed, countErr := s.participantRepo.CountSquaresByUser(ctx, contestID, user)
		if countErr != nil {
			log.Error("failed to count user squares", "contest_id", contestID, "user", user, "error", countErr)
			return nil, errs.ErrDatabaseUnavailable
		}

		if claimed >= participant.MaxSquares {
			log.Warn("user has reached square limit", "contest_id", contestID, "user", user, "claimed", claimed, "limit", participant.MaxSquares)
			return nil, errs.ErrSquareLimitReached
		}
	}

	// get claims so we can capture the owner's display name
	claims := util.ClaimsFromContext(ctx)
	if claims == nil {
		log.Error("claims not found in context")
		return nil, errs.ErrClaimsNotFound
	}

	// the square value is the claimant's profile default initials, seeded from their name on first visit
	profile, err := s.userRepo.GetOrCreate(ctx, user, claims.Name, util.InitialsFromName(claims.Name))
	if err != nil {
		log.Error("failed to load profile for square claim", "user", user, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if profile.DefaultInitials == "" {
		log.Warn("user has no default initials set", "user", user)
		return nil, errs.ErrMissingInitials
	}

	claimedSquare, err := s.repo.ClaimSquare(ctx, square, profile.DefaultInitials, user, claims.Name)
	if err != nil {
		log.Error("failed to claim square", "square_id", square.ID, "value", profile.DefaultInitials, "owner", user, "error", err)
		return nil, err
	}

	if wasUnclaimed {
		metrics.IncSquareClaimed()
	}

	go func() {
		if err := s.natsService.PublishSquareUpdate(contest.ID, user, claimedSquare); err != nil {
			log.Error("failed to publish square update", "contestId", claimedSquare.ContestID, "squareId", claimedSquare.ID, "error", err)
		}
	}()

	log.Info("square claimed successfully", "square_id", square.ID, "value", profile.DefaultInitials, "owner", user)
	return claimedSquare, nil
}

func (s *contestService) ClearSquare(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error) {
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

	// check authorization - allow if user can edit contest or is the square owner
	isSquareOwner := square.Owner == user
	editErr := s.participantService.Authorize(ctx, contestID, user, ActionEditContest)
	if editErr != nil && !isSquareOwner {
		log.Warn("user not authorized to clear square", "square_id", squareID, "square_owner", square.Owner, "user", user)
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	clearedSquare, err := s.repo.ClearSquare(ctx, square)
	if err != nil {
		log.Error("failed to clear square", "square_id", square.ID, "error", err)
		return nil, err
	}

	metrics.IncSquareCleared()

	go func() {
		if err := s.natsService.PublishSquareUpdate(contest.ID, user, clearedSquare); err != nil {
			log.Error("failed to publish square clear", "contestId", clearedSquare.ContestID, "squareId", clearedSquare.ID, "error", err)
		}
	}()

	log.Info("square cleared successfully", "square_id", square.ID)
	return clearedSquare, nil
}

func (s *contestService) ClearUserSquares(ctx context.Context, contestID uuid.UUID, user string) ([]model.Square, error) {
	log := util.LoggerFromContext(ctx)

	// get contest to check status
	contest, err := s.repo.GetByID(ctx, contestID)
	if err != nil {
		log.Error("failed to get contest for clearing user squares", "contest_id", contestID, "error", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		return nil, errs.ErrDatabaseUnavailable
	}

	// check if contest is in editable state
	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot clear squares when contest is not active", "contest_id", contestID, "contest_status", contest.Status)
		return nil, errs.ErrSquareNotEditable
	}

	clearedSquares, err := s.repo.ClearSquaresByOwner(ctx, contestID, user)
	if err != nil {
		log.Error("failed to clear user squares", "contest_id", contestID, "user", user, "error", err)
		return nil, err
	}

	for i := range clearedSquares {
		metrics.IncSquareCleared()

		square := clearedSquares[i]
		go func() {
			if err := s.natsService.PublishSquareUpdate(contest.ID, user, &square); err != nil {
				log.Error("failed to publish square clear", "contestId", square.ContestID, "squareId", square.ID, "error", err)
			}
		}()
	}

	log.Info("user squares cleared successfully", "contest_id", contestID, "user", user, "count", len(clearedSquares))
	return clearedSquares, nil
}
