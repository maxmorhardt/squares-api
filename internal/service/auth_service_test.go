package service

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	service AuthService
)

func setup(_ *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-unit-tests")
	config.InitJWT()
	service = NewAuthService()
}

func TestIsDeclaredUser(t *testing.T) {
	setup(t)

	ctx := context.WithValue(context.Background(), model.UserKey, "john")

	assert.True(t, service.IsDeclaredUser(ctx, "john"))
	assert.False(t, service.IsDeclaredUser(ctx, "jane"))
}

func TestGenerateAndValidateInviteToken(t *testing.T) {
	setup(t)

	contestID := uuid.New()
	squareLimit := 10

	// generate token
	tokenString, err := service.GenerateInviteToken(contestID, squareLimit)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	// validate token
	claims, err := service.ValidateInviteToken(tokenString)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// verify claims
	assert.Equal(t, contestID, claims.ContestID)
	assert.Equal(t, squareLimit, claims.SquareLimit)
	assert.Equal(t, "squares-api", claims.Issuer)
}
