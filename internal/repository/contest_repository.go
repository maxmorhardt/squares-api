package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ContestRepository interface {
	GetAll(ctx context.Context) ([]model.Contest, error)
	Create(ctx context.Context, contest *model.Contest) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error)
	UpdateLabels(ctx context.Context, contestID uuid.UUID, xLabels, yLabels []int8, user string) (*model.Contest, error)
	UpdateSquare(ctx context.Context, squareID uuid.UUID, value, user string) (*model.Square, error)
	GetAllByUser(ctx context.Context, username string) ([]model.Contest, error)
}

type contestRepository struct {
	db *gorm.DB
}

func NewContestRepository() ContestRepository {
	return &contestRepository{
		db: config.DB,
	}
}

func (r *contestRepository) GetAll(ctx context.Context) ([]model.Contest, error) {
	var contests []model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Find(&contests).Error

	return contests, err
}

func (r *contestRepository) Create(ctx context.Context, contest *model.Contest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(contest).Error; err != nil {
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

		return tx.Create(&squares).Error
	})
}

func (r *contestRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		First(&contest, "id = ?", id).Error

	return &contest, err
}

func (r *contestRepository) UpdateLabels(ctx context.Context, contestID uuid.UUID, xLabels, yLabels []int8, user string) (*model.Contest, error) {
	var updatedContest *model.Contest
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var contest model.Contest
		if err := tx.Where("id = ?", contestID).First(&contest).Error; err != nil {
			return err
		}

		xLabelsJSON, _ := json.Marshal(xLabels)
		yLabelsJSON, _ := json.Marshal(yLabels)

		contest.XLabels = xLabelsJSON
		contest.YLabels = yLabelsJSON

		if user != "" {
			contest.UpdatedBy = user
		}

		if err := tx.Save(&contest).Error; err != nil {
			return err
		}

		updatedContest = &contest
		return nil
	})

	return updatedContest, err
}

func (r *contestRepository) UpdateSquare(ctx context.Context, squareID uuid.UUID, value, user string) (*model.Square, error) {
	var updatedSquare *model.Square
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

		updatedSquare = &square
		return nil
	})

	return updatedSquare, err
}

func (r *contestRepository) GetAllByUser(ctx context.Context, username string) ([]model.Contest, error) {
	var contests []model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Where("created_by = ?", username).
		Find(&contests).Error

	return contests, err
}
