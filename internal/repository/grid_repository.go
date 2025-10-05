package repository

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/db"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type GridRepository interface {
	Create(ctx context.Context, grid *model.Grid) error
	GetAll(ctx context.Context) ([]model.Grid, error)
	GetAllByUser(ctx context.Context, username string) ([]model.Grid, error)
	GetByID(ctx context.Context, id string) (model.Grid, error)
}

type gridRepository struct{}

func NewGridRepository() GridRepository {
	return &gridRepository{}
}

func (r *gridRepository) Create(ctx context.Context, grid *model.Grid) error {
	return db.DB.WithContext(ctx).Create(grid).Error
}

func (r *gridRepository) GetAll(ctx context.Context) ([]model.Grid, error) {
	var grids []model.Grid
	err := db.DB.WithContext(ctx).Find(&grids).Error
	return grids, err
}

func (r *gridRepository) GetAllByUser(ctx context.Context, username string) ([]model.Grid, error) {
	var grids []model.Grid
	err := db.DB.WithContext(ctx).Where("created_by = ?", username).Find(&grids).Error
	return grids, err
}

func (r *gridRepository) GetByID(ctx context.Context, id string) (model.Grid, error) {
	var grid model.Grid
	err := db.DB.WithContext(ctx).First(&grid, "id = ?", id).Error
	return grid, err
}