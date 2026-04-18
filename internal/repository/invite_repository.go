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
	IncrementUses(ctx context.Context, inviteID uuid.UUID) error
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

func (r *inviteRepository) IncrementUses(ctx context.Context, inviteID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.ContestInvite{}).
		Where("id = ?", inviteID).
		UpdateColumn("uses", gorm.Expr("uses + 1")).Error
}

func (r *inviteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ContestInvite{}).Error
}
