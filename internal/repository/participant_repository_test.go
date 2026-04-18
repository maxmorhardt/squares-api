package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedContestForParticipant(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	contest := model.Contest{
		ID:         id,
		Name:       "Participant Test",
		Owner:      "owner1",
		XLabels:    []byte("[]"),
		YLabels:    []byte("[]"),
		Visibility: model.ContestVisibilityPrivate,
		Status:     model.ContestStatusActive,
	}
	require.NoError(t, db.Create(&contest).Error)
	return id
}

func createTestParticipant(t *testing.T, repo ParticipantRepository, ctx context.Context, contestID uuid.UUID, userID string, role model.ParticipantRole, maxSquares int) *model.ContestParticipant {
	t.Helper()

	p := &model.ContestParticipant{
		ContestID:  contestID,
		UserID:     userID,
		Role:       role,
		MaxSquares: maxSquares,
	}
	require.NoError(t, repo.Create(ctx, p))
	return p
}

func TestParticipantRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	p := createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleOwner, 100)

	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, "user1", p.UserID)
	assert.Equal(t, model.ParticipantRoleOwner, p.Role)
}

func TestParticipantRepository_GetByContestAndUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleOwner, 100)

	found, err := repo.GetByContestAndUser(ctx, contestID, "user1")
	require.NoError(t, err)
	assert.Equal(t, "user1", found.UserID)
	assert.Equal(t, model.ParticipantRoleOwner, found.Role)
}

func TestParticipantRepository_GetByContestAndUser_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	_, err := repo.GetByContestAndUser(ctx, uuid.New(), "nobody")
	assert.Error(t, err)
}

func TestParticipantRepository_GetAllByContestID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	createTestParticipant(t, repo, ctx, contestID, "owner1", model.ParticipantRoleOwner, 100)
	createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleParticipant, 10)
	createTestParticipant(t, repo, ctx, contestID, "user2", model.ParticipantRoleViewer, 0)

	participants, err := repo.GetAllByContestID(ctx, contestID)
	require.NoError(t, err)
	assert.Len(t, participants, 3)
}

func TestParticipantRepository_GetAllByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)

	// owner role is excluded from GetAllByUserID
	createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleOwner, 100)

	contestID2 := seedContestForParticipant(t, db)
	createTestParticipant(t, repo, ctx, contestID2, "user1", model.ParticipantRoleParticipant, 10)

	participants, err := repo.GetAllByUserID(ctx, "user1")
	require.NoError(t, err)
	// should exclude the owner role entry
	assert.Len(t, participants, 1)
	assert.Equal(t, model.ParticipantRoleParticipant, participants[0].Role)
}

func TestParticipantRepository_GetTotalAllocatedSquares(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	createTestParticipant(t, repo, ctx, contestID, "owner1", model.ParticipantRoleOwner, 100)
	createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleParticipant, 10)
	createTestParticipant(t, repo, ctx, contestID, "user2", model.ParticipantRoleParticipant, 15)

	// should only sum non-owner participants
	total, err := repo.GetTotalAllocatedSquares(ctx, contestID)
	require.NoError(t, err)
	assert.Equal(t, 25, total)
}

func TestParticipantRepository_CountSquaresByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)

	// seed squares — one claimed by user1, one unclaimed
	s1 := model.Square{ContestID: contestID, Row: 0, Col: 0, Value: "ABC", Owner: "user1"}
	s2 := model.Square{ContestID: contestID, Row: 0, Col: 1, Value: "", Owner: ""}
	s3 := model.Square{ContestID: contestID, Row: 0, Col: 2, Value: "XYZ", Owner: "user1"}
	require.NoError(t, db.Create(&s1).Error)
	require.NoError(t, db.Create(&s2).Error)
	require.NoError(t, db.Create(&s3).Error)

	count, err := repo.CountSquaresByUser(ctx, contestID, "user1")
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestParticipantRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	p := createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleParticipant, 10)

	p.Role = model.ParticipantRoleViewer
	p.MaxSquares = 0
	require.NoError(t, repo.Update(ctx, p))

	found, err := repo.GetByContestAndUser(ctx, contestID, "user1")
	require.NoError(t, err)
	assert.Equal(t, model.ParticipantRoleViewer, found.Role)
	assert.Equal(t, 0, found.MaxSquares)
}

func TestParticipantRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewParticipantRepository(db)
	ctx := context.Background()

	contestID := seedContestForParticipant(t, db)
	createTestParticipant(t, repo, ctx, contestID, "user1", model.ParticipantRoleParticipant, 10)

	require.NoError(t, repo.Delete(ctx, contestID, "user1"))

	_, err := repo.GetByContestAndUser(ctx, contestID, "user1")
	assert.Error(t, err)
}
