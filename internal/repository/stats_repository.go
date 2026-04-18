package repository

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

type StatsRepository interface {
	GetStats(ctx context.Context) (*model.StatsResponse, error)
}

type statsRepository struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) StatsRepository {
	return &statsRepository{
		db: db,
	}
}

func (r *statsRepository) GetStats(ctx context.Context) (*model.StatsResponse, error) {
	var stats model.StatsResponse

	today := gorm.Expr("CURRENT_DATE")

	if err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("created_at >= ? AND status != ?", today, model.ContestStatusDeleted).
		Count(&stats.ContestsCreatedToday).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Square{}).
		Where("owner != '' AND updated_at >= ?", today).
		Count(&stats.SquaresClaimedToday).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.Contest{}).
		Where("status NOT IN ?", []model.ContestStatus{model.ContestStatusDeleted, model.ContestStatusFinished}).
		Count(&stats.TotalActiveContests).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}
