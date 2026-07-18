package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// offseason shows week 1 while an in-season list stays capped to ~2 weeks of fixtures
const upcomingWindow = 14 * 24 * time.Hour

type GameRepository interface {
	Upsert(ctx context.Context, game *model.Game) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Game, error)
	GetUpcoming(ctx context.Context) ([]model.Game, error)

	UpsertScore(ctx context.Context, score *model.GameScore) (created bool, err error)

	HasLiveGame(ctx context.Context) (bool, error)
	NextKickoff(ctx context.Context) (time.Time, error)
}

type gameRepository struct {
	db *gorm.DB
}

func NewGameRepository(db *gorm.DB) GameRepository {
	return &gameRepository{db: db}
}

// liveColumns are the mutable fields refreshed from ESPN on every upsert
var liveColumns = []string{
	"home_team", "away_team", "home_abbr", "away_abbr", "game_time",
	"week", "season", "season_type", "status", "period", "home_score", "away_score",
}

func (r *gameRepository) Upsert(ctx context.Context, game *model.Game) error {
	existing := &model.Game{}
	err := r.db.WithContext(ctx).Where("espn_id = ?", game.ESPNID).First(existing).Error
	if err == nil {
		// preserve identity, refresh the mutable fields in place
		game.ID = existing.ID
		game.CreatedAt = existing.CreatedAt
		return r.db.WithContext(ctx).Model(existing).Select(liveColumns).Updates(game).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return r.db.WithContext(ctx).Create(game).Error
}

func (r *gameRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Game, error) {
	var game model.Game
	err := r.db.WithContext(ctx).
		Preload("Scores", func(db *gorm.DB) *gorm.DB { return db.Order("quarter ASC") }).
		First(&game, "id = ?", id).Error
	return &game, err
}

func (r *gameRepository) GetUpcoming(ctx context.Context) ([]model.Game, error) {
	now := time.Now()

	// only games that haven't kicked off yet can be linked to a new contest
	var nextKickoff []time.Time
	if err := r.db.WithContext(ctx).Model(&model.Game{}).
		Where("status = ? AND game_time > ?", model.GameStatusScheduled, now).
		Order("game_time ASC").Limit(1).
		Pluck("game_time", &nextKickoff).Error; err != nil {
		return nil, err
	}
	if len(nextKickoff) == 0 {
		return []model.Game{}, nil
	}

	var games []model.Game
	err := r.db.WithContext(ctx).
		Where("status = ? AND game_time > ? AND game_time <= ?",
			model.GameStatusScheduled, now, nextKickoff[0].Add(upcomingWindow)).
		Order("game_time ASC").
		Find(&games).Error
	return games, err
}

func (r *gameRepository) HasLiveGame(ctx context.Context) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Game{}).
		Where("status = ?", model.GameStatusInProgress).
		Count(&count).Error
	return count > 0, err
}

func (r *gameRepository) NextKickoff(ctx context.Context) (time.Time, error) {
	var kickoff []time.Time
	err := r.db.WithContext(ctx).Model(&model.Game{}).
		Where("status = ? AND game_time > ?", model.GameStatusScheduled, time.Now()).
		Order("game_time ASC").Limit(1).
		Pluck("game_time", &kickoff).Error
	if err != nil || len(kickoff) == 0 {
		return time.Time{}, err
	}
	return kickoff[0], nil
}

func (r *gameRepository) UpsertScore(ctx context.Context, score *model.GameScore) (bool, error) {
	res := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "game_id"}, {Name: "quarter"}},
			DoNothing: true,
		}).
		Create(score)
	if res.Error != nil {
		return false, res.Error
	}

	return res.RowsAffected > 0, nil
}
