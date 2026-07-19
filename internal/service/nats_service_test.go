package service_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNatsService_PublishWithoutConnection(t *testing.T) {
	svc := service.NewNatsService(nil)
	contestID := uuid.New()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"square update", func() error { return svc.PublishSquareUpdate(contestID, "user", &model.Square{}) }},
		{"contest update", func() error { return svc.PublishContestUpdate(contestID, "user", &model.Contest{}) }},
		{"quarter result", func() error { return svc.PublishQuarterResult(contestID, "user", &model.QuarterResult{}) }},
		{"contest deleted", func() error { return svc.PublishContestDeleted(contestID, "user") }},
		{"participant removed", func() error {
			return svc.PublishParticipantRemoved(contestID, "user", &model.ContestParticipant{})
		}},
		{"participant added", func() error { return svc.PublishParticipantAdded(contestID, &model.ContestParticipant{}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "NATS connection is not available")
		})
	}
}

func anyNats() *mocks.NatsService {
	m := &mocks.NatsService{}
	m.On("PublishSquareUpdate", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishContestUpdate", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishQuarterResult", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishQuarterResultRollback", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishContestDeleted", mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishParticipantRemoved", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	m.On("PublishParticipantAdded", mock.Anything, mock.Anything).Return(nil).Maybe()
	return m
}
