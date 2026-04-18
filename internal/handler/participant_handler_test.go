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

func defaultMockParticipantService() *mockParticipantService {
	return &mockParticipantService{}
}

// ====================
// GetParticipants
// ====================

func TestGetParticipants_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockParticipantService()
	svc.getParticipantsFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestParticipant, error) {
		return []model.ContestParticipant{
			{ID: uuid.New(), ContestID: contestID, UserID: "owner1", Role: model.ParticipantRoleOwner},
			{ID: uuid.New(), ContestID: contestID, UserID: "user1", Role: model.ParticipantRoleParticipant},
		}, nil
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
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
	svc := defaultMockParticipantService()
	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, "/contests/bad-id/participants", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetParticipants_NotFound(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.getParticipantsFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestParticipant, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetParticipants_Forbidden(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.getParticipantsFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestParticipant, error) {
		return nil, errs.ErrNotParticipant
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// GetMyContests
// ====================

func TestGetMyContests_Success(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.getMyContestsFn = func(_ context.Context, _ string) ([]model.Contest, error) {
		return []model.Contest{{ID: uuid.New(), Name: "MyContest"}}, nil
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
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
	svc := defaultMockParticipantService()
	svc.getMyContestsFn = func(_ context.Context, _ string) ([]model.Contest, error) {
		return nil, assert.AnError
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/me", h.GetMyContests)

	req, _ := http.NewRequest(http.MethodGet, "/contests/me", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ====================
// UpdateParticipant
// ====================

func TestUpdateParticipant_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, targetUserID string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return &model.ContestParticipant{ID: uuid.New(), ContestID: contestID, UserID: targetUserID, Role: model.ParticipantRoleViewer}, nil
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	body, _ := json.Marshal(model.UpdateParticipantRequest{Role: &role})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", contestID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateParticipant_InvalidContestID(t *testing.T) {
	svc := defaultMockParticipantService()
	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	req, _ := http.NewRequest(http.MethodPatch, "/contests/bad/participants/user1", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateParticipant_InvalidBody(t *testing.T) {
	svc := defaultMockParticipantService()
	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateParticipant_Forbidden(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrInsufficientRole
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	body, _ := json.Marshal(model.UpdateParticipantRequest{Role: &role})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateParticipant_CannotChangeOwner(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrCannotChangeOwner
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	body, _ := json.Marshal(model.UpdateParticipantRequest{Role: &role})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/owner1", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateParticipant_SquareLimitTooLow(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrSquareLimitTooLow
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	maxSquares := 0
	body, _ := json.Marshal(model.UpdateParticipantRequest{MaxSquares: &maxSquares})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ====================
// RemoveParticipant
// ====================

func TestRemoveParticipant_Success(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.removeParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
		return nil
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRemoveParticipant_InvalidID(t *testing.T) {
	svc := defaultMockParticipantService()
	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, "/contests/bad/participants/user1", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRemoveParticipant_Forbidden(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.removeParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
		return errs.ErrInsufficientRole
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRemoveParticipant_CannotRemoveOwner(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.removeParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
		return errs.ErrCannotRemoveOwner
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/owner1", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRemoveParticipant_NotFound(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.removeParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
		return errs.ErrNotParticipant
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/nobody", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ====================
// Additional error-branch coverage
// ====================

func TestGetParticipants_InsufficientRole(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.getParticipantsFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestParticipant, error) {
		return nil, errs.ErrInsufficientRole
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("viewer"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetParticipants_InternalError(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.getParticipantsFn = func(_ context.Context, _ uuid.UUID, _ string) ([]model.ContestParticipant, error) {
		return nil, assert.AnError
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/:id/participants", h.GetParticipants)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/%s/participants", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateParticipant_NotParticipant(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return nil, errs.ErrNotParticipant
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	body, _ := json.Marshal(model.UpdateParticipantRequest{Role: &role})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/unknown", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateParticipant_InternalError(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.updateParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ *model.UpdateParticipantRequest, _ string) (*model.ContestParticipant, error) {
		return nil, assert.AnError
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/participants/:userId", h.UpdateParticipant)

	role := "viewer"
	body, _ := json.Marshal(model.UpdateParticipantRequest{Role: &role})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRemoveParticipant_InternalError(t *testing.T) {
	svc := defaultMockParticipantService()
	svc.removeParticipantFn = func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
		return assert.AnError
	}

	h := NewParticipantHandler(svc)
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id/participants/:userId", h.RemoveParticipant)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s/participants/user1", uuid.New()), http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
