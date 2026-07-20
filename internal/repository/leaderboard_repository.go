package repository

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/model"
	"gorm.io/gorm"
)

// wins per user, excluding deleted contests and the ghost identity left behind by account deletion.
// joining users here keeps the ranked population identical to the one the board can display, so a
// winner without a profile can never inflate totalRanked or push a real player down a place
const winsCTE = `WITH wins AS (
	SELECT q.winner AS email, COUNT(*) AS quarter_wins
	FROM quarter_results q
	JOIN contests c ON c.id = q.contest_id AND c.status <> ?
	JOIN users u ON u.email = q.winner
	WHERE q.winner <> '' AND q.winner <> ?
	GROUP BY q.winner
)`

type LeaderboardRepository interface {
	GetTopWinners(ctx context.Context, limit int) ([]model.LeaderboardEntry, error)
	GetUserRank(ctx context.Context, email string) (*model.LeaderboardRankResponse, error)
}

type leaderboardRepository struct {
	db *gorm.DB
}

func NewLeaderboardRepository(db *gorm.DB) LeaderboardRepository {
	return &leaderboardRepository{
		db: db,
	}
}

func (r *leaderboardRepository) GetTopWinners(ctx context.Context, limit int) ([]model.LeaderboardEntry, error) {
	entries := make([]model.LeaderboardEntry, 0, limit)

	// joining users drops scrubbed accounts, so only live profiles are ever exposed
	if err := r.db.WithContext(ctx).Raw(winsCTE+`
		SELECT u.display_name AS display_name,
			w.quarter_wins AS quarter_wins,
			COALESCE(sq.squares_claimed, 0) AS squares_claimed
		FROM wins w
		JOIN users u ON u.email = w.email
		LEFT JOIN (
			SELECT owner, COUNT(*) AS squares_claimed
			FROM squares
			WHERE owner <> ''
			GROUP BY owner
		) sq ON sq.owner = w.email
		ORDER BY w.quarter_wins DESC, squares_claimed ASC, u.display_name ASC
		LIMIT ?`,
		model.ContestStatusDeleted, model.GhostUser, limit).
		Scan(&entries).Error; err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *leaderboardRepository) GetUserRank(ctx context.Context, email string) (*model.LeaderboardRankResponse, error) {
	var rank model.LeaderboardRankResponse

	// a user with no wins has no row in wins, so the EXISTS guard keeps them at rank 0
	if err := r.db.WithContext(ctx).Raw(winsCTE+`, me AS (
			SELECT quarter_wins FROM wins WHERE email = ?
		)
		SELECT
			(SELECT COUNT(*) FROM wins) AS total_ranked,
			COALESCE((SELECT quarter_wins FROM me), 0) AS quarter_wins,
			CASE WHEN EXISTS (SELECT 1 FROM me)
				THEN (SELECT COUNT(*) + 1 FROM wins w WHERE w.quarter_wins > (SELECT quarter_wins FROM me))
				ELSE 0
			END AS rank`,
		model.ContestStatusDeleted, model.GhostUser, email).
		Scan(&rank).Error; err != nil {
		return nil, err
	}

	rank.Ranked = rank.Rank > 0

	return &rank, nil
}
