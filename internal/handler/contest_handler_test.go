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
	"github.com/maxmorhardt/squares-api/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ====================
// GetContestsByOwner
// ====================

func TestGetContestsByOwner_Success(t *testing.T) {
	svc := mocks.NewContestService(t)
	svc.EXPECT().GetContestsByOwnerPaginated(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]model.Contest{{ID: uuid.New(), Name: "C1"}}, int64(1), nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.PaginatedContestResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp.Contests, 1)
	assert.Equal(t, int64(1), resp.Total)
}

func authenticatedMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := &model.Claims{Email: userID, EmailVerified: true, Name: "Test", Expire: 9999999999}
		util.SetGinContextValue(c, model.UserKey, userID)
		util.SetGinContextValue(c, model.ClaimsKey, claims)
		c.Next()
	}
}

func TestGetContestsByOwner_MissingPage(t *testing.T)  { getContestsBadRequest(t, "limit=10") }
func TestGetContestsByOwner_MissingLimit(t *testing.T) { getContestsBadRequest(t, "page=1") }
func TestGetContestsByOwner_InvalidLimit(t *testing.T) { getContestsBadRequest(t, "page=1&limit=50") }
func TestGetContestsByOwner_InvalidPageFormat(t *testing.T) {
	getContestsBadRequest(t, "page=abc&limit=10")
}
func TestGetContestsByOwner_ZeroPage(t *testing.T) { getContestsBadRequest(t, "page=0&limit=10") }
func TestGetContestsByOwner_InvalidLimitFormat(t *testing.T) {
	getContestsBadRequest(t, "page=1&limit=abc")
}

