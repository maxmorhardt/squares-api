package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSubmitContact_Success(t *testing.T) {
	svc := mocks.NewContactService(t)
	svc.EXPECT().SubmitContact(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	r := gin.New()
	r.POST("/contact", NewContactHandler(svc).SubmitContact)
	w := doRequest(r, postContact(validContactBody()))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.ContactResponse

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Message, "submitted successfully")
}

func doRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func postContact(body []byte) *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func validContactBody() []byte {
	body, _ := json.Marshal(model.ContactRequest{
		Name:           "John",
		Email:          "john@example.com",
		Subject:        "Hello",
		Message:        "Test message",
		TurnstileToken: "valid-token",
	})
	return body
}

func TestSubmitContact_InvalidBody(t *testing.T) {
	svc := mocks.NewContactService(t)

	r := gin.New()
	r.POST("/contact", NewContactHandler(svc).SubmitContact)
	w := doRequest(r, postContact([]byte(`{invalid`)))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitContact_InvalidTurnstile(t *testing.T) {
	svc := mocks.NewContactService(t)
	svc.EXPECT().SubmitContact(mock.Anything, mock.Anything, mock.Anything).Return(errs.ErrInvalidTurnstile)

	r := gin.New()
	r.POST("/contact", NewContactHandler(svc).SubmitContact)
	w := doRequest(r, postContact(validContactBody()))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitContact_ServiceError(t *testing.T) {
	svc := mocks.NewContactService(t)
	svc.EXPECT().SubmitContact(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	r := gin.New()
	r.POST("/contact", NewContactHandler(svc).SubmitContact)
	w := doRequest(r, postContact(validContactBody()))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
