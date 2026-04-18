package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsRepository_GetStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStatsRepository(db)
	ctx := context.Background()

	// Create active contest (should count toward ContestsCreatedToday and TotalActiveContests)
	c1 := model.Contest{
		ID:         uuid.New(),
		Name:       "Active Contest",
		Owner:      "owner1",
		XLabels:    []byte("[]"),
		YLabels:    []byte("[]"),
		Visibility: model.ContestVisibilityPublic,
		Status:     model.ContestStatusActive,
	}
	require.NoError(t, db.Create(&c1).Error)

	// Create deleted contest (should NOT count)
	c2 := model.Contest{
		ID:         uuid.New(),
		Name:       "Deleted Contest",
		Owner:      "owner1",
		XLabels:    []byte("[]"),
		YLabels:    []byte("[]"),
		Visibility: model.ContestVisibilityPublic,
		Status:     model.ContestStatusDeleted,
	}
	require.NoError(t, db.Create(&c2).Error)

	// Create a square with an owner (claimed today)
	s1 := model.Square{ContestID: c1.ID, Row: 0, Col: 0, Value: "ABC", Owner: "user1"}
	require.NoError(t, db.Create(&s1).Error)

	// Create a square without an owner (not claimed)
	s2 := model.Square{ContestID: c1.ID, Row: 0, Col: 1, Value: "", Owner: ""}
	require.NoError(t, db.Create(&s2).Error)

	stats, err := repo.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(1), stats.ContestsCreatedToday)
	assert.Equal(t, int64(1), stats.SquaresClaimedToday)
	assert.Equal(t, int64(1), stats.TotalActiveContests)
}

func TestStatsRepository_GetStats_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := NewStatsRepository(db)
	ctx := context.Background()

	stats, err := repo.GetStats(ctx)
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.ContestsCreatedToday)
	assert.Equal(t, int64(0), stats.SquaresClaimedToday)
	assert.Equal(t, int64(0), stats.TotalActiveContests)
}
