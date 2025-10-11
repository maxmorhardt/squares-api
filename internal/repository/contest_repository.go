package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ContestRepository interface {
	Create(ctx context.Context, contest *model.Contest) error
	GetAll(ctx context.Context) ([]model.Contest, error)
	GetAllByUser(ctx context.Context, username string) ([]model.Contest, error)
	GetByID(ctx context.Context, id string) (model.Contest, error)
	CreateSquares(ctx context.Context, squares []model.Square) error
	UpdateSquare(ctx context.Context, squareID uuid.UUID, value, user string) (model.Square, error)
}

type contestRepository struct {
	db *gorm.DB
}

func NewContestRepository() ContestRepository {
	return &contestRepository{
		db: config.DB,
	}
}

func (r *contestRepository) Create(ctx context.Context, contest *model.Contest) error {
	if err := r.db.WithContext(ctx).Create(contest).Error; err != nil {
		return err
	}

	var squares []model.Square
	for row := range 10 {
		for col := range 10 {
			squares = append(squares, model.Square{
				ContestID: contest.ID,
				Row:       row,
				Col:       col,
				Value:     "",
			})
		}
	}

	return r.CreateSquares(ctx, squares)
}

func (r *contestRepository) GetAll(ctx context.Context) ([]model.Contest, error) {
	var contests []model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Find(&contests).Error

	return contests, err
}

func (r *contestRepository) GetAllByUser(ctx context.Context, username string) ([]model.Contest, error) {
	var contests []model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Where("created_by = ?", username).
		Find(&contests).Error

	return contests, err
}

func (r *contestRepository) GetByID(ctx context.Context, id string) (model.Contest, error) {
	var contest model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		First(&contest, "id = ?", id).Error

	return contest, err
}

func (r *contestRepository) CreateSquares(ctx context.Context, squares []model.Square) error {
	return r.db.WithContext(ctx).Create(&squares).Error
}

func (r *contestRepository) UpdateSquare(ctx context.Context, squareID uuid.UUID, value, user string) (model.Square, error) {
	var updatedSquare model.Square

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var square model.Square
		if err := tx.Where("id = ?", squareID).First(&square).Error; err != nil {
			return err
		}

		square.Value = value
		if user != "" {
			square.Owner = user
		}

		if err := tx.Save(&square).Error; err != nil {
			return err
		}

		updatedSquare = square
		return nil
	})

	return updatedSquare, err
}
