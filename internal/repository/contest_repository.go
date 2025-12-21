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
	ExistsByID(ctx context.Context, id uuid.UUID) (bool, error)
	GetAllByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error)
	GetAllParticipatingContestsPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error)
	ExistsByUserAndName(ctx context.Context, username, name string) (bool, error)

	Create(ctx context.Context, contest *model.Contest) error
	Update(ctx context.Context, contest *model.Contest) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateQuarterResult(ctx context.Context, result *model.QuarterResult) error

	AddParticipant(ctx context.Context, contestID uuid.UUID, username string, squareLimit int) error
	GetParticipant(ctx context.Context, contestID uuid.UUID, username string) (*model.ContestParticipant, error)
	GetUserSquareCount(ctx context.Context, contestID uuid.UUID, username string) (int, error)

	GetSquareByID(ctx context.Context, squareID uuid.UUID) (*model.Square, error)
	UpdateSquare(ctx context.Context, square *model.Square, value, owner, firstName, lastName string) (*model.Square, error)
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

func (r *contestRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("id = ? AND status != ?", id, model.ContestStatusDeleted).
		Count(&count).Error

	return count > 0, err
}

func (r *contestRepository) GetAllByUserPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	// get total count of contests for user
	if err := r.db.WithContext(ctx).Model(&model.Contest{}).Where("created_by = ? AND status != ?", username, model.ContestStatusDeleted).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// calculate offset and fetch paginated contests
	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Preload("QuarterResults", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		Where("created_by = ? AND status != ?", username, model.ContestStatusDeleted).
		Order("updated_at DESC").
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

func (r *contestRepository) UpdateSquare(ctx context.Context, square *model.Square, value, owner, firstName, lastName string) (*model.Square, error) {
	var updatedSquare *model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// update square value and owner information
		square.Value = value
		square.Owner = owner
		square.OwnerFirstName = firstName
		square.OwnerLastName = lastName

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
		square.OwnerFirstName = ""
		square.OwnerLastName = ""

		// save cleared square
		if err := tx.Save(square).Error; err != nil {
			return err
		}

		clearedSquare = square
		return nil
	})

	return clearedSquare, err
}

// ====================
// Participant Management
// ====================

func (r *contestRepository) AddParticipant(ctx context.Context, contestID uuid.UUID, username string, squareLimit int) error {
	// check if already a participant
	participant, err := r.GetParticipant(ctx, contestID, username)
	if err == nil && participant != nil {
		// already a participant, update square limit if it's higher
		if squareLimit > participant.SquareLimit {
			participant.SquareLimit = squareLimit
			return r.db.WithContext(ctx).Save(participant).Error
		}
		return nil // already a participant with same or higher limit
	}

	// create new participant
	newParticipant := &model.ContestParticipant{
		ContestID:   contestID,
		Username:    username,
		SquareLimit: squareLimit,
	}

	return r.db.WithContext(ctx).Create(newParticipant).Error
}

func (r *contestRepository) GetParticipant(ctx context.Context, contestID uuid.UUID, username string) (*model.ContestParticipant, error) {
	var participant model.ContestParticipant
	err := r.db.WithContext(ctx).
		Where("contest_id = ? AND username = ?", contestID, username).
		First(&participant).Error

	if err != nil {
		return nil, err
	}
	return &participant, nil
}

func (r *contestRepository) GetUserSquareCount(ctx context.Context, contestID uuid.UUID, username string) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Square{}).
		Where("contest_id = ? AND owner = ?", contestID, username).
		Count(&count).Error

	return int(count), err
}

func (r *contestRepository) GetAllParticipatingContestsPaginated(ctx context.Context, username string, page, limit int) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	// get total count of contests user is participating in
	if err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Joins("JOIN contest_participants ON contest_participants.contest_id = contests.id").
		Where("contest_participants.username = ? AND contests.status != ?", username, model.ContestStatusDeleted).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// calculate offset and fetch paginated contests
	offset := (page - 1) * limit
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Preload("QuarterResults", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		Joins("JOIN contest_participants ON contest_participants.contest_id = contests.id").
		Where("contest_participants.username = ? AND contests.status != ?", username, model.ContestStatusDeleted).
		Order("contests.updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&contests).Error

	return contests, total, err
}
