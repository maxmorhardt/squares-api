package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ContactRepository interface {
	Create(ctx context.Context, submission *model.ContactSubmission) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.ContactSubmission, error)
	Update(ctx context.Context, submission *model.ContactSubmission) error
}

type contactRepository struct {
	db *gorm.DB
}

func NewContactRepository() ContactRepository {
	return &contactRepository{
		db: config.DB(),
	}
}

func (r *contactRepository) Create(ctx context.Context, submission *model.ContactSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}

func (r *contactRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ContactSubmission, error) {
	var submission model.ContactSubmission
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&submission).Error
	return &submission, err
}

func (r *contactRepository) Update(ctx context.Context, submission *model.ContactSubmission) error {
	return r.db.WithContext(ctx).Save(submission).Error
}
