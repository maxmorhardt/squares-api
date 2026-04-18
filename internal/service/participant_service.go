package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type ParticipantService interface {
	GetParticipants(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestParticipant, error)
	GetMyContests(ctx context.Context, user string) ([]model.Contest, error)
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
		contest, err := s.contestRepo.GetByID(ctx, contestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			log.Error("failed to get contest for authorization", "contest_id", contestID, "error", err)
			return errs.ErrDatabaseUnavailable
		}

		if contest.Visibility == model.ContestVisibilityPublic {
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
	log := util.LoggerFromContext(ctx)

	// any participant can view the participant list
	if err := s.Authorize(ctx, contestID, user, ActionView); err != nil {
		return nil, err
	}

	participants, err := s.participantRepo.GetAllByContestID(ctx, contestID)
	if err != nil {
		log.Error("failed to get participants", "contest_id", contestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return participants, nil
}

func (s *participantService) GetMyContests(ctx context.Context, user string) ([]model.Contest, error) {
	log := util.LoggerFromContext(ctx)

	participants, err := s.participantRepo.GetAllByUserID(ctx, user)
	if err != nil {
		log.Error("failed to get user participations", "user", user, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	var contests []model.Contest
	for i := range participants {
		contest, err := s.contestRepo.GetByID(ctx, participants[i].ContestID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			log.Error("failed to get contest", "contest_id", participants[i].ContestID, "error", err)
			continue
		}
		contests = append(contests, *contest)
	}

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
	if err := s.Authorize(ctx, contestID, user, ActionManageInvites); err != nil {
		return nil, err
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
	if participant.Role == model.ParticipantRoleOwner {
		return nil, errs.ErrCannotChangeOwner
	}

	if req.Role != nil {
		participant.Role = model.ParticipantRole(*req.Role)
	}

	if req.MaxSquares != nil {
		// new limit can't be below currently claimed squares
		claimed, err := s.participantRepo.CountSquaresByUser(ctx, contestID, targetUserID)
		if err != nil {
			log.Error("failed to count squares by user", "contest_id", contestID, "user_id", targetUserID, "error", err)
			return nil, errs.ErrDatabaseUnavailable
		}

		if *req.MaxSquares < claimed {
			return nil, errs.ErrSquareLimitTooLow
		}

		participant.MaxSquares = *req.MaxSquares
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

	if contest.Status != model.ContestStatusActive {
		log.Warn("cannot remove participant after contest has started", "contest_id", contestID, "status", contest.Status)
		return errs.ErrContestNotEditable
	}

	// verify caller is owner
	if err := s.Authorize(ctx, contestID, user, ActionManageInvites); err != nil {
		return err
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

	// cannot remove the owner
	if participant.Role == model.ParticipantRoleOwner {
		return errs.ErrCannotRemoveOwner
	}

	// clear the participant's squares
	if err := s.clearParticipantSquares(ctx, contestID, targetUserID); err != nil {
		log.Error("failed to clear participant squares", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return err
	}

	// delete participant
	if err := s.participantRepo.Delete(ctx, contestID, targetUserID); err != nil {
		log.Error("failed to delete participant", "contest_id", contestID, "user_id", targetUserID, "error", err)
		return err
	}

	go func() {
		if err := s.natsService.PublishParticipantRemoved(contestID, user, participant); err != nil {
			log.Error("failed to publish participant removed", "contest_id", contestID, "user_id", targetUserID, "error", err)
		}
	}()

	log.Info("participant removed", "contest_id", contestID, "target_user", targetUserID)
	return nil
}

func (s *participantService) clearParticipantSquares(ctx context.Context, contestID uuid.UUID, userID string) error {
	log := util.LoggerFromContext(ctx)

	contest, err := s.contestRepo.GetByID(ctx, contestID)
	if err != nil {
		return err
	}

	for i := range contest.Squares {
		if contest.Squares[i].Owner == userID {
			clearedSquare, err := s.contestRepo.ClearSquare(ctx, &contest.Squares[i])
			if err != nil {
				return err
			}
			go func() {
				if err := s.natsService.PublishSquareUpdate(contest.ID, userID, clearedSquare); err != nil {
					log.Error("failed to publish square clear for removed participant", "contestId", contest.ID, "squareId", clearedSquare.ID, "error", err)
				}
			}()
		}
	}

	return nil
}
