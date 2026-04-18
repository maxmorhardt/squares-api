package handler

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("safestring", func(fl validator.FieldLevel) bool {
			s := fl.Field().String()
			return s == "" || util.IsSafeString(s)
		})
	}
}

func newTestRouter() *gin.Engine {
	return gin.New()
}

func authenticatedMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := &model.Claims{Username: userID, Name: "Test", Expire: 9999999999}
		util.SetGinContextValue(c, model.UserKey, userID)
		util.SetGinContextValue(c, model.ClaimsKey, claims)
		c.Next()
	}
}

func doRequest(r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// =====================
// Mock Services
// =====================

// mockContactService implements service.ContactService
type mockContactService struct {
	submitContactFn func(ctx context.Context, req *model.ContactRequest, ipAddress string) error
}

func (m *mockContactService) SubmitContact(ctx context.Context, req *model.ContactRequest, ipAddress string) error {
	return m.submitContactFn(ctx, req, ipAddress)
}

// mockContestService implements service.ContestService
type mockContestService struct {
	getContestByOwnerAndNameFn    func(ctx context.Context, owner, name string) (*model.Contest, error)
	getContestsByOwnerPaginatedFn func(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error)
	createContestFn               func(ctx context.Context, req *model.CreateContestRequest, user string) (*model.Contest, error)
	updateContestFn               func(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error)
	startContestFn                func(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error)
	recordQuarterResultFn         func(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error)
	deleteContestFn               func(ctx context.Context, contestID uuid.UUID, user string) error
	updateSquareFn                func(ctx context.Context, contestID, squareID uuid.UUID, req *model.UpdateSquareRequest, user string) (*model.Square, error)
	clearSquareFn                 func(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error)
}

func (m *mockContestService) GetContestByOwnerAndName(ctx context.Context, owner, name string) (*model.Contest, error) {
	return m.getContestByOwnerAndNameFn(ctx, owner, name)
}
func (m *mockContestService) GetContestsByOwnerPaginated(ctx context.Context, owner string, page, limit int) ([]model.Contest, int64, error) {
	return m.getContestsByOwnerPaginatedFn(ctx, owner, page, limit)
}
func (m *mockContestService) CreateContest(ctx context.Context, req *model.CreateContestRequest, user string) (*model.Contest, error) {
	return m.createContestFn(ctx, req, user)
}
func (m *mockContestService) UpdateContest(ctx context.Context, contestID uuid.UUID, req *model.UpdateContestRequest, user string) (*model.Contest, error) {
	return m.updateContestFn(ctx, contestID, req, user)
}
func (m *mockContestService) StartContest(ctx context.Context, contestID uuid.UUID, user string) (*model.Contest, error) {
	return m.startContestFn(ctx, contestID, user)
}
func (m *mockContestService) RecordQuarterResult(ctx context.Context, contestID uuid.UUID, homeScore, awayScore int, user string) (*model.QuarterResult, error) {
	return m.recordQuarterResultFn(ctx, contestID, homeScore, awayScore, user)
}
func (m *mockContestService) DeleteContest(ctx context.Context, contestID uuid.UUID, user string) error {
	return m.deleteContestFn(ctx, contestID, user)
}
func (m *mockContestService) UpdateSquare(ctx context.Context, contestID, squareID uuid.UUID, req *model.UpdateSquareRequest, user string) (*model.Square, error) {
	return m.updateSquareFn(ctx, contestID, squareID, req, user)
}
func (m *mockContestService) ClearSquare(ctx context.Context, contestID, squareID uuid.UUID, user string) (*model.Square, error) {
	return m.clearSquareFn(ctx, contestID, squareID, user)
}

// mockAuthService implements service.AuthService
type mockAuthService struct {
	isDeclaredUserFn func(ctx context.Context, user string) bool
	hasGroupFn       func(ctx context.Context, role string) bool
}

func (m *mockAuthService) IsDeclaredUser(ctx context.Context, user string) bool {
	return m.isDeclaredUserFn(ctx, user)
}
func (m *mockAuthService) HasGroup(ctx context.Context, role string) bool {
	return m.hasGroupFn(ctx, role)
}

// mockParticipantService implements service.ParticipantService
type mockParticipantService struct {
	getParticipantsFn   func(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestParticipant, error)
	getMyContestsFn     func(ctx context.Context, user string) ([]model.Contest, error)
	updateParticipantFn func(ctx context.Context, contestID uuid.UUID, targetUserID string, req *model.UpdateParticipantRequest, user string) (*model.ContestParticipant, error)
	removeParticipantFn func(ctx context.Context, contestID uuid.UUID, targetUserID, user string) error
	authorizeFn         func(ctx context.Context, contestID uuid.UUID, userID string, act service.Action) error
}

func (m *mockParticipantService) GetParticipants(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestParticipant, error) {
	return m.getParticipantsFn(ctx, contestID, user)
}
func (m *mockParticipantService) GetMyContests(ctx context.Context, user string) ([]model.Contest, error) {
	return m.getMyContestsFn(ctx, user)
}
func (m *mockParticipantService) UpdateParticipant(ctx context.Context, contestID uuid.UUID, targetUserID string, req *model.UpdateParticipantRequest, user string) (*model.ContestParticipant, error) {
	return m.updateParticipantFn(ctx, contestID, targetUserID, req, user)
}
func (m *mockParticipantService) RemoveParticipant(ctx context.Context, contestID uuid.UUID, targetUserID, user string) error {
	return m.removeParticipantFn(ctx, contestID, targetUserID, user)
}
func (m *mockParticipantService) Authorize(ctx context.Context, contestID uuid.UUID, userID string, act service.Action) error {
	return m.authorizeFn(ctx, contestID, userID, act)
}

// mockInviteService implements service.InviteService
type mockInviteService struct {
	createInviteFn          func(ctx context.Context, contestID uuid.UUID, req *model.CreateInviteRequest, user string) (*model.ContestInvite, error)
	getInvitePreviewFn      func(ctx context.Context, token string) (*model.InvitePreviewResponse, error)
	redeemInviteFn          func(ctx context.Context, token, user string) (*model.ContestParticipant, error)
	getInvitesByContestIDFn func(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestInvite, error)
	deleteInviteFn          func(ctx context.Context, contestID, inviteID uuid.UUID, user string) error
}

func (m *mockInviteService) CreateInvite(ctx context.Context, contestID uuid.UUID, req *model.CreateInviteRequest, user string) (*model.ContestInvite, error) {
	return m.createInviteFn(ctx, contestID, req, user)
}
func (m *mockInviteService) GetInvitePreview(ctx context.Context, token string) (*model.InvitePreviewResponse, error) {
	return m.getInvitePreviewFn(ctx, token)
}
func (m *mockInviteService) RedeemInvite(ctx context.Context, token, user string) (*model.ContestParticipant, error) {
	return m.redeemInviteFn(ctx, token, user)
}
func (m *mockInviteService) GetInvitesByContestID(ctx context.Context, contestID uuid.UUID, user string) ([]model.ContestInvite, error) {
	return m.getInvitesByContestIDFn(ctx, contestID, user)
}
func (m *mockInviteService) DeleteInvite(ctx context.Context, contestID, inviteID uuid.UUID, user string) error {
	return m.deleteInviteFn(ctx, contestID, inviteID, user)
}

// mockStatsService implements service.StatsService
type mockStatsService struct {
	getStatsFn func(ctx context.Context) (*model.StatsResponse, error)
}

func (m *mockStatsService) GetStats(ctx context.Context) (*model.StatsResponse, error) {
	return m.getStatsFn(ctx)
}
