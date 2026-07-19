package service

import (
	"context"
	"time"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

// stats are identical for every caller and only shift at date granularity, so a
// short in-memory window spares the DB the repeated count queries under load
const statsCacheTTL = 30 * time.Second

type StatsService interface {
	GetStats(ctx context.Context) (*model.StatsResponse, error)
}

type statsService struct {
	repo  repository.StatsRepository
	cache *util.TTLCache[struct{}, *model.StatsResponse]
}

func NewStatsService(repo repository.StatsRepository) StatsService {
	return &statsService{
		repo:  repo,
		cache: util.NewTTLCache[struct{}, *model.StatsResponse](1, statsCacheTTL),
	}
}

func (s *statsService) GetStats(ctx context.Context) (*model.StatsResponse, error) {
	log := util.LoggerFromContext(ctx)

	stats, err := s.cache.GetOrLoad(ctx, struct{}{}, s.repo.GetStats)
	if err != nil {
		log.Error("failed to get stats", "error", err)
		return nil, err
	}

	return stats, nil
}
