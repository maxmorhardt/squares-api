package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetLeaderboard_Success(t *testing.T) {
	svc := mocks.NewLeaderboardService(t)
	svc.EXPECT().GetLeaderboard(mock.Anything, service.DefaultLeaderboardLimit).Return(&model.LeaderboardResponse{
		Entries: []model.LeaderboardEntry{
			{Rank: 1, DisplayName: "Max", QuarterWins: 12, SquaresClaimed: 48},
		},
	}, nil)

	r := gin.New()
	r.GET("/leaderboard", NewLeaderboardHandler(svc).GetLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.LeaderboardResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Entries, 1)
	assert.Equal(t, 1, resp.Entries[0].Rank)
	assert.Equal(t, "Max", resp.Entries[0].DisplayName)
	// the response must never carry the email that identifies the user
	assert.NotContains(t, w.Body.String(), "@")
}

func TestGetLeaderboard_CustomLimit(t *testing.T) {
	svc := mocks.NewLeaderboardService(t)
	svc.EXPECT().GetLeaderboard(mock.Anything, 5).Return(&model.LeaderboardResponse{}, nil)

	r := gin.New()
	r.GET("/leaderboard", NewLeaderboardHandler(svc).GetLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard?limit=5", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func getLeaderboardBadRequest(t *testing.T, query string) {
	t.Helper()

	svc := mocks.NewLeaderboardService(t)

	r := gin.New()
	r.GET("/leaderboard", NewLeaderboardHandler(svc).GetLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard?"+query, http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetLeaderboard_NonNumericLimit(t *testing.T) { getLeaderboardBadRequest(t, "limit=abc") }
func TestGetLeaderboard_ZeroLimit(t *testing.T)       { getLeaderboardBadRequest(t, "limit=0") }
func TestGetLeaderboard_NegativeLimit(t *testing.T)   { getLeaderboardBadRequest(t, "limit=-5") }
func TestGetLeaderboard_LimitTooLarge(t *testing.T)   { getLeaderboardBadRequest(t, "limit=101") }

func TestGetLeaderboard_Error(t *testing.T) {
	svc := mocks.NewLeaderboardService(t)
	svc.EXPECT().GetLeaderboard(mock.Anything, service.DefaultLeaderboardLimit).Return(nil, assert.AnError)

	r := gin.New()
	r.GET("/leaderboard", NewLeaderboardHandler(svc).GetLeaderboard)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetMyRank_Success(t *testing.T) {
	svc := mocks.NewLeaderboardService(t)
	svc.EXPECT().GetUserRank(mock.Anything, "user@example.com").Return(&model.LeaderboardRankResponse{
		Rank: 7, TotalRanked: 143, QuarterWins: 5, Ranked: true,
	}, nil)

	r := gin.New()
	r.GET("/leaderboard/me", authenticatedMiddleware("user@example.com"), NewLeaderboardHandler(svc).GetMyRank)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.LeaderboardRankResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 7, resp.Rank)
	assert.Equal(t, int64(143), resp.TotalRanked)
	assert.True(t, resp.Ranked)
}

func TestGetMyRank_Error(t *testing.T) {
	svc := mocks.NewLeaderboardService(t)
	svc.EXPECT().GetUserRank(mock.Anything, "user@example.com").Return(nil, assert.AnError)

	r := gin.New()
	r.GET("/leaderboard/me", authenticatedMiddleware("user@example.com"), NewLeaderboardHandler(svc).GetMyRank)

	req, _ := http.NewRequest(http.MethodGet, "/leaderboard/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
