package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type ContestRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error)
	GetByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error)
	ExistsByOwnerAndName(ctx context.Context, owner, name string) (bool, error)
	GetAllByOwnerPaginated(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error)

	Create(ctx context.Context, contest *model.Contest) error
	Update(ctx context.Context, contest *model.Contest) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateQuarterResult(ctx context.Context, result *model.QuarterResult) error

	GetSquareByID(ctx context.Context, squareID uuid.UUID) (*model.Square, error)
	UpdateSquare(ctx context.Context, square *model.Square, value, owner, ownerName string) (*model.Square, error)
	ClearSquare(ctx context.Context, square *model.Square) (*model.Square, error)
}

type contestRepository struct {
	db *gorm.DB
}

func NewContestRepository() ContestRepository {
	return &contestRepository{
		db: config.DB,
	}
}

// ====================
// Getters
// ====================

func (r *contestRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Preload("QuarterResults", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		First(&contest, "id = ? AND status != ?", id, model.ContestStatusDeleted).Error

	return &contest, err
}

func (r *contestRepository) GetByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error) {
	var contest model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Preload("QuarterResults", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		First(&contest, "owner = ? AND name = ? AND status != ?", owner, name, model.ContestStatusDeleted).Error

	return &contest, err
}

func (r *contestRepository) ExistsByOwnerAndName(ctx context.Context, owner, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("owner = ? AND name = ? AND status != ?", owner, name, model.ContestStatusDeleted).
		Count(&count).Error

	return count > 0, err
}

func (r *contestRepository) GetAllByOwnerPaginated(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	// get total count of contests for user
	if err := r.db.WithContext(ctx).Model(&model.Contest{}).Where("created_by = ? AND status != ?", owner, model.ContestStatusDeleted).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// calculate offset and fetch paginated contests
	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Where("created_by = ? AND status != ?", owner, model.ContestStatusDeleted).
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&contests).Error

	return contests, total, err
}

// ====================
// Contest Lifecycle Actions
// ====================

func (r *contestRepository) Create(ctx context.Context, contest *model.Contest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// create contest record
		if err := tx.Create(contest).Error; err != nil {
			return err
		}

		// initialize 10x10 grid of squares
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

func (r *contestRepository) Update(ctx context.Context, contest *model.Contest) error {
	return r.db.WithContext(ctx).Save(contest).Error
}

func (r *contestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("id = ?", id).
		Update("status", model.ContestStatusDeleted).Error
}

func (r *contestRepository) CreateQuarterResult(ctx context.Context, result *model.QuarterResult) error {
	return r.db.WithContext(ctx).Create(result).Error
}

// ====================
// Square Actions
// ====================

func (r *contestRepository) GetSquareByID(ctx context.Context, squareID uuid.UUID) (*model.Square, error) {
	var square model.Square
	err := r.db.WithContext(ctx).Where("id = ?", squareID).First(&square).Error
	return &square, err
}

func (r *contestRepository) UpdateSquare(ctx context.Context, square *model.Square, value, owner, ownerName string) (*model.Square, error) {
	var updatedSquare *model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// update square value and owner information
		square.Value = value
		square.Owner = owner
		square.OwnerName = ownerName

		// save updated square
		if err := tx.Save(square).Error; err != nil {
			return err
		}

		updatedSquare = square
		return nil
	})

	return updatedSquare, err
}

func (r *contestRepository) ClearSquare(ctx context.Context, square *model.Square) (*model.Square, error) {
	var clearedSquare *model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// clear all square data
		square.Value = ""
		square.Owner = ""
		square.OwnerName = ""

		// save cleared square
		if err := tx.Save(square).Error; err != nil {
			return err
		}

		clearedSquare = square
		return nil
	})

	return clearedSquare, err
}
