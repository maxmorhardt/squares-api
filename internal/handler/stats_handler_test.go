package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetStats_Success(t *testing.T) {
	svc := mocks.NewStatsService(t)
	svc.EXPECT().GetStats(mock.Anything).Return(&model.StatsResponse{ContestsCreatedToday: 5, SquaresClaimedToday: 42, TotalActiveContests: 12}, nil)

	r := gin.New()
	r.GET("/stats", NewStatsHandler(svc).GetStats)

	req, _ := http.NewRequest(http.MethodGet, "/stats", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.StatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(5), resp.ContestsCreatedToday)
	assert.Equal(t, int64(42), resp.SquaresClaimedToday)
	assert.Equal(t, int64(12), resp.TotalActiveContests)
}

func TestGetStats_Error(t *testing.T) {
	svc := mocks.NewStatsService(t)
	svc.EXPECT().GetStats(mock.Anything).Return(nil, assert.AnError)

	r := gin.New()
	r.GET("/stats", NewStatsHandler(svc).GetStats)

	req, _ := http.NewRequest(http.MethodGet, "/stats", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
