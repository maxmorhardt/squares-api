package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type RedisService interface	{
	PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error
	PublishLabelsUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, xLabels, yLabels []int8) error
}

type redisService struct{}

func NewRedisService() RedisService {
	return &redisService{}
}

func (s *redisService) PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error {
	updateMessage := model.NewSquareUpdateMessage(contestID, updatedBy, &model.SquareWSUpdate{
		SquareID: squareID,
		Value:    value,
	})

	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *redisService) PublishLabelsUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, xLabels, yLabels []int8) error {
	updateMessage := model.NewContestUpdateMessage(contestID, updatedBy, &model.ContestWSUpdate{
		XLabels: xLabels,
		YLabels: yLabels,
	})
	
	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *redisService) publishToContestChannel(ctx context.Context, contestID uuid.UUID, message any) error {
	channel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return config.RedisClient.Publish(ctx, channel, jsonData).Err()
}