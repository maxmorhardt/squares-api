package repository

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/db"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type GridRepository interface {
	Create(ctx context.Context, grid *model.Grid) error
}

type gridRepository struct{}

func NewGridRepository() GridRepository {
	return &gridRepository{}
}

func (r *gridRepository) Create(ctx context.Context, grid *model.Grid) error {
	return db.DB.WithContext(ctx).Create(grid).Error
}
