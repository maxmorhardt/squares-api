package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type RedisService interface {
	PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error
	PublishContestUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error
	PublishQuarterResult(ctx context.Context, contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error
	PublishContestDeleted(ctx context.Context, contestID uuid.UUID, updatedBy string) error
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

func (s *redisService) PublishContestUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error {
	updateMessage := model.NewContestUpdateMessage(contestID, updatedBy, contestUpdate)
	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *redisService) PublishQuarterResult(ctx context.Context, contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error {
	updateMessage := model.NewQuarterResultUpdateMessage(contestID, updatedBy, quarterResult)
	return s.publishToContestChannel(ctx, contestID, updateMessage)
}

func (s *redisService) PublishContestDeleted(ctx context.Context, contestID uuid.UUID, updatedBy string) error {
	updateMessage := model.NewContestDeletedMessage(contestID, updatedBy)
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
