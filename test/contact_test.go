package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSubmitContact(t *testing.T) {
	req := model.ContactRequest{
		Name:           "Test User",
		Email:          "test@example.com",
		Subject:        "Test Subject",
		Message:        "This is a test message.",
		TurnstileToken: "test-token",
	}

	resp, _, _ := submitContact(router, &req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSubmitContact_Validation(t *testing.T) {
	cases := []struct {
		name           string
		request        model.ContactRequest
		expectedStatus int
	}{
		{
			name: "Valid_Submission",
			request: model.ContactRequest{
				Name:           "John Doe",
				Email:          "john@example.com",
				Subject:        "Valid Subject",
				Message:        "This is a valid test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Missing_Name",
			request: model.ContactRequest{
				Name:           "",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Too_Long",
			request: model.ContactRequest{
				Name:           strings.Repeat("A", 101),
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_With_Dangerous_Characters",
			request: model.ContactRequest{
				Name:           "John <script>alert('xss')</script>",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid_Email_Format",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "not-an-email",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing_Email",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Email_Too_Long",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          strings.Repeat("A", 256) + "@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Email_With_Dangerous_Characters",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com<script>",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing_Subject",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Subject_Too_Long",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        strings.Repeat("A", 201),
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Subject_With_Dangerous_Characters",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Question about {product}",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Subject_With_Script_Tag",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Help <script>alert(1)</script>",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing_Message",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Message_Too_Long",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        strings.Repeat("A", 2001),
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Message_With_Dangerous_Characters",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "Check out this code <script>alert('hack')</script>",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Message_With_Pipe_Character",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is | a test",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Missing_Turnstile_Token",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Min_Length_1",
			request: model.ContactRequest{
				Name:           "A",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Name_Max_Length_100",
			request: model.ContactRequest{
				Name:           strings.Repeat("A", 100),
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Name_Over_Max_101",
			request: model.ContactRequest{
				Name:           strings.Repeat("A", 101),
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Email_Max_Length_255",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          strings.Repeat("A", 245) + "@test.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Email_Over_Max_256",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          strings.Repeat("A", 256) + "@example.com",
				Subject:        "Test Subject",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Subject_Min_Length_1",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "S",
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Subject_Max_Length_200",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        strings.Repeat("A", 200),
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Subject_Over_Max_201",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        strings.Repeat("A", 201),
				Message:        "This is a test message.",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Message_Min_Length_1",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        "M",
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Message_Max_Length_2000",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        strings.Repeat("A", 2000),
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Message_Over_Max_2001",
			request: model.ContactRequest{
				Name:           "Test User",
				Email:          "test@example.com",
				Subject:        "Test Subject",
				Message:        strings.Repeat("A", 2001),
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "All_Fields_Valid_Max_Length",
			request: model.ContactRequest{
				Name:           strings.Repeat("A", 100),
				Email:          "valid@example.com",
				Subject:        strings.Repeat("A", 200),
				Message:        strings.Repeat("A", 2000),
				TurnstileToken: "test-token",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, _, _ := submitContact(router, &tc.request)
			assert.Equal(t, tc.expectedStatus, resp.StatusCode, "Test case: %s", tc.name)
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
