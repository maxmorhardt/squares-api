package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ContestRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Contest, error)
	GetVisibilityByID(ctx context.Context, id uuid.UUID) (model.ContestVisibility, error)
	ExistsByOwnerAndName(ctx context.Context, owner, name string) (bool, error)
	GetAllByOwnerPaginated(ctx context.Context, owner string, page, limit int, search string) ([]model.Contest, int64, error)
	GetAllByParticipantUserID(ctx context.Context, userID, search string) ([]model.Contest, error)
	GetByGameID(ctx context.Context, gameID uuid.UUID) ([]model.Contest, error)

	Create(ctx context.Context, contest *model.Contest, owner *model.ContestParticipant) error
	Update(ctx context.Context, contest *model.Contest) error
	Delete(ctx context.Context, id uuid.UUID) error
	ApplyQuarterResults(ctx context.Context, contest *model.Contest, results []model.QuarterResult) error
	RollbackQuarterResult(ctx context.Context, resultID uuid.UUID, contest *model.Contest) error

	ClaimSquare(ctx context.Context, square *model.Square, value, owner, ownerName string) (*model.Square, error)
	ClearSquare(ctx context.Context, square *model.Square) (*model.Square, error)
	GhostSquare(ctx context.Context, square *model.Square) (*model.Square, error)
	ClearSquaresByOwner(ctx context.Context, contestID uuid.UUID, owner string) ([]model.Square, error)
}

type contestRepository struct {
	db *gorm.DB
}

func NewContestRepository(db *gorm.DB) ContestRepository {
	return &contestRepository{
		db: db,
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
		Preload("Game.Scores", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		First(&contest, "id = ? AND status != ?", id, model.ContestStatusDeleted).Error

	return &contest, err
}

func (r *contestRepository) GetVisibilityByID(ctx context.Context, id uuid.UUID) (model.ContestVisibility, error) {
	var contest model.Contest
	err := r.db.WithContext(ctx).
		Select("visibility").
		First(&contest, "id = ? AND status != ?", id, model.ContestStatusDeleted).Error
	return contest.Visibility, err
}

func (r *contestRepository) ExistsByOwnerAndName(ctx context.Context, owner, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("owner = ? AND name = ? AND status != ?", owner, name, model.ContestStatusDeleted).
		Count(&count).Error

	return count > 0, err
}

func (r *contestRepository) GetAllByOwnerPaginated(ctx context.Context, owner string, page, limit int, search string) ([]model.Contest, int64, error) {
	var contests []model.Contest
	var total int64

	q := r.db.WithContext(ctx).Model(&model.Contest{}).Where("created_by = ? AND status != ?", owner, model.ContestStatusDeleted)
	if search != "" {
		q = q.Where("name ILIKE ?", "%"+search+"%")
	}

	// get total count of contests for user
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// calculate offset and fetch paginated contests
	offset := (page - 1) * limit
	err := q.
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&contests).Error

	return contests, total, err
}

func (r *contestRepository) GetAllByParticipantUserID(ctx context.Context, userID, search string) ([]model.Contest, error) {
	var contests []model.Contest

	q := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Select("contests.*").
		Preload("Squares").
		Preload("QuarterResults", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		Preload("Game.Scores", func(db *gorm.DB) *gorm.DB {
			return db.Order("quarter ASC")
		}).
		Joins("JOIN contest_participants cp ON cp.contest_id = contests.id").
		Where("cp.user_id = ? AND cp.role != ? AND contests.status != ?", userID, model.ParticipantRoleOwner, model.ContestStatusDeleted)

	if search != "" {
		q = q.Where("contests.name ILIKE ?", "%"+search+"%")
	}

	err := q.Order("cp.joined_at DESC").Find(&contests).Error
	return contests, err
}

func (r *contestRepository) GetByGameID(ctx context.Context, gameID uuid.UUID) ([]model.Contest, error) {
	var contests []model.Contest
	err := r.db.WithContext(ctx).
		Preload("Squares").
		Where("game_id = ? AND status != ?", gameID, model.ContestStatusDeleted).
		Find(&contests).Error
	return contests, err
}

// ====================
// Contest Lifecycle Actions
// ====================

func (r *contestRepository) Create(ctx context.Context, contest *model.Contest, owner *model.ContestParticipant) error {
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

		if err := tx.Create(&squares).Error; err != nil {
			return err
		}

		// create owner participant within the same transaction
		owner.ContestID = contest.ID
		return tx.Create(owner).Error
	})
}

