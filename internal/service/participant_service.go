package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ParticipantService interface {
	GetParticipants(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestParticipant, error)
	GetParticipantsInternal(ctx context.Context, contestID uuid.UUID) ([]model.ContestParticipant, error)
	GetMyContests(ctx context.Context, user, search string) ([]model.Contest, error)
	UpdateParticipant(ctx context.Context, contestID uuid.UUID, targetUserID string, req *model.UpdateParticipantRequest, user string) (*model.ContestParticipant, error)
	RemoveParticipant(ctx context.Context, contestID uuid.UUID, targetUserID, user string) error
	Authorize(ctx context.Context, contestID uuid.UUID, userID string, act Action) error
}

type participantService struct {
	participantRepo repository.ParticipantRepository
	contestRepo     repository.ContestRepository
	natsService     NatsService
}

func NewParticipantService(
	participantRepo repository.ParticipantRepository,
	contestRepo repository.ContestRepository,
	natsService NatsService,
) ParticipantService {
	return &participantService{
		participantRepo: participantRepo,
		contestRepo:     contestRepo,
		natsService:     natsService,
	}
}

// ====================
// Authorization
// ====================

type Action int

const (
	ActionView Action = iota
	ActionClaimSquare
	ActionEditContest
	ActionManageInvites
	ActionDeleteContest
)

var rolePermissions = map[model.ParticipantRole]map[Action]bool{
	model.ParticipantRoleOwner: {
		ActionView:          true,
		ActionClaimSquare:   true,
		ActionEditContest:   true,
		ActionManageInvites: true,
		ActionDeleteContest: true,
	},
	model.ParticipantRoleParticipant: {
		ActionView:        true,
		ActionClaimSquare: true,
	},
	model.ParticipantRoleViewer: {
		ActionView: true,
	},
}

func (s *participantService) Authorize(ctx context.Context, contestID uuid.UUID, userID string, act Action) error {
	log := util.LoggerFromContext(ctx)

	// for view actions, check if contest is public first
	if act == ActionView {
		visibility, err := s.contestRepo.GetVisibilityByID(ctx, contestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			log.Error("failed to get contest visibility for authorization", "contest_id", contestID, "error", err)
			return errs.ErrDatabaseUnavailable
		}

		if visibility == model.ContestVisibilityPublic {
			return nil
		}
	}

	participant, err := s.participantRepo.GetByContestAndUser(ctx, contestID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotParticipant
		}
		log.Error("failed to get participant for authorization", "contest_id", contestID, "user_id", userID, "error", err)
		return errs.ErrDatabaseUnavailable
	}

	perms, exists := rolePermissions[participant.Role]
	if !exists || !perms[act] {
		return errs.ErrInsufficientRole
	}

	return nil
}

// ====================
// Participant CRUD
// ====================

func (s *participantService) GetParticipants(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestParticipant, error) {
	if err := s.Authorize(ctx, contestID, user, ActionView); err != nil {
		return nil, err
	}

	return s.fetchParticipants(ctx, contestID)
}

func (s *participantService) GetParticipantsInternal(ctx context.Context, contestID uuid.UUID) ([]model.ContestParticipant, error) {
	return s.fetchParticipants(ctx, contestID)
}

