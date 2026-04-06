package util

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

func newTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestLoggerFromContext_WithLogger(t *testing.T) {
	logger := slog.Default()
	ctx := context.WithValue(context.Background(), model.LoggerKey, logger)

	result := LoggerFromContext(ctx)

	assert.Equal(t, logger, result)
}

func TestLoggerFromContext_WithoutLogger(t *testing.T) {
	ctx := context.Background()

	result := LoggerFromContext(ctx)

	assert.Equal(t, slog.Default(), result)
}

func TestLoggerFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), model.LoggerKey, "not a logger")

	result := LoggerFromContext(ctx)

	assert.Equal(t, slog.Default(), result)
}

func TestLoggerFromGinContext_WithLogger(t *testing.T) {
	c := newTestGinContext()
	logger := slog.Default()
	c.Set(model.LoggerKey, logger)

	result := LoggerFromGinContext(c)

	assert.Equal(t, logger, result)
}

func TestLoggerFromGinContext_WithoutLogger(t *testing.T) {
	c := newTestGinContext()

	result := LoggerFromGinContext(c)

	assert.Equal(t, slog.Default(), result)
}

func TestClaimsFromGinContext_WithClaims(t *testing.T) {
	c := newTestGinContext()
	claims := &model.Claims{Username: "testuser"}
	c.Set(model.ClaimsKey, claims)

	result := ClaimsFromGinContext(c)

	assert.Equal(t, claims, result)
}

func TestClaimsFromGinContext_WithoutClaims(t *testing.T) {
	c := newTestGinContext()

	result := ClaimsFromGinContext(c)

	assert.Nil(t, result)
}

func TestClaimsFromContext_WithClaims(t *testing.T) {
	claims := &model.Claims{Username: "testuser"}
	ctx := context.WithValue(context.Background(), model.ClaimsKey, claims)

	result := ClaimsFromContext(ctx)

	assert.Equal(t, claims, result)
}

func TestClaimsFromContext_WithoutClaims(t *testing.T) {
	ctx := context.Background()

	result := ClaimsFromContext(ctx)

	assert.Nil(t, result)
}

func TestClaimsFromContext_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), model.ClaimsKey, "not claims")

	result := ClaimsFromContext(ctx)

	assert.Nil(t, result)
}

func TestSetGinContextValue(t *testing.T) {
	c := newTestGinContext()

	SetGinContextValue(c, model.LoggerKey, "test-value")

	val, exists := c.Get(model.LoggerKey)
	assert.True(t, exists)
	assert.Equal(t, "test-value", val)

	ctxVal := c.Request.Context().Value(model.LoggerKey)
	assert.Equal(t, "test-value", ctxVal)
}
