package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type RedisService struct{}

func NewRedisService() *RedisService {
	return &RedisService{}
}

func (s *RedisService) PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, squareID uuid.UUID, value string, updatedBy string) error {
	updateMessage := model.NewSquareUpdateMessage(contestID, squareID, value, updatedBy)
	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *RedisService) PublishContestUpdate(ctx context.Context, contestID uuid.UUID, xLabels, yLabels []int8, updatedBy string) error {
	updateMessage := model.NewContestUpdateMessage(contestID, xLabels, yLabels, updatedBy)
	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *RedisService) publishToContestChannel(ctx context.Context, contestID uuid.UUID, message interface{}) error {
	channel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return config.RedisClient.Publish(ctx, channel, jsonData).Err()
}