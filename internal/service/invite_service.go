package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

type InviteService interface {
	CreateInvite(ctx context.Context, contestID uuid.UUID, req *model.CreateInviteRequest, user string) (*model.ContestInvite, error)
	GetInvitePreview(ctx context.Context, token string) (*model.InvitePreviewResponse, error)
	RedeemInvite(ctx context.Context, token string, user string) (*model.ContestParticipant, error)
	GetInvitesByContestID(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestInvite, error)
	DeleteInvite(ctx context.Context, contestID uuid.UUID, inviteID uuid.UUID, user string) error
}

type inviteService struct {
	inviteRepo         repository.InviteRepository
	participantRepo    repository.ParticipantRepository
	contestRepo        repository.ContestRepository
	participantService ParticipantService
	natsService        NatsService
}

func NewInviteService(
	inviteRepo repository.InviteRepository,
	participantRepo repository.ParticipantRepository,
	contestRepo repository.ContestRepository,
	participantService ParticipantService,
	natsService NatsService,
) InviteService {
	return &inviteService{
		inviteRepo:         inviteRepo,
		participantRepo:    participantRepo,
		contestRepo:        contestRepo,
		participantService: participantService,
		natsService:        natsService,
	}
}

func (s *inviteService) CreateInvite(ctx context.Context, contestID uuid.UUID, req *model.CreateInviteRequest, user string) (*model.ContestInvite, error) {
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
		log.Warn("cannot create invite for contest in terminal state", "contest_id", contestID, "status", contest.Status)
		return nil, errs.ErrContestFinalized
	}

	// verify user has permission to manage invites
	if err := s.participantService.Authorize(ctx, contestID, user, ActionManageInvites); err != nil {
		return nil, err
	}

	// build invite
	invite := &model.ContestInvite{
		ContestID:  contestID,
		MaxSquares: req.MaxSquares,
		Role:       model.ParticipantRole(req.Role),
		CreatedBy:  user,
		MaxUses:    req.MaxUses,
	}

	if req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Minute)
		invite.ExpiresAt = &expiresAt
	}

	if err := s.inviteRepo.Create(ctx, invite); err != nil {
		log.Error("failed to create invite", "contest_id", contestID, "error", err)
		return nil, err
	}

	log.Info("invite created", "invite_id", invite.ID, "contest_id", contestID, "token", invite.Token)
	return invite, nil
}

func (s *inviteService) GetInvitePreview(ctx context.Context, token string) (*model.InvitePreviewResponse, error) {
	log := util.LoggerFromContext(ctx)

	invite, err := s.inviteRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrInviteNotFound
		}
		log.Error("failed to get invite by token", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if invite.IsExpired() {
		return nil, errs.ErrInviteExpired
	}

	if !invite.HasUsesRemaining() {
		return nil, errs.ErrInviteMaxUsesReached
	}

	contest, err := s.contestRepo.GetByID(ctx, invite.ContestID)
	if err != nil {
		log.Error("failed to get contest for invite preview", "contest_id", invite.ContestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return &model.InvitePreviewResponse{
		ContestName: contest.Name,
		Owner:       contest.Owner,
		Role:        string(invite.Role),
		MaxSquares:  invite.MaxSquares,
	}, nil
}

func (s *inviteService) RedeemInvite(ctx context.Context, token string, user string) (*model.ContestParticipant, error) {
	log := util.LoggerFromContext(ctx)

	// get and validate invite
	invite, err := s.inviteRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errs.ErrInviteNotFound
		}
		log.Error("failed to get invite by token", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if invite.IsExpired() {
		log.Warn("attempted to redeem expired invite", "invite_id", invite.ID)
		return nil, errs.ErrInviteExpired
	}

	if !invite.HasUsesRemaining() {
		log.Warn("attempted to redeem invite with no uses remaining", "invite_id", invite.ID)
		return nil, errs.ErrInviteMaxUsesReached
	}

	// check contest is not in a terminal state
	contest, err := s.contestRepo.GetByID(ctx, invite.ContestID)
	if err != nil {
		log.Error("failed to get contest for invite redemption", "contest_id", invite.ContestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if contest.Status.IsTerminal() {
		log.Warn("cannot redeem invite for contest in terminal state", "contest_id", invite.ContestID, "status", contest.Status)
		return nil, errs.ErrContestFinalized
	}

	// check user is not already a participant
	_, err = s.participantRepo.GetByContestAndUser(ctx, invite.ContestID, user)
	if err == nil {
		log.Warn("user already a participant", "contest_id", invite.ContestID, "user", user)
		return nil, errs.ErrAlreadyParticipant
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("failed to check existing participant", "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	// check total allocated squares won't exceed 100
	totalAllocated, err := s.participantRepo.GetTotalAllocatedSquares(ctx, invite.ContestID)
	if err != nil {
		log.Error("failed to get total allocated squares", "contest_id", invite.ContestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	if totalAllocated+invite.MaxSquares > 100 {
		log.Warn("not enough squares remaining", "contest_id", invite.ContestID, "allocated", totalAllocated, "requested", invite.MaxSquares)
		return nil, errs.ErrNotEnoughSquares
	}

	// create participant
	participant := &model.ContestParticipant{
		ContestID:  invite.ContestID,
		UserID:     user,
		Role:       invite.Role,
		MaxSquares: invite.MaxSquares,
		InviteID:   &invite.ID,
	}

	if err := s.participantRepo.Create(ctx, participant); err != nil {
		log.Error("failed to create participant", "contest_id", invite.ContestID, "user", user, "error", err)
		return nil, err
	}

	// increment invite usage
	if err := s.inviteRepo.IncrementUses(ctx, invite.ID); err != nil {
		log.Error("failed to increment invite uses", "invite_id", invite.ID, "error", err)
	}

	go func() {
		if err := s.natsService.PublishParticipantAdded(invite.ContestID, participant); err != nil {
			log.Error("failed to publish participant added", "contest_id", invite.ContestID, "user", user, "error", err)
		}
	}()

	log.Info("invite redeemed", "invite_id", invite.ID, "contest_id", invite.ContestID, "user", user)
	return participant, nil
}

func (s *inviteService) GetInvitesByContestID(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestInvite, error) {
	log := util.LoggerFromContext(ctx)

	// verify user has permission to manage invites
	if err := s.participantService.Authorize(ctx, contestID, user, ActionManageInvites); err != nil {
		return nil, err
	}

	invites, err := s.inviteRepo.GetAllByContestID(ctx, contestID)
	if err != nil {
		log.Error("failed to get invites", "contest_id", contestID, "error", err)
		return nil, errs.ErrDatabaseUnavailable
	}

	return invites, nil
}

func (s *inviteService) DeleteInvite(ctx context.Context, contestID uuid.UUID, inviteID uuid.UUID, user string) error {
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

	if contest.Status.IsTerminal() {
		log.Warn("cannot delete invite for contest in terminal state", "contest_id", contestID, "status", contest.Status)
		return errs.ErrContestFinalized
	}

	// verify user has permission to manage invites
	if err := s.participantService.Authorize(ctx, contestID, user, ActionManageInvites); err != nil {
		return err
	}

	if err := s.inviteRepo.Delete(ctx, inviteID); err != nil {
		log.Error("failed to delete invite", "invite_id", inviteID, "error", err)
		return err
	}

	log.Info("invite deleted", "invite_id", inviteID, "contest_id", contestID)
	return nil
}
