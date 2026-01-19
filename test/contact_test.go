package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSubmitContact(t *testing.T) {
	req := model.ContactRequest{
		Name:    "Test User",
		Email:   "test@example.com",
		Subject: "Test Subject",
		Message: "This is a test message.",
	}

	resp, body, _ := submitContact(router, &req)

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Contains(t, string(body), "Contact form submitted successfully")
}

func TestSubmitContact_Validation(t *testing.T) {
	cases := []struct {
		name           string
		request        model.ContactRequest
		expectedStatus int
	}{
		{
			name: "Missing Name",
			request: model.ContactRequest{
				Name:    "",
				Email:   "test@example.com",
				Subject: "Test Subject",
				Message: "This is a test message.",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Email",
			request: model.ContactRequest{
				Name:    "Test User",
				Email:   "not-an-email",
				Subject: "Test Subject",
				Message: "This is a test message.",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Message",
			request: model.ContactRequest{
				Name:    "Test User",
				Email:   "test@example.com",
				Subject: "Test Subject",
				Message: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _, _ := submitContact(router, &tc.request)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func submitContact(router http.Handler, reqBody *model.ContactRequest) (*http.Response, []byte, error) {
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/contact", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Result(), w.Body.Bytes(), nil
}