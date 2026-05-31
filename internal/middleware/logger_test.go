package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware_GeneratesRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var gotRequestID string
	var gotLogger any
	r.GET("/x", LoggerMiddleware, func(c *gin.Context) {
		gotRequestID = c.GetString(model.RequestIDKey)
		gotLogger, _ = c.Get(model.LoggerKey)
		c.Status(http.StatusOK)
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", http.NoBody))

	assert.NotEmpty(t, gotRequestID, "request id should be generated when header absent")
	_, ok := gotLogger.(*slog.Logger)
	assert.True(t, ok, "logger should be stored in context")
}

func TestLoggerMiddleware_UsesProvidedRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var gotRequestID string
	r.GET("/x", LoggerMiddleware, func(c *gin.Context) {
		gotRequestID = c.GetString(model.RequestIDKey)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set("X-Request-Id", "fixed-id")
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "fixed-id", gotRequestID)
}
