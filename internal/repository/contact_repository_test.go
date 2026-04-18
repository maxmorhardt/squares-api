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

func TestContactRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	submission := &model.ContactSubmission{
		Name:    "John Doe",
		Email:   "john@example.com",
		Subject: "Test Subject",
		Message: "Test Message",
	}
	require.NoError(t, repo.Create(ctx, submission))

	found, err := repo.GetByID(ctx, submission.ID)
	require.NoError(t, err)
	assert.Equal(t, submission.ID, found.ID)
	assert.Equal(t, "John Doe", found.Name)
	assert.Equal(t, "john@example.com", found.Email)
}

func TestContactRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestContactRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContactRepository(db)
	ctx := context.Background()

	submission := &model.ContactSubmission{
		Name:    "John Doe",
		Email:   "john@example.com",
		Subject: "Test Subject",
		Message: "Test Message",
	}
	require.NoError(t, repo.Create(ctx, submission))

	submission.Status = "resolved"
	submission.Response = "Thanks for reaching out"
	err := repo.Update(ctx, submission)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, submission.ID)
	require.NoError(t, err)
	assert.Equal(t, "resolved", found.Status)
	assert.Equal(t, "Thanks for reaching out", found.Response)
}