func (r *contestRepository) Update(ctx context.Context, contest *model.Contest) error {
	// only persist the contest row itself; never write preloaded associations
	return r.db.WithContext(ctx).Omit(clause.Associations).Save(contest).Error
}

func (r *contestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("id = ?", id).
		Update("status", model.ContestStatusDeleted).Error
}

func (r *contestRepository) ApplyQuarterResults(ctx context.Context, contest *model.Contest, results []model.QuarterResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(contest).Error; err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		// a quarter already on record wins, so a retried tick advances status without duplicating rows
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "contest_id"}, {Name: "quarter"}},
			DoNothing: true,
		}).Create(results).Error
	})
}

func (r *contestRepository) RollbackQuarterResult(ctx context.Context, resultID uuid.UUID, contest *model.Contest) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&model.QuarterResult{}, "id = ?", resultID).Error; err != nil {
			return err
		}

		// only persist the contest row itself; don't write preloaded associations
		return tx.Omit(clause.Associations).Save(contest).Error
	})
}

// ====================
// Square Actions
// ====================

func (r *contestRepository) ClaimSquare(ctx context.Context, square *model.Square, value, owner, ownerName string) (*model.Square, error) {
	var claimedSquare *model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// lock the participant row so this user's concurrent claims serialize and the limit below holds
		var participant model.ContestParticipant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("contest_id = ? AND user_id = ?", square.ContestID, owner).
			First(&participant).Error; err != nil {
			return err
		}

		// only an unclaimed square or one the caller already owns can be written, decided by the database
		res := tx.Model(&model.Square{}).
			Where("id = ? AND (owner = ? OR owner = ?)", square.ID, "", owner).
			Updates(map[string]any{
				"value":      value,
				"owner":      owner,
				"owner_name": ownerName,
				"updated_by": owner,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errs.ErrSquareTaken
		}

		// count after the write so the new claim is included, then roll back if it broke the limit
		var claimed int64
		if err := tx.Model(&model.Square{}).
			Where("contest_id = ? AND owner = ? AND value != ?", square.ContestID, owner, "").
			Count(&claimed).Error; err != nil {
			return err
		}
		if claimed > int64(participant.MaxSquares) {
			return errs.ErrSquareLimitReached
		}

		square.Value = value
		square.Owner = owner
		square.OwnerName = ownerName
		square.UpdatedBy = owner
		claimedSquare = square
		return nil
	})

	if err != nil {
		return nil, err
	}

	return claimedSquare, nil
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

func (r *contestRepository) GhostSquare(ctx context.Context, square *model.Square) (*model.Square, error) {
	var ghostedSquare *model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// keep the value so the started grid stays filled and scoring is unaffected
		square.Owner = model.GhostUser
		square.OwnerName = ""

		if err := tx.Save(square).Error; err != nil {
			return err
		}

		ghostedSquare = square
		return nil
	})

	return ghostedSquare, err
}

func (r *contestRepository) ClearSquaresByOwner(ctx context.Context, contestID uuid.UUID, owner string) ([]model.Square, error) {
	var clearedSquares []model.Square
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// load the caller's squares so their cleared state can be broadcast
		if err := tx.Where("contest_id = ? AND owner = ?", contestID, owner).Find(&clearedSquares).Error; err != nil {
			return err
		}

		if len(clearedSquares) == 0 {
			return nil
		}

		// clear value and owner for every square the caller owns in one update
		if err := tx.Model(&model.Square{}).
			Where("contest_id = ? AND owner = ?", contestID, owner).
			Updates(map[string]any{"value": "", "owner": "", "owner_name": ""}).Error; err != nil {
			return err
		}

		// reflect the cleared state on the returned copies for broadcasting
		for i := range clearedSquares {
			clearedSquares[i].Value = ""
			clearedSquares[i].Owner = ""
			clearedSquares[i].OwnerName = ""
		}

		return nil
	})

	return clearedSquares, err
}
