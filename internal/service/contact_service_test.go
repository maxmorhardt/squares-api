package service_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewContactService(t *testing.T) {
	require.NotNil(t, service.NewContactService(nil, nil))
}

func TestContactService_SubmitContact_TurnstileValidationFails(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	svc := service.NewContactService(mocks.NewContactRepository(t), contactCfg())

	err := svc.SubmitContact(ctx, contactReq(), "127.0.0.1")

	assert.ErrorIs(t, err, errs.ErrInvalidTurnstile)
}

func contactCfg() *model.AppConfig {
	cfg := &model.AppConfig{}
	cfg.Turnstile.SecretKey = "test-secret"
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = 1
	cfg.SMTP.User = "sender@example.com"
	cfg.SMTP.Password = "pass"
	cfg.SMTP.SupportEmail = "support@example.com"
	return cfg
}

func contactReq() *model.ContactRequest {
	return &model.ContactRequest{
		Name:           "Alice",
		Email:          "alice@example.com",
		Subject:        "Hello",
		Message:        "Test message",
		TurnstileToken: "token",
	}
}

func TestContactService_SubmitContact_TurnstileAPIReturnsFail(t *testing.T) {
	cfg := turnstileServer(t, http.StatusOK, `{"success":false,"error-codes":["invalid-input-response"]}`)

	svc := service.NewContactService(mocks.NewContactRepository(t), cfg)

	err := svc.SubmitContact(context.Background(), contactReq(), "127.0.0.1")

	assert.ErrorIs(t, err, errs.ErrInvalidTurnstile)
}

func turnstileServer(t *testing.T, statusCode int, body string) *model.AppConfig {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(ts.Close)

	cfg := &model.AppConfig{}
	cfg.Turnstile.SecretKey = "test-secret"
	cfg.Turnstile.BaseURL = ts.URL
	cfg.SMTP.Host = "127.0.0.1"
	cfg.SMTP.Port = 1
	cfg.SMTP.User = "sender@example.com"
	cfg.SMTP.Password = "pass"
	cfg.SMTP.SupportEmail = "support@example.com"
	return cfg
}

func TestContactService_SubmitContact_TurnstileAPIStatusError(t *testing.T) {
	cfg := turnstileServer(t, http.StatusInternalServerError, `{}`)

	svc := service.NewContactService(mocks.NewContactRepository(t), cfg)

	err := svc.SubmitContact(context.Background(), contactReq(), "127.0.0.1")

	assert.ErrorIs(t, err, errs.ErrInvalidTurnstile)
}

func TestContactService_SubmitContact_RepoCreateFails(t *testing.T) {
	cfg := turnstileServer(t, http.StatusOK, `{"success":true}`)

	repo := mocks.NewContactRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db error"))

	svc := service.NewContactService(repo, cfg)

	err := svc.SubmitContact(context.Background(), contactReq(), "127.0.0.1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestContactService_SubmitContact_Success(t *testing.T) {
	cfg := turnstileServer(t, http.StatusOK, `{"success":true}`)

	repo := mocks.NewContactRepository(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	svc := service.NewContactService(repo, cfg)

	err := svc.SubmitContact(context.Background(), contactReq(), "127.0.0.1")

	assert.NoError(t, err)
}
