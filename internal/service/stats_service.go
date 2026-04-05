package service

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type StatsService interface {
	GetStats(ctx context.Context) (*model.StatsResponse, error)
}

type statsService struct {
	repo repository.StatsRepository
}

func NewStatsService(repo repository.StatsRepository) StatsService {
	return &statsService{
		repo: repo,
	}
}

func (s *statsService) GetStats(ctx context.Context) (*model.StatsResponse, error) {
	log := util.LoggerFromContext(ctx)

	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		log.Error("failed to get stats", "error", err)
		return nil, err
	}

	return stats, nil
}