func (s *participantService) fetchParticipants(ctx context.Context, contestID uuid.UUID) ([]model.ContestParticipant, error) {
	log := util.LoggerFromContext(ctx)

	participants, err := s.participantRepo.GetAllByContestID(ctx, contestID)
	if err != nil {
		log.Error("failed to get participants", "contest_id", contestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	log.Info("retrieved participants", "contest_id", contestID, "count", len(participants))
	return participants, nil
}

func (s *participantService) GetMyContests(ctx context.Context, user, search string) ([]model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	contests, err := s.contestRepo.GetAllByParticipantUserID(ctx, user, strings.TrimSpace(search))
	if err != nil {
		log.Error("failed to get user contests", "user", user, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	// game-linked contests read their quarter results from the shared game record
	for i := range contests {
		util.SynthesizeFromGame(&contests[i])
	}

	log.Info("retrieved joined contests", "count", len(contests))
	return contests, nil
}

func (s *participantService) UpdateParticipant(ctx context.Context, contestID uuid.UUID, targetUserID string, req *model.UpdateParticipantRequest, user string) (*model.ContestParticipant, error) {
	log := util.LoggerFromContext(ctx)

	// check contest is not in a terminal state
	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if contest.Status.IsTerminal() {
		log.Warn("cannot update participant in terminal state", "contest_id", contestID, "status", contest.Status)
		return nil, errs.ErrContestFinalized
	}

	// verify caller is owner
	if authErr := s.Authorize(ctx, contestID, user, ActionManageInvites); authErr != nil {
		return nil, authErr
	}

	// get the target participant
	participant, err := s.participantRepo.GetByContestAndUser(ctx, contestID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotParticipant
		}
		log.Error("failed to get participant", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	// cannot change the owner's role
	if participant.Role == model.ParticipantRoleOwner && req.Role != nil {
		return nil, errs.ErrCannotChangeOwner
	}

	if req.Role != nil {
		participant.Role = model.ParticipantRole(*req.Role)
	}

	// viewers always 0, participants must be >= 1, owners may be 0-100
	targetMax := participant.MaxSquares
	if req.MaxSquares != nil {
		targetMax = *req.MaxSquares
	}

	switch participant.Role {
	case model.ParticipantRoleViewer:
		// a viewer explicitly given squares is a client bug; reject it outright
		if req.MaxSquares != nil && *req.MaxSquares > 0 {
			return nil, errs.ErrViewerCannotHaveSquares
		}
		targetMax = 0
	case model.ParticipantRoleParticipant:
		if targetMax < 1 {
			return nil, errs.ErrInvalidSquareCount
		}
	case model.ParticipantRoleOwner:
		// owners may hold anywhere from 0 to 100 squares
	}

	if targetMax != participant.MaxSquares {
		// new limit can't be below currently claimed squares
		claimed, err := s.participantRepo.CountSquaresByUser(ctx, contestID, targetUserID)
		if err != nil {
			log.Error("failed to count squares by user", "contest_id", contestID, "user_id", targetUserID, "error", err)
			return nil, errs.ErrDatabaseUnavailable
		}

		if targetMax < claimed {
			return nil, errs.ErrSquareLimitTooLow
		}

		// total allocation across all participants (including owner) cannot exceed 100
		totalAllocated, err := s.participantRepo.GetTotalAllocatedSquares(ctx, contestID)
		if err != nil {
			log.Error("failed to get total allocated squares", "contest_id", contestID, "error", err)
			return nil, errs.ErrDatabaseUnavailable
		}

		newTotal := totalAllocated - participant.MaxSquares + targetMax
		if newTotal > 100 {
			return nil, errs.ErrNotEnoughSquares
		}

		participant.MaxSquares = targetMax
	}

	if err := s.participantRepo.Update(ctx, participant); err != nil {
		log.Error("failed to update participant", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return nil, err
	}

	log.Info("participant updated", "contest_id", contestID, "target_user", targetUserID)
	return participant, nil
}

func (s *participantService) RemoveParticipant(ctx context.Context, contestID uuid.UUID, targetUserID, user string) error {
	log := util.LoggerFromContext(ctx)

	// check contest is not in a terminal state
	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		log.Error("failed to get contest", "contest_id", contestID, "error", err)
		return errs.ErrDatabaseUnavailable
	}

	// participants can leave until the contest is finalized
	if contest.Status.IsTerminal() {
		log.Warn("cannot remove participant from a finalized contest", "contest_id", contestID, "status", contest.Status)
		return errs.ErrContestFinalized
	}

	// removing someone else requires owner permissions
	if targetUserID != user {
		if authErr := s.Authorize(ctx, contestID, user, ActionManageInvites); authErr != nil {
			return authErr
		}
	}

	// get the target participant
	participant, err := s.participantRepo.GetByContestAndUser(ctx, contestID, targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrNotParticipant
		}
		log.Error("failed to get participant", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return errs.ErrDatabaseUnavailable
	}

	// the owner cannot be removed by anyone, including themselves — they must delete the contest
	if participant.Role == model.ParticipantRoleOwner {
		return errs.ErrCannotRemoveOwner
	}

	// pre-kickoff squares are freed for others; in-progress squares are ghosted to keep scoring
	if contest.Status == model.ContestStatusActive {
		if err := s.releaseParticipantSquares(ctx, contestID, targetUserID, false); err != nil {
			log.Error("failed to clear participant squares", "contest_id", contestID, "user_id", targetUserID, "error", err)
			return err
		}
	} else {
		if err := s.releaseParticipantSquares(ctx, contestID, targetUserID, true); err != nil {
			log.Error("failed to ghost participant squares", "contest_id", contestID, "user_id", targetUserID, "error", err)
			return err
		}
	}

	// delete participant
	if err := s.participantRepo.Delete(ctx, contestID, targetUserID); err != nil {
		log.Error("failed to delete participant", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return err
	}

	metrics.IncParticipantRemoved()

	go func() {
		if err := s.natsService.PublishParticipantRemoved(contestID, user, participant); err != nil {
			log.Error("failed to publish participant removed", "contest_id", contestID, "user_id", targetUserID, "error", err)
		}
	}()

	log.Info("participant removed", "contest_id", contestID, "target_user", targetUserID)
	return nil
}

func (s *participantService) releaseParticipantSquares(ctx context.Context, contestID uuid.UUID, userID string, ghost bool) error {
	log := util.LoggerFromContext(ctx)

	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		return err
	}

	for i := range contest.Squares {
		if contest.Squares[i].Owner != userID {
			continue
		}

		var updatedSquare *model.Square
		if ghost {
			updatedSquare, err = s.contestRepo.GhostSquare(ctx, &contest.Squares[i])
		} else {
			updatedSquare, err = s.contestRepo.ClearSquare(ctx, &contest.Squares[i])
		}
		if err != nil {
			return err
		}

		go func() {
			if err := s.natsService.PublishSquareUpdate(contest.ID, userID, updatedSquare); err != nil {
				log.Error("failed to publish square update for removed participant", "contestId", contest.ID, "squareId", updatedSquare.ID, "error", err)
			}
		}()
	}

	return nil
}
