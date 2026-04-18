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

func defaultMockContestService() *mockContestService {
	return &mockContestService{}
}

func defaultMockAuthService() *mockAuthService {
	return &mockAuthService{
		isDeclaredUserFn: func(_ context.Context, _ string) bool { return true },
		hasGroupFn:       func(_ context.Context, _ string) bool { return false },
	}
}

// ====================
// GetContestByOwnerAndName
// ====================

func TestGetContestByOwnerAndName_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockContestService()
	svc.getContestByOwnerAndNameFn = func(_ context.Context, owner, name string) (*model.Contest, error) {
		return &model.Contest{ID: contestID, Owner: owner, Name: name, Status: model.ContestStatusActive}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner/name/:name", h.GetContestByOwnerAndName)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1/name/TestContest", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.Contest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, contestID, resp.ID)
}

func TestGetContestByOwnerAndName_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestByOwnerAndNameFn = func(_ context.Context, _, _ string) (*model.Contest, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/owner/:owner/name/:name", h.GetContestByOwnerAndName)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1/name/Missing", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetContestByOwnerAndName_Forbidden(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestByOwnerAndNameFn = func(_ context.Context, _, _ string) (*model.Contest, error) {
		return nil, errs.ErrNotParticipant
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.GET("/contests/owner/:owner/name/:name", h.GetContestByOwnerAndName)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1/name/Private", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// GetContestsByOwner
// ====================

func TestGetContestsByOwner_Success(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestsByOwnerPaginatedFn = func(_ context.Context, _ string, _, _ int) ([]model.Contest, int64, error) {
		return []model.Contest{{ID: uuid.New(), Name: "C1"}}, 1, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.PaginatedContestResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 1, len(resp.Contests))
	assert.Equal(t, int64(1), resp.Total)
}

func TestGetContestsByOwner_MissingPage(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?limit=10", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_InvalidLimit(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=50", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_ServiceError(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestsByOwnerPaginatedFn = func(_ context.Context, _ string, _, _ int) ([]model.Contest, int64, error) {
		return nil, 0, assert.AnError
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ====================
// CreateContest
// ====================

func TestCreateContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockContestService()
	svc.createContestFn = func(_ context.Context, req *model.CreateContestRequest, _ string) (*model.Contest, error) {
		return &model.Contest{ID: contestID, Name: req.Name, Owner: req.Owner, Status: model.ContestStatusActive}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	body, _ := json.Marshal(model.CreateContestRequest{Owner: "owner1", Name: "NewContest"})
	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.Contest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, contestID, resp.ID)
}

func TestCreateContest_InvalidBody(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContest_OwnerMismatch(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.PUT("/contests", h.CreateContest)

	body, _ := json.Marshal(model.CreateContestRequest{Owner: "someone-else", Name: "Test"})
	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContest_AlreadyExists(t *testing.T) {
	svc := defaultMockContestService()
	svc.createContestFn = func(_ context.Context, _ *model.CreateContestRequest, _ string) (*model.Contest, error) {
		return nil, errs.ErrContestAlreadyExists
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	body, _ := json.Marshal(model.CreateContestRequest{Owner: "owner1", Name: "Dup"})
	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ====================
// UpdateContest
// ====================

func TestUpdateContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockContestService()
	svc.updateContestFn = func(_ context.Context, id uuid.UUID, _ *model.UpdateContestRequest, _ string) (*model.Contest, error) {
		return &model.Contest{ID: id, Name: "Updated", Status: model.ContestStatusActive}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	homeTeam := "Eagles"
	body, _ := json.Marshal(model.UpdateContestRequest{HomeTeam: &homeTeam})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", contestID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateContest_InvalidID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	req, _ := http.NewRequest(http.MethodPatch, "/contests/not-a-uuid", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateContest_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateContestFn = func(_ context.Context, _ uuid.UUID, _ *model.UpdateContestRequest, _ string) (*model.Contest, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	homeTeam := "Eagles"
	body, _ := json.Marshal(model.UpdateContestRequest{HomeTeam: &homeTeam})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateContest_Forbidden(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateContestFn = func(_ context.Context, _ uuid.UUID, _ *model.UpdateContestRequest, _ string) (*model.Contest, error) {
		return nil, errs.ErrUnauthorizedContestEdit
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.PATCH("/contests/:id", h.UpdateContest)

	homeTeam := "Eagles"
	body, _ := json.Marshal(model.UpdateContestRequest{HomeTeam: &homeTeam})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// DeleteContest
// ====================

func TestDeleteContest_Success(t *testing.T) {
	svc := defaultMockContestService()
	svc.deleteContestFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteContest_InvalidID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, "/contests/bad-id", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteContest_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.deleteContestFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteContest_Forbidden(t *testing.T) {
	svc := defaultMockContestService()
	svc.deleteContestFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return errs.ErrUnauthorizedContestDelete
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// StartContest
// ====================

func TestStartContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockContestService()
	svc.startContestFn = func(_ context.Context, id uuid.UUID, _ string) (*model.Contest, error) {
		return &model.Contest{ID: id, Status: model.ContestStatusQ1}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", contestID), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStartContest_InvalidID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, "/contests/not-valid/start", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartContest_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.startContestFn = func(_ context.Context, _ uuid.UUID, _ string) (*model.Contest, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ====================
// RecordQuarterResult
// ====================

func TestRecordQuarterResult_Success(t *testing.T) {
	contestID := uuid.New()
	svc := defaultMockContestService()
	svc.recordQuarterResultFn = func(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*model.QuarterResult, error) {
		return &model.QuarterResult{ContestID: contestID, Quarter: 1, HomeTeamScore: 7, AwayTeamScore: 3}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", contestID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecordQuarterResult_InvalidID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, "/contests/bad/quarter-result", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_InvalidBody(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_AlreadyExists(t *testing.T) {
	svc := defaultMockContestService()
	svc.recordQuarterResultFn = func(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*model.QuarterResult, error) {
		return nil, errs.ErrQuarterResultAlreadyExists
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ====================
// UpdateSquare
// ====================

func TestUpdateSquare_Success(t *testing.T) {
	contestID := uuid.New()
	squareID := uuid.New()
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, sID uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return &model.Square{ID: sID, ContestID: contestID, Value: "ABC", Owner: "owner1"}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", contestID, squareID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateSquare_InvalidContestID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/bad/squares/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSquare_InvalidSquareID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/bad", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSquare_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ====================
// ClearSquare
// ====================

func TestClearSquare_Success(t *testing.T) {
	contestID := uuid.New()
	squareID := uuid.New()
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, sID uuid.UUID, _ string) (*model.Square, error) {
		return &model.Square{ID: sID, ContestID: contestID, Value: "", Owner: ""}, nil
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", contestID, squareID), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClearSquare_InvalidContestID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/bad/squares/%s/clear", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClearSquare_InvalidSquareID(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/bad/clear", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClearSquare_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Square, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestClearSquare_Forbidden(t *testing.T) {
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Square, error) {
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ====================
// Additional error-branch coverage
// ====================

func TestGetContestByOwnerAndName_InsufficientRole(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestByOwnerAndNameFn = func(_ context.Context, _, _ string) (*model.Contest, error) {
		return nil, errs.ErrInsufficientRole
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.GET("/contests/owner/:owner/name/:name", h.GetContestByOwnerAndName)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1/name/Private", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestGetContestByOwnerAndName_InternalError(t *testing.T) {
	svc := defaultMockContestService()
	svc.getContestByOwnerAndNameFn = func(_ context.Context, _, _ string) (*model.Contest, error) {
		return nil, assert.AnError
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("user1"))
	r.GET("/contests/owner/:owner/name/:name", h.GetContestByOwnerAndName)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1/name/Test", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetContestsByOwner_MissingLimit(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_InvalidPageFormat(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=abc&limit=10", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_ZeroPage(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=0&limit=10", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_InvalidLimitFormat(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=abc", nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContest_DatabaseUnavailable(t *testing.T) {
	svc := defaultMockContestService()
	svc.createContestFn = func(_ context.Context, _ *model.CreateContestRequest, _ string) (*model.Contest, error) {
		return nil, errs.ErrDatabaseUnavailable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	body, _ := json.Marshal(model.CreateContestRequest{Owner: "owner1", Name: "Test"})
	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateContest_InternalError(t *testing.T) {
	svc := defaultMockContestService()
	svc.createContestFn = func(_ context.Context, _ *model.CreateContestRequest, _ string) (*model.Contest, error) {
		return nil, assert.AnError
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	body, _ := json.Marshal(model.CreateContestRequest{Owner: "owner1", Name: "Test"})
	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateContest_InvalidBody(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateContest_DatabaseUnavailable(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateContestFn = func(_ context.Context, _ uuid.UUID, _ *model.UpdateContestRequest, _ string) (*model.Contest, error) {
		return nil, errs.ErrDatabaseUnavailable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	homeTeam := "Eagles"
	body, _ := json.Marshal(model.UpdateContestRequest{HomeTeam: &homeTeam})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateContest_OtherError(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateContestFn = func(_ context.Context, _ uuid.UUID, _ *model.UpdateContestRequest, _ string) (*model.Contest, error) {
		return nil, errs.ErrContestNotEditable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	homeTeam := "Eagles"
	body, _ := json.Marshal(model.UpdateContestRequest{HomeTeam: &homeTeam})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteContest_InternalError(t *testing.T) {
	svc := defaultMockContestService()
	svc.deleteContestFn = func(_ context.Context, _ uuid.UUID, _ string) error {
		return assert.AnError
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestStartContest_OtherError(t *testing.T) {
	svc := defaultMockContestService()
	svc.startContestFn = func(_ context.Context, _ uuid.UUID, _ string) (*model.Contest, error) {
		return nil, errs.ErrContestNotEditable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_NotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.recordQuarterResultFn = func(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*model.QuarterResult, error) {
		return nil, gorm.ErrRecordNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRecordQuarterResult_InvalidData(t *testing.T) {
	svc := defaultMockContestService()
	svc.recordQuarterResultFn = func(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*model.QuarterResult, error) {
		return nil, gorm.ErrInvalidData
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_InternalError(t *testing.T) {
	svc := defaultMockContestService()
	svc.recordQuarterResultFn = func(_ context.Context, _ uuid.UUID, _, _ int, _ string) (*model.QuarterResult, error) {
		return nil, assert.AnError
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	body, _ := json.Marshal(model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateSquare_InvalidBody(t *testing.T) {
	svc := defaultMockContestService()
	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSquare_SquareNotEditable(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, errs.ErrSquareNotEditable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateSquare_UnauthorizedSquareEdit(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, errs.ErrUnauthorizedSquareEdit
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("stranger"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUpdateSquare_ClaimsNotFound(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, errs.ErrClaimsNotFound
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdateSquare_DatabaseUnavailable(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, errs.ErrDatabaseUnavailable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateSquare_OtherError(t *testing.T) {
	svc := defaultMockContestService()
	svc.updateSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ *model.UpdateSquareRequest, _ string) (*model.Square, error) {
		return nil, errs.ErrInvalidSquareValue
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	body, _ := json.Marshal(model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"})
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClearSquare_SquareNotEditable(t *testing.T) {
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Square, error) {
		return nil, errs.ErrSquareNotEditable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestClearSquare_DatabaseUnavailable(t *testing.T) {
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Square, error) {
		return nil, errs.ErrDatabaseUnavailable
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestClearSquare_OtherError(t *testing.T) {
	svc := defaultMockContestService()
	svc.clearSquareFn = func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*model.Square, error) {
		return nil, errs.ErrInvalidSquareValue
	}

	h := NewContestHandler(svc, defaultMockAuthService())
	r := newTestRouter()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), nil)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
