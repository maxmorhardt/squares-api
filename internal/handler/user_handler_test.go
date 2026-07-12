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

func newUserRouter(t *testing.T) (*mocks.UserService, *gin.Engine) {
	t.Helper()
	svc := mocks.NewUserService(t)
	h := NewUserHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("a@b.com"))
	r.GET("/users/me", h.GetMe)
	r.DELETE("/users/me", h.DeleteMe)
	r.GET("/users/me/stats", h.GetMyStats)

	return svc, r
}

// ====================
// GetMe
// ====================

func TestGetMe_Success(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().GetProfile(mock.Anything, "a@b.com", "Test").
		Return(&model.User{Email: "a@b.com", DisplayName: "Max"}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/users/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.UserProfileResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "a@b.com", resp.Email)
	assert.Equal(t, "Max", resp.DisplayName)
}

func TestGetMe_ServiceError(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().GetProfile(mock.Anything, "a@b.com", "Test").
		Return(nil, errs.ErrDatabaseUnavailable)

	req, _ := http.NewRequest(http.MethodGet, "/users/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ====================
// GetMyStats
// ====================

func TestGetMyStats_Success(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(&model.UserStatsResponse{ContestsCreated: 2, ContestsJoined: 4, SquaresClaimed: 10, QuarterWins: 1}, nil)

	req, _ := http.NewRequest(http.MethodGet, "/users/me/stats", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.UserStatsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(2), resp.ContestsCreated)
	assert.Equal(t, int64(10), resp.SquaresClaimed)
}

func TestGetMyStats_ServiceError(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(nil, errs.ErrDatabaseUnavailable)

	req, _ := http.NewRequest(http.MethodGet, "/users/me/stats", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ====================
// DeleteMe
// ====================

func TestDeleteMe_Success(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().DeleteAccount(mock.Anything, "a@b.com").Return(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/users/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteMe_ServiceError(t *testing.T) {
	svc, r := newUserRouter(t)
	svc.EXPECT().DeleteAccount(mock.Anything, "a@b.com").Return(errs.ErrDatabaseUnavailable)

	req, _ := http.NewRequest(http.MethodDelete, "/users/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
