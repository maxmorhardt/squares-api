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
	GetAllPaginated(ctx context.Context, page, limit int) ([]model.Contest, int64, error)
	Create(ctx context.Context, contest *model.Contest) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error)
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	UpdateLabels(ctx context.Context, contestID uuid.UUID, xLabels, yLabels []int8, user string) (*model.Contest, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateSquare(ctx context.Context, squareID uuid.UUID, value, user string) (*model.Square, error)
	GetAllByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error)
	ExistsByUserAndName(ctx context.Context, username, name string) (bool, error)
}

type contestRepository struct {
	db *gorm.DB
}

func NewContestRepository() ContestRepository {
	return &contestRepository{
		db: config.DB,
	}
}

func (r *contestRepository) GetAllPaginated(ctx context.Context, page, limit int) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Contest{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Offset(offset).
		Limit(limit).
		Find(&contests).Error

	return contests, total, err
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

func (r *contestRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("id = ? AND status != ?", id, model.ContestStatusDeleted).
		Count(&count).Error

	return count > 0, err
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

func (r *contestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("id = ?", id).
		Update("status", model.ContestStatusDeleted).Error
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

func (r *contestRepository) GetAllByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Contest{}).Where("created_by = ? AND status != ?", username, model.ContestStatusDeleted).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Where("created_by = ? AND status != ?", username, model.ContestStatusDeleted).
		Offset(offset).
		Limit(limit).
		Find(&contests).Error

	return contests, total, err
}

func (r *contestRepository) ExistsByUserAndName(ctx context.Context, username, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("created_by = ? AND name = ? AND status != ?", username, name, model.ContestStatusDeleted).
		Count(&count).Error

	return count > 0, err
}
