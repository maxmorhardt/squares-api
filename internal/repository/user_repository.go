package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository interface {
	GetOrCreate(ctx context.Context, email, defaultDisplayName string) (*model.User, error)
	GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error)
	GetOwnedActiveContestIDs(ctx context.Context, email string) ([]uuid.UUID, error)
	ScrubUserData(ctx context.Context, email string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) GetOrCreate(ctx context.Context, email, defaultDisplayName string) (*model.User, error) {
	// first authenticated visit creates the row; later visits return the existing one
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "email"}}, DoNothing: true}).
		Create(&model.User{Email: email, DisplayName: defaultDisplayName}).Error; err != nil {
		return nil, err
	}

	// select into a fresh struct so the created struct's id is not added to the where clause
	user := &model.User{}
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error) {
	var stats model.UserStatsResponse

	if err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("owner = ? AND status != ?", email, model.ContestStatusDeleted).
		Count(&stats.ContestsCreated).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.ContestParticipant{}).
		Where("user_id = ?", email).
		Count(&stats.ContestsJoined).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Square{}).
		Where("owner = ?", email).
		Count(&stats.SquaresClaimed).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.QuarterResult{}).
		Where("winner = ?", email).
		Count(&stats.QuarterWins).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *userRepository) GetOwnedActiveContestIDs(ctx context.Context, email string) ([]uuid.UUID, error) {
	var ids []uuid.UUID

	if err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("owner = ? AND status NOT IN ?", email, []model.ContestStatus{model.ContestStatusFinished, model.ContestStatusDeleted}).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}

	return ids, nil
}

func (r *userRepository) ScrubUserData(ctx context.Context, email string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// free the user's squares in contests that are still being played
		if err := tx.Model(&model.Square{}).
			Where("owner = ? AND contest_id IN (?)", email,
				tx.Model(&model.Contest{}).Select("id").
					Where("status NOT IN ?", []model.ContestStatus{model.ContestStatusFinished, model.ContestStatusDeleted})).
			Updates(map[string]any{"value": "", "owner": "", "owner_name": ""}).Error; err != nil {
			return err
		}

		// finished/deleted contests keep their history but drop the email
		if err := tx.Model(&model.Square{}).
			Where("owner = ?", email).
			Update("owner", "").Error; err != nil {
			return err
		}

		if err := tx.Model(&model.QuarterResult{}).
			Where("winner = ?", email).
			Update("winner", "").Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ?", email).Delete(&model.ContestParticipant{}).Error; err != nil {
			return err
		}

		if err := tx.Where("email = ?", email).Delete(&model.User{}).Error; err != nil {
			return err
		}

		return nil
	})
}
