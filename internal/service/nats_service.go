package service

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
)

type NatsService interface {
	PublishSquareUpdate(contestID uuid.UUID, updatedBy string, square *model.Square) error
	PublishContestUpdate(contestID uuid.UUID, updatedBy string, contest *model.Contest) error
	PublishQuarterResult(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResult) error
	PublishContestDeleted(contestID uuid.UUID, updatedBy string) error
}

type natsService struct{}

func NewNatsService() NatsService {
	return &natsService{}
}

func (s *natsService) PublishSquareUpdate(contestID uuid.UUID, updatedBy string, square *model.Square) error {
	updateMessage := model.NewSquareUpdateMessage(contestID, updatedBy, square)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishContestUpdate(contestID uuid.UUID, updatedBy string, contest *model.Contest) error {
	updateMessage := model.NewContestUpdateMessage(contestID, updatedBy, contest)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishQuarterResult(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResult) error {
	updateMessage := model.NewQuarterResultUpdateMessage(contestID, updatedBy, quarterResult)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishContestDeleted(contestID uuid.UUID, updatedBy string) error {
	updateMessage := model.NewContestDeletedMessage(contestID, updatedBy)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) publishToContestSubject(contestID uuid.UUID, message any) error {
	subject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID.String())
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	natsConn := config.NATS()
	if natsConn == nil || !natsConn.IsConnected() {
		return fmt.Errorf("NATS connection is not available")
	}

	if err := natsConn.Publish(subject, jsonData); err != nil {
		return fmt.Errorf("failed to publish to NATS subject %s: %w", subject, err)
	}

	return nil
}
