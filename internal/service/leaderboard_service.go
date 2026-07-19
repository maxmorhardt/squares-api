package service

import (
	"context"
	"strings"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const (
	leaderboardCacheTTL = 60 * time.Second
	// distinct limits worth caching at once
	leaderboardCacheSize = 8

	DefaultLeaderboardLimit = 25
	MaxLeaderboardLimit     = 100
)

type LeaderboardService interface {
	GetLeaderboard(ctx context.Context, limit int) (*model.LeaderboardResponse, error)
	GetUserRank(ctx context.Context, email string) (*model.LeaderboardRankResponse, error)
}

type leaderboardService struct {
	repo  repository.LeaderboardRepository
	cache *util.TTLCache[int, *model.LeaderboardResponse]
}

func NewLeaderboardService(repo repository.LeaderboardRepository) LeaderboardService {
	return &leaderboardService{
		repo:  repo,
		cache: util.NewTTLCache[int, *model.LeaderboardResponse](leaderboardCacheSize, leaderboardCacheTTL),
	}
}

func (s *leaderboardService) GetLeaderboard(ctx context.Context, limit int) (*model.LeaderboardResponse, error) {
	log := util.LoggerFromContext(ctx)

	if limit <= 0 {
		limit = DefaultLeaderboardLimit
	}
	if limit > MaxLeaderboardLimit {
		limit = MaxLeaderboardLimit
	}

	leaderboard, err := s.cache.GetOrLoad(ctx, limit, func(ctx context.Context) (*model.LeaderboardResponse, error) {
		entries, err := s.repo.GetTopWinners(ctx, limit)
		if err != nil {
			return nil, err
		}

		return &model.LeaderboardResponse{Entries: assignRanks(entries)}, nil
	})
	if err != nil {
		log.Error("failed to get leaderboard", "error", err)
		return nil, err
	}

	return leaderboard, nil
}

func (s *leaderboardService) GetUserRank(ctx context.Context, email string) (*model.LeaderboardRankResponse, error) {
	log := util.LoggerFromContext(ctx)

	rank, err := s.repo.GetUserRank(ctx, email)
	if err != nil {
		log.Error("failed to get user rank", "error", err)
		return nil, err
	}

	return rank, nil
}

// entries arrive ordered by wins desc, so tied players share the rank of the first of their group
func assignRanks(entries []model.LeaderboardEntry) []model.LeaderboardEntry {
	for i := range entries {
		entries[i].DisplayName = maskEmail(entries[i].DisplayName)

		if i > 0 && entries[i].QuarterWins == entries[i-1].QuarterWins {
			entries[i].Rank = entries[i-1].Rank
			continue
		}

		entries[i].Rank = i + 1
	}

	return entries
}

// a display name defaults to the email when the provider sends no name claim, and this
// board is public, so never publish anything past the local part
func maskEmail(displayName string) string {
	at := strings.Index(displayName, "@")
	if at < 0 {
		return displayName
	}
	if at == 0 {
		return "player"
	}

	return displayName[:at]
}
