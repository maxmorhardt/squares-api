package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func defaultMockInviteService() *mockInviteService {
	return &mockInviteService{}
}

// ====================
// CreateInvite
// ====================

func TestCreateInvite_Success(t *testing.T) {
	svc := defaultMockInviteService()
	svc.createInviteFn = func(_ context.Context, _ uuid.UUID, _ *model.CreateInviteRequest, _ string) (*model.ContestInvite, error) {
		return &model.ContestInvite{ID: uuid.New(), Token: "abc123"}, nil
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	body, _ := json.Marshal(model.CreateInviteRequest{MaxSquares: 10, Role: "participant"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp model.InviteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "abc123", resp.Token)
}

func TestCreateInvite_InvalidContestID(t *testing.T) {
	svc := defaultMockInviteService()
	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	body, _ := json.Marshal(model.CreateInviteRequest{MaxSquares: 10, Role: "participant"})
	req, _ := http.NewRequest(http.MethodPost, "/contests/bad/invites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvite_InvalidBody(t *testing.T) {
	svc := defaultMockInviteService()
	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateInvite_Forbidden(t *testing.T) {
	svc := defaultMockInviteService()
	svc.createInviteFn = func(_ context.Context, _ uuid.UUID, _ *model.CreateInviteRequest, _ string) (*model.ContestInvite, error) {
		return nil, errs.ErrInsufficientRole
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	body, _ := json.Marshal(model.CreateInviteRequest{MaxSquares: 10, Role: "participant"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateInvite_NotFound(t *testing.T) {
	svc := defaultMockInviteService()
	svc.createInviteFn = func(_ context.Context, _ uuid.UUID, _ *model.CreateInviteRequest, _ string) (*model.ContestInvite, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	body, _ := json.Marshal(model.CreateInviteRequest{MaxSquares: 10, Role: "participant"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ====================
// GetInvitePreview
// ====================

func TestGetInvitePreview_Success(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitePreviewFn = func(_ context.Context, _ string) (*model.InvitePreviewResponse, error) {
		return &model.InvitePreviewResponse{ContestName: "Super Bowl", Owner: "owner1", Role: "participant", MaxSquares: 10}, nil
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/abc123", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.InvitePreviewResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Super Bowl", resp.ContestName)
}

func TestGetInvitePreview_NotFound(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitePreviewFn = func(_ context.Context, _ string) (*model.InvitePreviewResponse, error) {
		return nil, errs.ErrInviteNotFound
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/missing", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetInvitePreview_Expired(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitePreviewFn = func(_ context.Context, _ string) (*model.InvitePreviewResponse, error) {
		return nil, errs.ErrInviteExpired
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/expired-token", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusGone, w.Code)
}

func TestGetInvitePreview_MaxUsesReached(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitePreviewFn = func(_ context.Context, _ string) (*model.InvitePreviewResponse, error) {
		return nil, errs.ErrInviteMaxUsesReached
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/used-up-token", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusGone, w.Code)
}

// ====================
// RedeemInvite
// ====================

func TestRedeemInvite_Success(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return &model.ContestParticipant{ID: uuid.New(), UserID: "user1", Role: model.ParticipantRoleParticipant}, nil
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/valid-token/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestRedeemInvite_NotFound(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrInviteNotFound
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/missing/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRedeemInvite_AlreadyParticipant(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrAlreadyParticipant
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/valid-token/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRedeemInvite_Expired(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrInviteExpired
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/expired-token/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusGone, w.Code)
}

func TestRedeemInvite_NotEnoughSquares(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrNotEnoughSquares
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/full-token/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ====================
// GetInvites
// ====================

func TestGetInvites_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockInviteService()
	svc.getInvitesByContestIDFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestInvite, error) {
		return []model.ContestInvite{{ID: uuid.New(), ContestID: contestID, Token: "tok1"}}, nil
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", contestID), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []model.ContestInvite
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
}

func TestGetInvites_InvalidContestID(t *testing.T) {
	svc := defaultMockInviteService()
	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, "/contests/bad/invites", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetInvites_Forbidden(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitesByContestIDFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestInvite, error) {
		return nil, errs.ErrInsufficientRole
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// DeleteInvite
// ====================

func TestDeleteInvite_Success(t *testing.T) {
	svc := defaultMockInviteService()
	svc.deleteInviteFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return nil
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteInvite_InvalidContestID(t *testing.T) {
	svc := defaultMockInviteService()
	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/bad/invites/%s", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteInvite_InvalidInviteID(t *testing.T) {
	svc := defaultMockInviteService()
	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/bad", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteInvite_Forbidden(t *testing.T) {
	svc := defaultMockInviteService()
	svc.deleteInviteFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return errs.ErrInsufficientRole
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// Additional error-branch coverage
// ====================

func TestCreateInvite_InternalError(t *testing.T) {
	svc := defaultMockInviteService()
	svc.createInviteFn = func(_ context.Context, _ uuid.UUID, _ *model.CreateInviteRequest, _ string) (*model.ContestInvite, error) {
		return nil, assert.AnError
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/invites", h.CreateInvite)

	body, _ := json.Marshal(model.CreateInviteRequest{MaxSquares: 10, Role: "participant"})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/invites", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetInvitePreview_InternalError(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitePreviewFn = func(_ context.Context, _ string) (*model.InvitePreviewResponse, error) {
		return nil, assert.AnError
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.GET("/invites/:token", h.GetInvitePreview)

	req, _ := http.NewRequest(http.MethodGet, "/invites/some-token", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRedeemInvite_MaxUsesReached(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrInviteMaxUsesReached
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/used-up/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusGone, w.Code)
}

func TestRedeemInvite_InternalError(t *testing.T) {
	svc := defaultMockInviteService()
	svc.redeemInviteFn = func(_ context.Context, _ string, _ string) (*model.ContestParticipant, error) {
		return nil, assert.AnError
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.POST("/invites/:token/redeem", h.RedeemInvite)

	req, _ := http.NewRequest(http.MethodPost, "/invites/token/redeem", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetInvites_NotFound(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitesByContestIDFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestInvite, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetInvites_InternalError(t *testing.T) {
	svc := defaultMockInviteService()
	svc.getInvitesByContestIDFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestInvite, error) {
		return nil, assert.AnError
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/invites", h.GetInvites)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/invites", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteInvite_NotFound(t *testing.T) {
	svc := defaultMockInviteService()
	svc.deleteInviteFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return gorm.ErrRecordNotFound
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteInvite_InternalError(t *testing.T) {
	svc := defaultMockInviteService()
	svc.deleteInviteFn = func(_ context.Context, _, _ uuid.UUID, _ string) error {
		return assert.AnError
	}

	h := NewInviteHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/invites/:inviteId", h.DeleteInvite)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/invites/%s", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
