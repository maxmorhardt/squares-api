package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository interface {
	GetOrCreate(ctx context.Context, email, defaultDisplayName, defaultInitials string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	UpdateProfile(ctx context.Context, email, initials string) (*model.User, []model.Square, error)
	GetStats(ctx context.Context, email string) (*model.UserStatsResponse, error)
	GetActiveContests(ctx context.Context, email string) ([]model.UserActiveContest, error)
	ScrubUserData(ctx context.Context, email string) error
	IsTokenRevoked(ctx context.Context, email string, issuedAtUnix int64) (bool, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) GetOrCreate(ctx context.Context, email, defaultDisplayName, defaultInitials string) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("email = ?", email).First(user).Error
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// member since reflects the user's first activity, not their first profile visit
	var firstActivity sql.NullTime
	if err := r.db.WithContext(ctx).Raw(
		`SELECT MIN(t) FROM (
			SELECT MIN(created_at) AS t FROM contests WHERE owner = ?
			UNION ALL
			SELECT MIN(joined_at) FROM contest_participants WHERE user_id = ?
		) activity`, email, email).Scan(&firstActivity).Error; err != nil {
		return nil, err
	}

	newUser := &model.User{Email: email, DisplayName: defaultDisplayName, DefaultInitials: defaultInitials}
	if firstActivity.Valid {
		newUser.CreatedAt = firstActivity.Time
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "email"}}, DoNothing: true}).
		Create(newUser).Error; err != nil {
		return nil, err
	}

	user = &model.User{}
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) IsTokenRevoked(ctx context.Context, email string, issuedAtUnix int64) (bool, error) {
	// a tombstone revokes every token issued at or before the deletion instant
	var revoked bool
	if err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM deleted_accounts WHERE email = ? AND deleted_at >= to_timestamp(?))`,
		email, issuedAtUnix).Scan(&revoked).Error; err != nil {
		return false, err
	}

	return revoked, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user := &model.User{}
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) UpdateProfile(ctx context.Context, email, initials string) (*model.User, []model.Square, error) {
	user := &model.User{}
	var squares []model.Square

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).
			Where("email = ?", email).
			Update("default_initials", initials).Error; err != nil {
			return err
		}

		if err := tx.Where("email = ?", email).First(user).Error; err != nil {
			return err
		}

		// cascade the new initials to the user's squares in contests still in play
		liveContests := tx.Model(&model.Contest{}).Select("id").
			Where("status NOT IN ?", []model.ContestStatus{model.ContestStatusFinished, model.ContestStatusDeleted})

		if err := tx.Model(&model.Square{}).
			Where("owner = ? AND contest_id IN (?)", email, liveContests).
			Update("value", initials).Error; err != nil {
			return err
		}

		// re-select the affected squares so the caller can broadcast the change
		if err := tx.Where("owner = ? AND contest_id IN (?)", email, liveContests).
			Find(&squares).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return user, squares, nil
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
		Joins("JOIN contests c ON c.id = squares.contest_id AND c.status <> ?", model.ContestStatusDeleted).
		Where("squares.owner = ?", email).
		Count(&stats.SquaresClaimed).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).
		Model(&model.QuarterResult{}).
		Joins("JOIN contests c ON c.id = quarter_results.contest_id AND c.status <> ?", model.ContestStatusDeleted).
		Where("quarter_results.winner = ?", email).
		Count(&stats.QuarterWins).Error; err != nil {
		return nil, err
	}

	// every quarter the user had a stake in, so the win rate is wins per opportunity
	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*)
		FROM quarter_results q
		JOIN contests c ON c.id = q.contest_id AND c.status <> ?
		WHERE EXISTS (
			SELECT 1 FROM squares s WHERE s.contest_id = q.contest_id AND s.owner = ?
		)`, model.ContestStatusDeleted, email).
		Scan(&stats.QuartersPlayed).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

func (r *userRepository) GetActiveContests(ctx context.Context, email string) ([]model.UserActiveContest, error) {
	var contests []model.UserActiveContest

	// contests that still receive live updates, where the user is the owner or a participant
	if err := r.db.WithContext(ctx).Raw(
		`SELECT c.id, c.name, c.owner, CASE WHEN c.owner = ? THEN 'owner' ELSE 'participant' END AS role
		FROM contests c
		WHERE c.status NOT IN ?
		AND (c.owner = ? OR c.id IN (SELECT contest_id FROM contest_participants WHERE user_id = ?))
		ORDER BY c.created_at`,
		email, []model.ContestStatus{model.ContestStatusFinished, model.ContestStatusDeleted}, email, email).
		Scan(&contests).Error; err != nil {
		return nil, err
	}

	return contests, nil
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

		// finished/deleted contests keep their history under the ghost identity
		anonymize := []struct {
			tableModel any
			column     string
		}{
			{&model.Square{}, "owner"},
			{&model.Square{}, "created_by"},
			{&model.Square{}, "updated_by"},
			{&model.QuarterResult{}, "winner"},
			{&model.QuarterResult{}, "created_by"},
			{&model.QuarterResult{}, "updated_by"},
			{&model.Contest{}, "owner"},
			{&model.Contest{}, "created_by"},
			{&model.Contest{}, "updated_by"},
			{&model.ContestInvite{}, "created_by"},
		}
		for _, a := range anonymize {
			if err := tx.Model(a.tableModel).
				Where(a.column+" = ?", email).
				Update(a.column, model.GhostUser).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", email).Delete(&model.ContestParticipant{}).Error; err != nil {
			return err
		}

		if err := tx.Where("email = ?", email).Delete(&model.User{}).Error; err != nil {
			return err
		}

		// tombstone the account so pre-deletion tokens are rejected everywhere
		if err := tx.Exec(
			`INSERT INTO deleted_accounts (email, deleted_at) VALUES (?, ?)
			ON CONFLICT (email) DO UPDATE SET deleted_at = EXCLUDED.deleted_at`,
			email, time.Now()).Error; err != nil {
			return err
		}

		return nil
	})
}
