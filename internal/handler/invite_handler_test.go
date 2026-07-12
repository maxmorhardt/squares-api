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
// CreateInvite
// ====================

func TestCreateInvite_Success(t *testing.T) {
	svc := mocks.NewInviteService(t)
	svc.EXPECT().CreateInvite(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.ContestInvite{ID: uuid.New(), Token: "abc123"}, nil)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	w := doRequest(r, jsonReq(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), model.CreateInviteRequest{MaxSquares: 10, Role: "participant"}))
	assert.Equal(t, http.StatusCreated, w.Code)
	var resp model.InviteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "abc123", resp.Token)
}

func TestCreateInvite_InvalidContestID(t *testing.T) {
	h := NewInviteHandler(mocks.NewInviteService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	w := doRequest(r, jsonReq(http.MethodPost, "/contests/bad/invites", model.CreateInviteRequest{MaxSquares: 10, Role: "participant"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvite_InvalidBody(t *testing.T) {
	h := NewInviteHandler(mocks.NewInviteService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvite_Forbidden(t *testing.T) {
	createInviteErr(t, "stranger", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestCreateInvite_NotFound(t *testing.T) {
	createInviteErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestCreateInvite_InternalError(t *testing.T) {
	createInviteErr(t, "owner1", assert.AnError, http.StatusInternalServerError)
}

func createInviteErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().CreateInvite(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.POST("/contests/:id/invites", h.CreateInvite)

	w := doRequest(r, jsonReq(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), model.CreateInviteRequest{MaxSquares: 10, Role: "participant"}))
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// GetInvitePreview
// ====================

func TestGetInvitePreview_Success(t *testing.T) {
	svc := mocks.NewInviteService(t)
	svc.EXPECT().GetInvitePreview(mock.Anything, mock.Anything).
		Return(&model.InvitePreviewResponse{ContestName: "Super Bowl", Owner: "owner1", Role: "participant", MaxSquares: 10}, nil)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/abc123", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.InvitePreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Super Bowl", resp.ContestName)
}

func TestGetInvitePreview_NotFound(t *testing.T) {
	getInvitePreviewErr(t, errs.ErrInviteNotFound, http.StatusNotFound)
}
func TestGetInvitePreview_Expired(t *testing.T) {
	getInvitePreviewErr(t, errs.ErrInviteExpired, http.StatusBadRequest)
}
func TestGetInvitePreview_MaxUsesReached(t *testing.T) {
	getInvitePreviewErr(t, errs.ErrInviteMaxUsesReached, http.StatusBadRequest)
}
func TestGetInvitePreview_InternalError(t *testing.T) {
	getInvitePreviewErr(t, assert.AnError, http.StatusInternalServerError)
}

func getInvitePreviewErr(t *testing.T, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().GetInvitePreview(mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/tok", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// RedeemInvite
// ====================

func TestRedeemInvite_Success(t *testing.T) {
	svc := mocks.NewInviteService(t)
	svc.EXPECT().RedeemInvite(mock.Anything, mock.Anything, mock.Anything).
		Return(&model.ContestParticipant{ID: uuid.New(), UserID: "user1", Role: model.ParticipantRoleParticipant}, nil)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/valid-token/redeem", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestRedeemInvite_NotFound(t *testing.T) {
	redeemInviteErr(t, errs.ErrInviteNotFound, http.StatusNotFound)
}
func TestRedeemInvite_AlreadyParticipant(t *testing.T) {
	redeemInviteErr(t, errs.ErrAlreadyParticipant, http.StatusConflict)
}
func TestRedeemInvite_Expired(t *testing.T) {
	redeemInviteErr(t, errs.ErrInviteExpired, http.StatusBadRequest)
}
func TestRedeemInvite_MaxUsesReached(t *testing.T) {
	redeemInviteErr(t, errs.ErrInviteMaxUsesReached, http.StatusBadRequest)
}
func TestRedeemInvite_NotEnoughSquares(t *testing.T) {
	redeemInviteErr(t, errs.ErrNotEnoughSquares, http.StatusUnprocessableEntity)
}
func TestRedeemInvite_InternalError(t *testing.T) {
	redeemInviteErr(t, assert.AnError, http.StatusInternalServerError)
}

func redeemInviteErr(t *testing.T, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().RedeemInvite(mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/tok/redeem", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// GetInvites
// ====================

func TestGetInvites_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().GetInvitesByContestID(mock.Anything, mock.Anything, mock.Anything).
		Return([]model.ContestInvite{{ID: uuid.New(), ContestID: contestID, Token: "tok1"}}, nil)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", contestID), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp []model.ContestInvite
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestGetInvites_InvalidContestID(t *testing.T) {
	h := NewInviteHandler(mocks.NewInviteService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, "/contests/bad/invites", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetInvites_Forbidden(t *testing.T) {
	getInvitesErr(t, "stranger", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestGetInvites_NotFound(t *testing.T) {
	getInvitesErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestGetInvites_InternalError(t *testing.T) {
	getInvitesErr(t, "owner1", assert.AnError, http.StatusInternalServerError)
}

func getInvitesErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().GetInvitesByContestID(mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// DeleteInvite
// ====================

func TestDeleteInvite_Success(t *testing.T) {
	svc := mocks.NewInviteService(t)
	svc.EXPECT().DeleteInvite(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteInvite_InvalidContestID(t *testing.T) {
	deleteInviteBadID(t, fmt.Sprintf("/contests/bad/invites/%s", uuid.New()))
}
func TestDeleteInvite_InvalidInviteID(t *testing.T) {
	deleteInviteBadID(t, fmt.Sprintf("/contests/%s/invites/bad", uuid.New()))
}

func deleteInviteBadID(t *testing.T, target string) {
	t.Helper()
	h := NewInviteHandler(mocks.NewInviteService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, target, http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteInvite_Forbidden(t *testing.T) {
	deleteInviteErr(t, "stranger", errs.ErrInsufficientRole, http.StatusForbidden)
}
func TestDeleteInvite_NotFound(t *testing.T) {
	deleteInviteErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestDeleteInvite_InternalError(t *testing.T) {
	deleteInviteErr(t, "owner1", assert.AnError, http.StatusInternalServerError)
}

func deleteInviteErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewInviteService(t)
	svc.EXPECT().DeleteInvite(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(svcErr)
	h := NewInviteHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}
