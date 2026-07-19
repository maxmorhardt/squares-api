package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ====================
// GetParticipants
// ====================

func TestGetParticipants_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().GetParticipants(mock.Anything, mock.Anything, mock.Anything).Return([]model.ContestParticipant{
		{ID: uuid.New(), ContestID: contestID, UserID: "owner1", Role: model.ParticipantRoleOwner},
		{ID: uuid.New(), ContestID: contestID, UserID: "user1", Role: model.ParticipantRoleParticipant},
	}, nil)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", contestID), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp []model.ContestParticipant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestGetParticipants_InvalidID(t *testing.T) {
	h := NewParticipantHandler(mocks.NewParticipantService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, "/contests/bad-id/participants", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func getParticipantsErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().GetParticipants(mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

func TestGetParticipants_NotFound(t *testing.T) {
	getParticipantsErr(t, "user1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestGetParticipants_Forbidden(t *testing.T) {
	getParticipantsErr(t, "stranger", errs.ErrNotParticipant, http.StatusForbidden)
}
func TestGetParticipants_InsufficientRole(t *testing.T) {
	getParticipantsErr(t, "viewer", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestGetParticipants_InternalError(t *testing.T) {
	getParticipantsErr(t, "user1", assert.AnError, http.StatusInternalServerError)
}

// ====================
// GetMyContests
// ====================

func TestGetMyContests_Success(t *testing.T) {
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().GetMyContests(mock.Anything, mock.Anything, mock.Anything).Return([]model.Contest{{ID: uuid.New(), Name: "MyContest"}}, nil)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/me", h.GetMyContests)

	req, _ := http.NewRequest(http.MethodGet, "/contests/me", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp []model.Contest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestGetMyContests_Error(t *testing.T) {
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().GetMyContests(mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/me", h.GetMyContests)

	req, _ := http.NewRequest(http.MethodGet, "/contests/me", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func getMyContestsSearch(t *testing.T, query, wantSearch string) {
	t.Helper()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().GetMyContests(mock.Anything, mock.Anything, wantSearch).Return([]model.Contest{}, nil)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/me", h.GetMyContests)

	req, _ := http.NewRequest(http.MethodGet, "/contests/me?"+query, http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetMyContests_PassesSearchQuery(t *testing.T) { getMyContestsSearch(t, "search=foo", "foo") }
func TestGetMyContests_TrimsSearchQuery(t *testing.T) {
	getMyContestsSearch(t, "search=%20%20bar%20%20", "bar")
}

// ====================
// UpdateParticipant
// ====================

func TestUpdateParticipant_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().UpdateParticipant(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.ContestParticipant{ID: uuid.New(), ContestID: contestID, UserID: "user1", Role: model.ParticipantRoleViewer}, nil)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", contestID), model.UpdateParticipantRequest{Role: &role}))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateParticipant_InvalidContestID(t *testing.T) {
	h := NewParticipantHandler(mocks.NewParticipantService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	req, _ := http.NewRequest(http.MethodPatch, "/contests/bad/participants/user1", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateParticipant_InvalidBody(t *testing.T) {
	h := NewParticipantHandler(mocks.NewParticipantService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func updateParticipantErr(t *testing.T, user, target string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().UpdateParticipant(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/%s", uuid.New(), target), model.UpdateParticipantRequest{Role: &role}))
	assert.Equal(t, wantCode, w.Code)
}

func TestUpdateParticipant_Forbidden(t *testing.T) {
	updateParticipantErr(t, "stranger", "user1", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestUpdateParticipant_CannotChangeOwner(t *testing.T) {
	updateParticipantErr(t, "owner1", "owner1", errs.ErrCannotChangeOwner, http.StatusForbidden)
}
func TestUpdateParticipant_SquareLimitTooLow(t *testing.T) {
	updateParticipantErr(t, "owner1", "user1", errs.ErrSquareLimitTooLow, http.StatusBadRequest)
}
func TestUpdateParticipant_NotParticipant(t *testing.T) {
	updateParticipantErr(t, "owner1", "unknown", errs.ErrNotParticipant, http.StatusNotFound)
}
func TestUpdateParticipant_InternalError(t *testing.T) {
	updateParticipantErr(t, "owner1", "user1", assert.AnError, http.StatusInternalServerError)
}

// ====================
// RemoveParticipant
// ====================

func TestRemoveParticipant_Success(t *testing.T) {
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().RemoveParticipant(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRemoveParticipant_InvalidID(t *testing.T) {
	h := NewParticipantHandler(mocks.NewParticipantService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, "/contests/bad/participants/user1", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func removeParticipantErr(t *testing.T, user, target string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewParticipantService(t)
	svc.EXPECT().RemoveParticipant(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(svcErr)
	h := NewParticipantHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/%s", uuid.New(), target), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

func TestRemoveParticipant_Forbidden(t *testing.T) {
	removeParticipantErr(t, "stranger", "user1", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestRemoveParticipant_CannotRemoveOwner(t *testing.T) {
	removeParticipantErr(t, "owner1", "owner1", errs.ErrCannotRemoveOwner, http.StatusForbidden)
}
func TestRemoveParticipant_Finalized(t *testing.T) {
	removeParticipantErr(t, "user1", "user1", errs.ErrContestFinalized, http.StatusForbidden)
}
func TestRemoveParticipant_NotFound(t *testing.T) {
	removeParticipantErr(t, "owner1", "nobody", errs.ErrNotParticipant, http.StatusNotFound)
}
func TestRemoveParticipant_InternalError(t *testing.T) {
	removeParticipantErr(t, "owner1", "user1", assert.AnError, http.StatusInternalServerError)
}
