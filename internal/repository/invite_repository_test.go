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

func createTestInvite(t *testing.T, repo InviteRepository, ctx context.Context, contestID uuid.UUID) *model.ContestInvite {
	t.Helper()

	invite := &model.ContestInvite{
		ContestID:  contestID,
		MaxSquares: 10,
		Role:       model.ParticipantRoleParticipant,
		CreatedBy:  "owner1",
		MaxUses:    5,
	}
	require.NoError(t, repo.Create(ctx, invite))
	return invite
}

func TestInviteRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	contestID := seedContest(t, db)

	invite := &model.ContestInvite{
		ContestID:  contestID,
		MaxSquares: 10,
		Role:       model.ParticipantRoleParticipant,
		CreatedBy:  "owner1",
		MaxUses:    5,
	}

	err := repo.Create(ctx, invite)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, invite.ID)
	assert.NotEmpty(t, invite.Token)
}

func TestInviteRepository_GetByToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	contestID := seedContest(t, db)
	invite := createTestInvite(t, repo, ctx, contestID)

	found, err := repo.GetByToken(ctx, invite.Token)
	require.NoError(t, err)
	assert.Equal(t, invite.ID, found.ID)
	assert.Equal(t, invite.Token, found.Token)
	assert.Equal(t, 10, found.MaxSquares)
}

func TestInviteRepository_GetByToken_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	_, err := repo.GetByToken(ctx, "nonexistent-token")
	assert.Error(t, err)
}

func TestInviteRepository_GetAllByContestID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	contestID := seedContest(t, db)
	createTestInvite(t, repo, ctx, contestID)
	createTestInvite(t, repo, ctx, contestID)

	invites, err := repo.GetAllByContestID(ctx, contestID)
	require.NoError(t, err)
	assert.Len(t, invites, 2)
}

func TestInviteRepository_GetAllByContestID_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	invites, err := repo.GetAllByContestID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, invites)
}

func TestInviteRepository_IncrementUses(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	contestID := seedContest(t, db)
	invite := createTestInvite(t, repo, ctx, contestID)
	assert.Equal(t, 0, invite.Uses)

	require.NoError(t, repo.IncrementUses(ctx, invite.ID))
	require.NoError(t, repo.IncrementUses(ctx, invite.ID))

	found, err := repo.GetByToken(ctx, invite.Token)
	require.NoError(t, err)
	assert.Equal(t, 2, found.Uses)
}

func TestInviteRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInviteRepository(db)
	ctx := context.Background()

	contestID := seedContest(t, db)
	invite := createTestInvite(t, repo, ctx, contestID)

	require.NoError(t, repo.Delete(ctx, invite.ID))

	_, err := repo.GetByToken(ctx, invite.Token)
	assert.Error(t, err)
}

// seedContest creates a minimal contest row needed for invite FK references
func seedContest(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()

	id := uuid.New()
	contest := model.Contest{
		ID:         id,
		Name:       "Invite Test",
		Owner:      "owner1",
		XLabels:    []byte("[]"),
		YLabels:    []byte("[]"),
		Visibility: model.ContestVisibilityPrivate,
		Status:     model.ContestStatusActive,
	}
	require.NoError(t, db.Create(&contest).Error)
	return id
}
