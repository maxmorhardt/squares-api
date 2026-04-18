package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type InviteRepository interface {
	GetByToken(ctx context.Context, token string) (*model.ContestInvite, error)
	GetAllByContestID(ctx context.Context, contestID uuid.UUID) ([]model.ContestInvite, error)
	Create(ctx context.Context, invite *model.ContestInvite) error
	RedeemInvite(ctx context.Context, inviteID uuid.UUID, participant *model.ContestParticipant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type inviteRepository struct {
	db *gorm.DB
}

func NewInviteRepository(db *gorm.DB) InviteRepository {
	return &inviteRepository{
		db: db,
	}
}

func (r *inviteRepository) GetByToken(ctx context.Context, token string) (*model.ContestInvite, error) {
	var invite model.ContestInvite
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&invite).Error
	return &invite, err
}

func (r *inviteRepository) GetAllByContestID(ctx context.Context, contestID uuid.UUID) ([]model.ContestInvite, error) {
	var invites []model.ContestInvite
	err := r.db.WithContext(ctx).Where("contest_id = ?", contestID).Order("created_at DESC").Find(&invites).Error
	return invites, err
}

func (r *inviteRepository) Create(ctx context.Context, invite *model.ContestInvite) error {
	return r.db.WithContext(ctx).Create(invite).Error
}

func (r *inviteRepository) RedeemInvite(ctx context.Context, inviteID uuid.UUID, participant *model.ContestParticipant) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(participant).Error; err != nil {
			return err
		}

		result := tx.Model(&model.ContestInvite{}).
			Where("id = ?", inviteID).
			UpdateColumn("uses", gorm.Expr("uses + 1"))
		if result.Error != nil {
			return result.Error
		}

		return nil
	})
}

func (r *inviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ContestInvite{}).Error
}
