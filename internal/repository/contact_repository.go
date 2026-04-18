package repository

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ContactRepository interface {
	Create(ctx context.Context, submission *model.ContactSubmission) error
}

type contactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) ContactRepository {
	return &contactRepository{
		db: db,
	}
}

func (r *contactRepository) Create(ctx context.Context, submission *model.ContactSubmission) error {
	return r.db.WithContext(ctx).Create(submission).Error
}
