package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type NatsService interface {
	PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error
	PublishContestUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error
	PublishQuarterResult(ctx context.Context, contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error
	PublishContestDeleted(ctx context.Context, contestID uuid.UUID, updatedBy string) error
}

type natsService struct{}

func NewNatsService() NatsService {
	return &natsService{}
}

func (s *natsService) PublishSquareUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error {
	updateMessage := model.NewSquareUpdateMessage(contestID, updatedBy, &model.SquareWSUpdate{
		SquareID: squareID,
		Value:    value,
	})

	return s.publishToContestSubject(ctx, contestID, updateMessage)
}

func (s *natsService) PublishContestUpdate(ctx context.Context, contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error {
	updateMessage := model.NewContestUpdateMessage(contestID, updatedBy, contestUpdate)
	return s.publishToContestSubject(ctx, contestID, updateMessage)
}

func (s *natsService) PublishQuarterResult(ctx context.Context, contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error {
	updateMessage := model.NewQuarterResultUpdateMessage(contestID, updatedBy, quarterResult)
	return s.publishToContestSubject(ctx, contestID, updateMessage)
}

func (s *natsService) PublishContestDeleted(ctx context.Context, contestID uuid.UUID, updatedBy string) error {
	updateMessage := model.NewContestDeletedMessage(contestID, updatedBy)
	return s.publishToContestSubject(ctx, contestID, updateMessage)
}

func (s *natsService) publishToContestSubject(ctx context.Context, contestID uuid.UUID, message any) error {
	subject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return config.NATS().Publish(subject, jsonData)
}
