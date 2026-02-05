package service

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type NatsService interface {
	PublishSquareUpdate(contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error
	PublishContestUpdate(contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error
	PublishQuarterResult(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error
	PublishContestDeleted(contestID uuid.UUID, updatedBy string) error
}

type natsService struct{}

func NewNatsService() NatsService {
	return &natsService{}
}

func (s *natsService) PublishSquareUpdate(contestID uuid.UUID, updatedBy string, squareID uuid.UUID, value string) error {
	updateMessage := model.NewSquareUpdateMessage(contestID, updatedBy, &model.SquareWSUpdate{
		SquareID: squareID,
		Value:    value,
	})

	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishContestUpdate(contestID uuid.UUID, updatedBy string, contestUpdate *model.ContestWSUpdate) error {
	updateMessage := model.NewContestUpdateMessage(contestID, updatedBy, contestUpdate)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishQuarterResult(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResultWSUpdate) error {
	updateMessage := model.NewQuarterResultUpdateMessage(contestID, updatedBy, quarterResult)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishContestDeleted(contestID uuid.UUID, updatedBy string) error {
	updateMessage := model.NewContestDeletedMessage(contestID, updatedBy)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) publishToContestSubject(contestID uuid.UUID, message any) error {
	subject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return config.NATS().Publish(subject, jsonData)
}
