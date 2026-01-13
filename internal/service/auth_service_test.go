package service

import (
	"context"
	"os"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
)

var (
	service AuthService
)

func setup(_ *testing.T) {
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	service = NewAuthService()
}

func TestIsDeclaredUser(t *testing.T) {
	setup(t)

	ctx := context.WithValue(context.Background(), model.UserKey, "john")

	assert.True(t, service.IsDeclaredUser(ctx, "john"))
	assert.False(t, service.IsDeclaredUser(ctx, "jane"))
}
