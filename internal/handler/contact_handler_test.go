package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newContactHandler(fn func(ctx context.Context, req *model.ContactRequest, ipAddress string) error) ContactHandler {
	return NewContactHandler(&mockContactService{submitContactFn: fn})
}

func TestSubmitContact_Success(t *testing.T) {
	h := newContactHandler(func(_ context.Context, _ *model.ContactRequest, _ string) error {
		return nil
	})

	r := newTestRouter()
	r.POST("/contact", h.SubmitContact)

	body, _ := json.Marshal(model.ContactRequest{
		Name:           "John",
		Email:          "john@example.com",
		Subject:        "Hello",
		Message:        "Test message",
		TurnstileToken: "valid-token",
	})
	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.ContactResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Message, "submitted successfully")
}

func TestSubmitContact_InvalidBody(t *testing.T) {
	h := newContactHandler(nil) // should not be called

	r := newTestRouter()
	r.POST("/contact", h.SubmitContact)

	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitContact_InvalidTurnstile(t *testing.T) {
	h := newContactHandler(func(_ context.Context, _ *model.ContactRequest, _ string) error {
		return errs.ErrInvalidTurnstile
	})

	r := newTestRouter()
	r.POST("/contact", h.SubmitContact)

	body, _ := json.Marshal(model.ContactRequest{
		Name:           "John",
		Email:          "john@example.com",
		Subject:        "Hello",
		Message:        "Test message",
		TurnstileToken: "bad-token",
	})
	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitContact_ServiceError(t *testing.T) {
	h := newContactHandler(func(_ context.Context, _ *model.ContactRequest, _ string) error {
		return assert.AnError
	})

	r := newTestRouter()
	r.POST("/contact", h.SubmitContact)

	body, _ := json.Marshal(model.ContactRequest{
		Name:           "John",
		Email:          "john@example.com",
		Subject:        "Hello",
		Message:        "Test message",
		TurnstileToken: "valid-token",
	})

	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
