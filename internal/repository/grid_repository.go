package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/db"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type GridRepository interface {
	Create(ctx context.Context, grid *model.Grid) error
	GetAll(ctx context.Context) ([]model.Grid, error)
	GetAllByUser(ctx context.Context, username string) ([]model.Grid, error)
	GetByID(ctx context.Context, id string) (model.Grid, error)
	CreateCells(ctx context.Context, cells []model.GridCell) error
	UpdateCell(ctx context.Context, gridID uuid.UUID, row, col int, value, user string) error
}

type gridRepository struct {
	db *gorm.DB
}

func NewGridRepository() GridRepository {
	return &gridRepository{
		db: db.DB,
	}
}

func (r *gridRepository) Create(ctx context.Context, grid *model.Grid) error {
	if err := r.db.WithContext(ctx).Create(grid).Error; err != nil {
		return err
	}

	var cells []model.GridCell
	for row := range 10 {
		for col := range 10 {
			cells = append(cells, model.GridCell{
				GridID: grid.ID,
				Row:    row,
				Col:    col,
				Value:  "",
			})
		}
	}

	return r.CreateCells(ctx, cells)
}

func (r *gridRepository) GetAll(ctx context.Context) ([]model.Grid, error) {
	var grids []model.Grid
	err := r.db.WithContext(ctx).
		Preload("Cells").
		Find(&grids).Error

	return grids, err
}

func (r *gridRepository) GetAllByUser(ctx context.Context, username string) ([]model.Grid, error) {
	var grids []model.Grid
	err := r.db.WithContext(ctx).
		Preload("Cells").
		Where("created_by = ?", username).
		Find(&grids).Error

	return grids, err
}


func (r *gridRepository) GetByID(ctx context.Context, id string) (model.Grid, error) {
	var grid model.Grid
	err := r.db.WithContext(ctx).
		Preload("Cells").
		First(&grid, "id = ?", id).Error
		
	return grid, err
}


func (r *gridRepository) CreateCells(ctx context.Context, cells []model.GridCell) error {
	return r.db.WithContext(ctx).Create(&cells).Error
}

func (r *gridRepository) UpdateCell(ctx context.Context, gridID uuid.UUID, row, col int, value, user string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cell model.GridCell
		if err := tx.Where("grid_id = ? AND row = ? AND col = ?", gridID, row, col).First(&cell).Error; err != nil {
			return fmt.Errorf("cell not found: %w", err)
		}

		cell.Value = value
		if user != "" {
			cell.UpdatedBy = user
		}

		if err := tx.Save(&cell).Error; err != nil {
			return fmt.Errorf("failed to update cell: %w", err)
		}

		return nil
	})
}