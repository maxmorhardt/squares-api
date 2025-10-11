package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type EventService struct{}

func NewEventService() *EventService {
	return &EventService{}
}

func (s *EventService) PublishCellUpdate(ctx context.Context, gridID uuid.UUID, cellID uuid.UUID, value string, updatedBy string) error {
	updateMessage := model.NewCellUpdateMessage(gridID, cellID, value, updatedBy)

	channel := fmt.Sprintf("%s:%s", model.GridChannelPrefix, gridID)
	jsonData, err := json.Marshal(updateMessage)
	if err != nil {
		return err
	}

	return config.RedisClient.Publish(ctx, channel, jsonData).Err()
}