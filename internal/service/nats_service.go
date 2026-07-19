package service

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/nats-io/nats.go"
)

type NatsService interface {
	PublishSquareUpdate(contestID uuid.UUID, updatedBy string, square *model.Square) error
	PublishContestUpdate(contestID uuid.UUID, updatedBy string, contest *model.Contest) error
	PublishQuarterResult(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResult) error
	PublishQuarterResultRollback(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResult, contest *model.Contest) error
	PublishContestDeleted(contestID uuid.UUID, updatedBy string) error
	PublishParticipantRemoved(contestID uuid.UUID, updatedBy string, participant *model.ContestParticipant) error
	PublishParticipantAdded(contestID uuid.UUID, participant *model.ContestParticipant) error
}

type natsService struct {
	nats *nats.Conn
}

func NewNatsService(nc *nats.Conn) NatsService {
	return &natsService{nats: nc}
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

func (s *natsService) PublishQuarterResultRollback(contestID uuid.UUID, updatedBy string, quarterResult *model.QuarterResult, contest *model.Contest) error {
	updateMessage := model.NewQuarterResultRollbackMessage(contestID, updatedBy, quarterResult, contest)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishContestDeleted(contestID uuid.UUID, updatedBy string) error {
	updateMessage := model.NewContestDeletedMessage(contestID, updatedBy)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishParticipantRemoved(contestID uuid.UUID, updatedBy string, participant *model.ContestParticipant) error {
	updateMessage := model.NewParticipantRemovedMessage(contestID, updatedBy, participant)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) PublishParticipantAdded(contestID uuid.UUID, participant *model.ContestParticipant) error {
	updateMessage := model.NewParticipantAddedMessage(contestID, participant)
	return s.publishToContestSubject(contestID, updateMessage)
}

func (s *natsService) publishToContestSubject(contestID uuid.UUID, message any) error {
	subject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID.String())
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if s.nats == nil || !s.nats.IsConnected() {
		return fmt.Errorf("NATS connection is not available")
	}

	if err := s.nats.Publish(subject, jsonData); err != nil {
		return fmt.Errorf("failed to publish to NATS subject %s: %w", subject, err)
	}

	metrics.IncNATSMessagePublished(model.ContestChannelPrefix)
	return nil
}