func getContestsBadRequest(t *testing.T, query string) {
	t.Helper()
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?"+query, http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetContestsByOwner_ServiceError(t *testing.T) {
	svc := mocks.NewContestService(t)
	svc.EXPECT().GetContestsByOwnerPaginated(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, int64(0), assert.AnError)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetContestsByOwner_PassesSearchQuery(t *testing.T) {
	svc := mocks.NewContestService(t)
	svc.EXPECT().GetContestsByOwnerPaginated(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "foo").
		Return([]model.Contest{}, int64(0), nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10&search=foo", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetContestsByOwner_TrimsSearchQuery(t *testing.T) {
	svc := mocks.NewContestService(t)
	svc.EXPECT().GetContestsByOwnerPaginated(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "bar").
		Return([]model.Contest{}, int64(0), nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.GET("/contests/owner/:owner", h.GetContestsByOwner)

	req, _ := http.NewRequest(http.MethodGet, "/contests/owner/owner1?page=1&limit=10&search=%20%20bar%20%20", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ====================
// CreateContest
// ====================

func TestCreateContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().CreateContest(mock.Anything, mock.Anything, mock.Anything).
		Return(&model.Contest{ID: contestID, Name: "NewContest", Owner: "owner1", Status: model.ContestStatusActive}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	w := doRequest(r, jsonReq(http.MethodPut, "/contests", model.CreateContestRequest{Owner: "owner1", Name: "NewContest", MaxSquares: 10}))
	assert.Equal(t, http.StatusOK, w.Code)
	var resp model.Contest
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, contestID, resp.ID)
}

func jsonReq(method, target string, v any) *http.Request {
	body, _ := json.Marshal(v)
	req, _ := http.NewRequest(method, target, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestCreateContest_InvalidBody(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateContest_OwnerMismatch(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("user1"))
	r.PUT("/contests", h.CreateContest)

	w := doRequest(r, jsonReq(http.MethodPut, "/contests", model.CreateContestRequest{Owner: "someone-else", Name: "Test", MaxSquares: 10}))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreateContest_AlreadyExists(t *testing.T) {
	createContestErr(t, errs.ErrContestAlreadyExists, http.StatusBadRequest)
}
func TestCreateContest_DatabaseUnavailable(t *testing.T) {
	createContestErr(t, errs.ErrDatabaseUnavailable, http.StatusInternalServerError)
}
func TestCreateContest_InternalError(t *testing.T) {
	createContestErr(t, assert.AnError, http.StatusInternalServerError)
}

func createContestErr(t *testing.T, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().CreateContest(mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PUT("/contests", h.CreateContest)

	w := doRequest(r, jsonReq(http.MethodPut, "/contests", model.CreateContestRequest{Owner: "owner1", Name: "Test", MaxSquares: 10}))
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// UpdateContest
// ====================

func TestUpdateContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().UpdateContest(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.Contest{ID: contestID, Name: "Updated", Status: model.ContestStatusActive}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	home := "Eagles"
	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s", contestID), model.UpdateContestRequest{HomeTeam: &home}))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateContest_InvalidID(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	req, _ := http.NewRequest(http.MethodPatch, "/contests/not-a-uuid", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateContest_InvalidBody(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id", h.UpdateContest)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateContest_NotFound(t *testing.T) {
	updateContestErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestUpdateContest_Finalized(t *testing.T) {
	updateContestErr(t, "owner1", errs.ErrContestFinalized, http.StatusForbidden)
}
func TestUpdateContest_Forbidden(t *testing.T) {
	updateContestErr(t, "stranger", errs.ErrUnauthorizedContestEdit, http.StatusForbidden)
}
func TestUpdateContest_DatabaseUnavailable(t *testing.T) {
	updateContestErr(t, "owner1", errs.ErrDatabaseUnavailable, http.StatusInternalServerError)
}
func TestUpdateContest_OtherError(t *testing.T) {
	updateContestErr(t, "owner1", errs.ErrContestNotEditable, http.StatusBadRequest)
}

func updateContestErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().UpdateContest(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.PATCH("/contests/:id", h.UpdateContest)

	home := "Eagles"
	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s", uuid.New()), model.UpdateContestRequest{HomeTeam: &home}))
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// DeleteContest
// ====================

func TestDeleteContest_Success(t *testing.T) {
	svc := mocks.NewContestService(t)
	svc.EXPECT().DeleteContest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteContest_InvalidID(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, "/contests/bad-id", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteContest_NotFound(t *testing.T) {
	deleteContestErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestDeleteContest_Finalized(t *testing.T) {
	deleteContestErr(t, "owner1", errs.ErrContestFinalized, http.StatusForbidden)
}
func TestDeleteContest_Forbidden(t *testing.T) {
	deleteContestErr(t, "stranger", errs.ErrUnauthorizedContestDelete, http.StatusForbidden)
}
func TestDeleteContest_InternalError(t *testing.T) {
	deleteContestErr(t, "owner1", assert.AnError, http.StatusInternalServerError)
}

func deleteContestErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().DeleteContest(mock.Anything, mock.Anything, mock.Anything).Return(svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.DELETE("/contests/:id", h.DeleteContest)

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// StartContest
// ====================

func TestStartContest_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().StartContest(mock.Anything, mock.Anything, mock.Anything).
		Return(&model.Contest{ID: contestID, Status: model.ContestStatusQ1}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", contestID), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestStartContest_InvalidID(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, "/contests/not-valid/start", http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStartContest_NotFound(t *testing.T) {
	startContestErr(t, gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestStartContest_OtherError(t *testing.T) {
	startContestErr(t, errs.ErrContestNotEditable, http.StatusBadRequest)
}

func startContestErr(t *testing.T, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().StartContest(mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/start", h.StartContest)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// RecordQuarterResult
// ====================

func TestRecordQuarterResult_Success(t *testing.T) {
	contestID := uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().RecordQuarterResult(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.QuarterResult{ContestID: contestID, Quarter: 1, HomeTeamScore: 7, AwayTeamScore: 3}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	w := doRequest(r, jsonReq(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", contestID), model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3}))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecordQuarterResult_InvalidID(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	w := doRequest(r, jsonReq(http.MethodPost, "/contests/bad/quarter-result", model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_InvalidBody(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRecordQuarterResult_AlreadyExists(t *testing.T) {
	recordQuarterErr(t, errs.ErrQuarterResultAlreadyExists, http.StatusBadRequest)
}
func TestRecordQuarterResult_NotFound(t *testing.T) {
	recordQuarterErr(t, gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestRecordQuarterResult_InvalidData(t *testing.T) {
	recordQuarterErr(t, gorm.ErrInvalidData, http.StatusBadRequest)
}
func TestRecordQuarterResult_InternalError(t *testing.T) {
	recordQuarterErr(t, assert.AnError, http.StatusInternalServerError)
}

func recordQuarterErr(t *testing.T, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().RecordQuarterResult(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/quarter-result", h.RecordQuarterResult)

	w := doRequest(r, jsonReq(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", uuid.New()), model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3}))
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// UpdateSquare
// ====================

func TestUpdateSquare_Success(t *testing.T) {
	contestID, squareID := uuid.New(), uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().UpdateSquare(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.Square{ID: squareID, ContestID: contestID, Value: "ABC", Owner: "owner1"}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", contestID, squareID), model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"}))
	assert.Equal(t, http.StatusOK, w.Code)
}

func updateSquareBadID(t *testing.T, target string) {
	t.Helper()
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	w := doRequest(r, jsonReq(http.MethodPatch, target, model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"}))
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSquare_InvalidContestID(t *testing.T) {
	updateSquareBadID(t, fmt.Sprintf("/contests/bad/squares/%s", uuid.New()))
}
func TestUpdateSquare_InvalidSquareID(t *testing.T) {
	updateSquareBadID(t, fmt.Sprintf("/contests/%s/squares/bad", uuid.New()))
}

func TestUpdateSquare_InvalidBody(t *testing.T) {
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), bytes.NewReader([]byte(`{bad`)))
	req.Header.Set("Content-Type", "application/json")
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSquare_NotFound(t *testing.T) {
	updateSquareErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestUpdateSquare_SquareNotEditable(t *testing.T) {
	updateSquareErr(t, "owner1", errs.ErrSquareNotEditable, http.StatusForbidden)
}
func TestUpdateSquare_UnauthorizedSquareEdit(t *testing.T) {
	updateSquareErr(t, "stranger", errs.ErrUnauthorizedSquareEdit, http.StatusForbidden)
}
func TestUpdateSquare_ClaimsNotFound(t *testing.T) {
	updateSquareErr(t, "owner1", errs.ErrClaimsNotFound, http.StatusUnauthorized)
}
func TestUpdateSquare_DatabaseUnavailable(t *testing.T) {
	updateSquareErr(t, "owner1", errs.ErrDatabaseUnavailable, http.StatusInternalServerError)
}
func TestUpdateSquare_OtherError(t *testing.T) {
	updateSquareErr(t, "owner1", errs.ErrInvalidSquareValue, http.StatusBadRequest)
}

func updateSquareErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().UpdateSquare(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.PATCH("/contests/:id/squares/:squareId", h.UpdateSquare)

	w := doRequest(r, jsonReq(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", uuid.New(), uuid.New()), model.UpdateSquareRequest{Value: "ABC", Owner: "owner1"}))
	assert.Equal(t, wantCode, w.Code)
}

// ====================
// ClearSquare
// ====================

func TestClearSquare_Success(t *testing.T) {
	contestID, squareID := uuid.New(), uuid.New()
	svc := mocks.NewContestService(t)
	svc.EXPECT().ClearSquare(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&model.Square{ID: squareID, ContestID: contestID}, nil)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", contestID, squareID), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClearSquare_InvalidContestID(t *testing.T) {
	clearSquareBadID(t, fmt.Sprintf("/contests/bad/squares/%s/clear", uuid.New()))
}
func TestClearSquare_InvalidSquareID(t *testing.T) {
	clearSquareBadID(t, fmt.Sprintf("/contests/%s/squares/bad/clear", uuid.New()))
}

func clearSquareBadID(t *testing.T, target string) {
	t.Helper()
	h := NewContestHandler(mocks.NewContestService(t))
	r := gin.New()
	r.Use(authenticatedMiddleware("owner1"))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, target, http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestClearSquare_NotFound(t *testing.T) {
	clearSquareErr(t, "owner1", gorm.ErrRecordNotFound, http.StatusNotFound)
}
func TestClearSquare_Forbidden(t *testing.T) {
	clearSquareErr(t, "stranger", errs.ErrUnauthorizedSquareEdit, http.StatusForbidden)
}
func TestClearSquare_SquareNotEditable(t *testing.T) {
	clearSquareErr(t, "owner1", errs.ErrSquareNotEditable, http.StatusForbidden)
}
func TestClearSquare_DatabaseUnavailable(t *testing.T) {
	clearSquareErr(t, "owner1", errs.ErrDatabaseUnavailable, http.StatusInternalServerError)
}
func TestClearSquare_OtherError(t *testing.T) {
	clearSquareErr(t, "owner1", errs.ErrInvalidSquareValue, http.StatusBadRequest)
}

func clearSquareErr(t *testing.T, user string, svcErr error, wantCode int) {
	t.Helper()
	svc := mocks.NewContestService(t)
	svc.EXPECT().ClearSquare(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, svcErr)
	h := NewContestHandler(svc)

	r := gin.New()
	r.Use(authenticatedMiddleware(user))
	r.POST("/contests/:id/squares/:squareId/clear", h.ClearSquare)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/squares/%s/clear", uuid.New(), uuid.New()), http.NoBody)
	w := doRequest(r, req)
	assert.Equal(t, wantCode, w.Code)
}
