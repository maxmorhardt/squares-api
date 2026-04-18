package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ParticipantRepository interface {
	GetByContestAndUser(ctx context.Context, contestID uuid.UUID, userID string) (*model.ContestParticipant, error)
	GetAllByContestID(ctx context.Context, contestID uuid.UUID) ([]model.ContestParticipant, error)
	GetAllByUserID(ctx context.Context, userID string) ([]model.ContestParticipant, error)
	GetTotalAllocatedSquares(ctx context.Context, contestID uuid.UUID) (int, error)
	CountSquaresByUser(ctx context.Context, contestID uuid.UUID, userID string) (int, error)
	Create(ctx context.Context, participant *model.ContestParticipant) error
	Update(ctx context.Context, participant *model.ContestParticipant) error
	Delete(ctx context.Context, contestID uuid.UUID, userID string) error
}

type participantRepository struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepository{
		db: db,
	}
}

func (r *participantRepository) GetByContestAndUser(ctx context.Context, contestID uuid.UUID, userID string) (*model.ContestParticipant, error) {
	var participant model.ContestParticipant
	err := r.db.WithContext(ctx).
		Where("contest_id = ? AND user_id = ?", contestID, userID).
		First(&participant).Error
	return &participant, err
}

func (r *participantRepository) GetAllByContestID(ctx context.Context, contestID uuid.UUID) ([]model.ContestParticipant, error) {
	var participants []model.ContestParticipant
	err := r.db.WithContext(ctx).
		Where("contest_id = ?", contestID).
		Order("joined_at ASC").
		Find(&participants).Error
	return participants, err
}

func (r *participantRepository) GetAllByUserID(ctx context.Context, userID string) ([]model.ContestParticipant, error) {
	var participants []model.ContestParticipant
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role != ?", userID, model.ParticipantRoleOwner).
		Order("joined_at DESC").
		Find(&participants).Error
	return participants, err
}

func (r *participantRepository) GetTotalAllocatedSquares(ctx context.Context, contestID uuid.UUID) (int, error) {
	var total int
	err := r.db.WithContext(ctx).
		Model(&model.ContestParticipant{}).
		Where("contest_id = ? AND role != ?", contestID, model.ParticipantRoleOwner).
		Select("COALESCE(SUM(max_squares), 0)").
		Row().
		Scan(&total)
	return total, err
}

func (r *participantRepository) CountSquaresByUser(ctx context.Context, contestID uuid.UUID, userID string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Square{}).
		Where("contest_id = ? AND owner = ? AND value != ''", contestID, userID).
		Count(&count).Error
	return int(count), err
}

func (r *participantRepository) Create(ctx context.Context, participant *model.ContestParticipant) error {
	return r.db.WithContext(ctx).Create(participant).Error
}

func (r *participantRepository) Update(ctx context.Context, participant *model.ContestParticipant) error {
	return r.db.WithContext(ctx).Save(participant).Error
}

func (r *participantRepository) Delete(ctx context.Context, contestID uuid.UUID, userID string) error {
	return r.db.WithContext(ctx).
		Where("contest_id = ? AND user_id = ?", contestID, userID).
		Delete(&model.ContestParticipant{}).Error
}
