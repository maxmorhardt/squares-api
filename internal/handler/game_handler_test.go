package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetUpcomingGames_Success(t *testing.T) {
	svc := mocks.NewGameService(t)
	svc.EXPECT().GetUpcoming(mock.Anything).Return([]model.Game{{ESPNID: "1"}, {ESPNID: "2"}}, nil)
	h := NewGameHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("u"))
	r.GET("/games/upcoming", h.GetUpcoming)

	w := doRequest(r, jsonReq(http.MethodGet, "/games/upcoming", nil))
	assert.Equal(t, http.StatusOK, w.Code)
	var games []model.Game
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &games))
	assert.Len(t, games, 2)
}

func TestGetUpcomingGames_Error(t *testing.T) {
	svc := mocks.NewGameService(t)
	svc.EXPECT().GetUpcoming(mock.Anything).Return(nil, errs.ErrDatabaseUnavailable)
	h := NewGameHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("u"))
	r.GET("/games/upcoming", h.GetUpcoming)

	w := doRequest(r, jsonReq(http.MethodGet, "/games/upcoming", nil))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
