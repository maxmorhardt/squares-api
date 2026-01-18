package test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubmitContact(t *testing.T) {
	h := &ContactTestHelper{Router: router}

	validReq := map[string]interface{}{
		"name":    "Test User",
		"email":   "test@example.com",
		"subject": "Test Subject",
		"message": "This is a test message.",
	}
	resp, body, _ := h.SubmitContact(validReq)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Contains(t, string(body), "Contact form submitted successfully")
}

func TestSubmitContact_Validation(t *testing.T) {
	h := &ContactTestHelper{Router: router}

	cases := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
	}{
		{
			name: "Missing Name",
			request: map[string]interface{}{
				"email":   "test@example.com",
				"subject": "Test Subject",
				"message": "This is a test message.",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Email",
			request: map[string]interface{}{
				"name":    "Test User",
				"email":   "not-an-email",
				"subject": "Test Subject",
				"message": "This is a test message.",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing Message",
			request: map[string]interface{}{
				"name":    "Test User",
				"email":   "test@example.com",
				"subject": "Test Subject",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _, _ := h.SubmitContact(tc.request)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}