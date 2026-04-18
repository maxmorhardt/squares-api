package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStats_Success(t *testing.T) {
	svc := &mockStatsService{
		getStatsFn: func(_ context.Context) (*model.StatsResponse, error) {
			return &model.StatsResponse{ContestsCreatedToday: 5, SquaresClaimedToday: 42, TotalActiveContests: 12}, nil
		},
	}

	h := NewStatsHandler(svc)
	r := newTestRouter()
	r.GET("/stats", h.GetStats)

	req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.StatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(5), resp.ContestsCreatedToday)
	assert.Equal(t, int64(42), resp.SquaresClaimedToday)
	assert.Equal(t, int64(12), resp.TotalActiveContests)
}

func TestGetStats_Error(t *testing.T) {
	svc := &mockStatsService{
		getStatsFn: func(_ context.Context) (*model.StatsResponse, error) {
			return nil, assert.AnError
		},
	}

	h := NewStatsHandler(svc)
	r := newTestRouter()
	r.GET("/stats", h.GetStats)

	req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
