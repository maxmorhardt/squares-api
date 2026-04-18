package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestContest(t *testing.T, repo ContestRepository, ctx context.Context, name string) *model.Contest {
	t.Helper()

	labels, _ := json.Marshal([]int8{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1})
	contest := &model.Contest{
		Name:       name,
		Owner:      "owner1",
		HomeTeam:   "Team A",
		AwayTeam:   "Team B",
		XLabels:    labels,
		YLabels:    labels,
		Visibility: model.ContestVisibilityPrivate,
		Status:     model.ContestStatusActive,
	}
	require.NoError(t, repo.Create(ctx, contest))
	return contest
}

func TestContestRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Test Contest")

	assert.NotEqual(t, uuid.Nil, contest.ID)
	assert.Equal(t, "Test Contest", contest.Name)

	// verify 100 squares were created
	found, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)
	assert.Len(t, found.Squares, 100)
}

func TestContestRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Test Contest")

	found, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)
	assert.Equal(t, contest.ID, found.ID)
	assert.Equal(t, "Test Contest", found.Name)
	assert.Len(t, found.Squares, 100)
}

func TestContestRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestContestRepository_GetByID_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "To Delete")
	require.NoError(t, repo.Delete(ctx, contest.ID))

	_, err := repo.GetByID(ctx, contest.ID)
	assert.Error(t, err)
}

func TestContestRepository_GetByOwnerAndName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	createTestContest(t, repo, ctx, "My Contest")

	found, err := repo.GetByOwnerAndName(ctx, "owner1", "My Contest")
	require.NoError(t, err)
	assert.Equal(t, "My Contest", found.Name)
	assert.Equal(t, "owner1", found.Owner)
}

func TestContestRepository_GetByOwnerAndName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	_, err := repo.GetByOwnerAndName(ctx, "nobody", "Nothing")
	assert.Error(t, err)
}

func TestContestRepository_ExistsByOwnerAndName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	createTestContest(t, repo, ctx, "Exists")

	exists, err := repo.ExistsByOwnerAndName(ctx, "owner1", "Exists")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByOwnerAndName(ctx, "owner1", "Nope")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestContestRepository_ExistsByOwnerAndName_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "DeleteMe")
	require.NoError(t, repo.Delete(ctx, contest.ID))

	exists, err := repo.ExistsByOwnerAndName(ctx, "owner1", "DeleteMe")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestContestRepository_GetAllByOwnerPaginated(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.WithValue(context.Background(), model.UserKey, "owner1")

	for i := range 5 {
		createTestContest(t, repo, ctx, "Contest"+string(rune('A'+i)))
	}

	// page 1, limit 2
	contests, total, err := repo.GetAllByOwnerPaginated(ctx, "owner1", 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, contests, 2)

	// page 3, limit 2 — should get 1 remaining
	contests, total, err = repo.GetAllByOwnerPaginated(ctx, "owner1", 3, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, contests, 1)
}

func TestContestRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Original")
	contest.HomeTeam = "Updated Team"
	require.NoError(t, repo.Update(ctx, contest))

	found, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Team", found.HomeTeam)
}

func TestContestRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Delete Me")
	require.NoError(t, repo.Delete(ctx, contest.ID))

	// should not be found via GetByID (excludes deleted)
	_, err := repo.GetByID(ctx, contest.ID)
	assert.Error(t, err)

	// verify it's a soft delete — status changed to DELETED
	var raw model.Contest
	require.NoError(t, db.First(&raw, "id = ?", contest.ID).Error)
	assert.Equal(t, model.ContestStatusDeleted, raw.Status)
}

func TestContestRepository_CreateQuarterResult(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "QR Contest")

	result := &model.QuarterResult{
		ContestID:     contest.ID,
		Quarter:       1,
		HomeTeamScore: 14,
		AwayTeamScore: 7,
		WinnerRow:     3,
		WinnerCol:     4,
		Winner:        "user1",
		WinnerName:    "John",
	}
	require.NoError(t, repo.CreateQuarterResult(ctx, result))
	assert.NotEqual(t, uuid.Nil, result.ID)

	// verify it's preloaded on fetch
	found, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)
	assert.Len(t, found.QuarterResults, 1)
	assert.Equal(t, 1, found.QuarterResults[0].Quarter)
}

func TestContestRepository_GetSquareByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Square Contest")
	full, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)

	square, err := repo.GetSquareByID(ctx, full.Squares[0].ID)
	require.NoError(t, err)
	assert.Equal(t, full.Squares[0].ID, square.ID)
}

func TestContestRepository_UpdateSquare(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Update Square")
	full, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)

	square := &full.Squares[0]
	updated, err := repo.UpdateSquare(ctx, square, "ABC", "user1", "John")
	require.NoError(t, err)
	assert.Equal(t, "ABC", updated.Value)
	assert.Equal(t, "user1", updated.Owner)
	assert.Equal(t, "John", updated.OwnerName)
}

func TestContestRepository_ClearSquare(t *testing.T) {
	db := setupTestDB(t)
	repo := NewContestRepository(db)
	ctx := context.Background()

	contest := createTestContest(t, repo, ctx, "Clear Square")
	full, err := repo.GetByID(ctx, contest.ID)
	require.NoError(t, err)

	// claim square first
	square := &full.Squares[0]
	_, err = repo.UpdateSquare(ctx, square, "XYZ", "user1", "John")
	require.NoError(t, err)

	// clear it
	cleared, err := repo.ClearSquare(ctx, square)
	require.NoError(t, err)
	assert.Equal(t, "", cleared.Value)
	assert.Equal(t, "", cleared.Owner)
	assert.Equal(t, "", cleared.OwnerName)
}
