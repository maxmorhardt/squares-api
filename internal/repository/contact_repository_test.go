package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContactRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	submission := &model.ContactSubmission{
		Name:    "John Doe",
		Email:   "john@example.com",
		Subject: "Test Subject",
		Message: "Test Message",
	}

	err := repo.Create(ctx, submission)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, submission.ID)
	assert.Equal(t, "pending", submission.Status)
}
