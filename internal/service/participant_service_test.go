package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAuthorize_PublicViewSkipsParticipantLookup(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibilityPublic, nil)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	assert.NoError(t, svc.Authorize(context.Background(), uuid.New(), "anyone", service.ActionView))
}

func TestAuthorize_VisibilityNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibility(""), gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	assert.ErrorIs(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionView), gorm.ErrRecordNotFound)
}

func TestAuthorize_VisibilityDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibility(""), errors.New("boom"))

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	assert.ErrorIs(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionView), errs.ErrDatabaseUnavailable)
}

func TestAuthorize_NotParticipant(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibilityPrivate, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	assert.ErrorIs(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionView), errs.ErrNotParticipant)
}

func TestAuthorize_ParticipantDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibilityPrivate, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	svc := service.NewParticipantService(p, c, anyNats())
	assert.ErrorIs(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionView), errs.ErrDatabaseUnavailable)
}

func TestAuthorize_InsufficientRole(t *testing.T) {
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(&model.ContestParticipant{Role: model.ParticipantRoleViewer}, nil)

	svc := service.NewParticipantService(p, mocks.NewContestRepository(t), anyNats())
	assert.ErrorIs(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionEditContest), errs.ErrInsufficientRole)
}

func TestAuthorize_OwnerCanDelete(t *testing.T) {
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)

	svc := service.NewParticipantService(p, mocks.NewContestRepository(t), anyNats())
	assert.NoError(t, svc.Authorize(context.Background(), uuid.New(), "u", service.ActionDeleteContest))
}

func TestGetParticipants_AuthorizeFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibilityPrivate, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	_, err := svc.GetParticipants(context.Background(), uuid.New(), "stranger")
	assert.ErrorIs(t, err, errs.ErrNotParticipant)
}

func TestGetParticipants_Success(t *testing.T) {
	contestID := uuid.New()
	want := []model.ContestParticipant{{ContestID: contestID, UserID: "u"}}
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetVisibilityByID(mock.Anything, mock.Anything).Return(model.ContestVisibilityPublic, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetAllByContestID(mock.Anything, mock.Anything).Return(want, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	got, err := svc.GetParticipants(context.Background(), contestID, "u")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetParticipantsInternal_Success(t *testing.T) {
	want := []model.ContestParticipant{{UserID: "u"}}
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetAllByContestID(mock.Anything, mock.Anything).Return(want, nil)

	svc := service.NewParticipantService(p, mocks.NewContestRepository(t), anyNats())
	got, err := svc.GetParticipantsInternal(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetParticipantsInternal_Error(t *testing.T) {
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetAllByContestID(mock.Anything, mock.Anything).Return(nil, errors.New("db"))

	svc := service.NewParticipantService(p, mocks.NewContestRepository(t), anyNats())
	_, err := svc.GetParticipantsInternal(context.Background(), uuid.New())
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGetMyContests_Success(t *testing.T) {
	want := []model.Contest{{Name: "c1"}}
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetAllByParticipantUserID(mock.Anything, "u", "search").Return(want, nil)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	got, err := svc.GetMyContests(context.Background(), "u", " search ")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetMyContests_Error(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetAllByParticipantUserID(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	_, err := svc.GetMyContests(context.Background(), "u", "")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUpdateParticipant_ContestNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "owner")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestUpdateParticipant_TerminalContest(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusFinished}, nil)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "owner")
	assert.ErrorIs(t, err, errs.ErrContestFinalized)
}

func TestUpdateParticipant_CannotChangeOwner(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	role := "viewer"
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "owner", &model.UpdateParticipantRequest{Role: &role}, "owner")
	assert.ErrorIs(t, err, errs.ErrCannotChangeOwner)
}

func TestUpdateParticipant_MaxSquaresBelowClaimed(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().CountSquaresByUser(mock.Anything, mock.Anything, "target").Return(8, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	maxSq := 5
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{MaxSquares: &maxSq}, "owner")
	assert.ErrorIs(t, err, errs.ErrSquareLimitTooLow)
}

func TestUpdateParticipant_Success(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

	svc := service.NewParticipantService(p, c, anyNats())
	role := "viewer"
	got, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{Role: &role}, "owner")
	require.NoError(t, err)
	assert.Equal(t, model.ParticipantRoleViewer, got.Role)
}

func TestUpdateParticipant_ContestDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db"))

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUpdateParticipant_AuthorizeFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "caller").Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "caller")
	assert.ErrorIs(t, err, errs.ErrNotParticipant)
}

func TestUpdateParticipant_TargetNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "owner")
	assert.ErrorIs(t, err, errs.ErrNotParticipant)
}

func TestUpdateParticipant_TargetDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(nil, errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{}, "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUpdateParticipant_CountSquaresDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().CountSquaresByUser(mock.Anything, mock.Anything, "target").Return(0, errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	maxSq := 5
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{MaxSquares: &maxSq}, "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUpdateParticipant_TotalAllocatedDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().CountSquaresByUser(mock.Anything, mock.Anything, "target").Return(3, nil)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(0, errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	maxSq := 8
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{MaxSquares: &maxSq}, "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUpdateParticipant_ExceedsSquareLimit(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().CountSquaresByUser(mock.Anything, mock.Anything, "target").Return(3, nil)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(96, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	maxSq := 15
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{MaxSquares: &maxSq}, "owner")
	assert.ErrorIs(t, err, errs.ErrNotEnoughSquares)
}

func TestUpdateParticipant_UpdateDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().Update(mock.Anything, mock.Anything).Return(errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	role := "viewer"
	_, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{Role: &role}, "owner")
	assert.Error(t, err)
}

func TestUpdateParticipant_MaxSquaresSuccess(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	p.EXPECT().CountSquaresByUser(mock.Anything, mock.Anything, "target").Return(3, nil)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(50, nil)
	p.EXPECT().Update(mock.Anything, mock.Anything).Return(nil)

	svc := service.NewParticipantService(p, c, anyNats())
	maxSq := 8
	got, err := svc.UpdateParticipant(context.Background(), uuid.New(), "target", &model.UpdateParticipantRequest{MaxSquares: &maxSq}, "owner")
	require.NoError(t, err)
	assert.Equal(t, 8, got.MaxSquares)
}

func TestRemoveParticipant_NotActive(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusFinished}, nil)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	assert.ErrorIs(t, svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner"), errs.ErrContestNotEditable)
}

func TestRemoveParticipant_CannotRemoveOwner(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	assert.ErrorIs(t, svc.RemoveParticipant(context.Background(), uuid.New(), "owner", "owner"), errs.ErrCannotRemoveOwner)
}

func TestRemoveParticipant_Success(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().Delete(mock.Anything, mock.Anything, "target").Return(nil)

	svc := service.NewParticipantService(p, c, anyNats())
	require.NoError(t, svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner"))
}

func TestRemoveParticipant_ContestNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestRemoveParticipant_ContestDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db"))

	svc := service.NewParticipantService(mocks.NewParticipantRepository(t), c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestRemoveParticipant_AuthorizeFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "caller").Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "caller")
	assert.ErrorIs(t, err, errs.ErrNotParticipant)
}

func TestRemoveParticipant_SelfRemovalSuccess(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "self").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().Delete(mock.Anything, mock.Anything, "self").Return(nil)

	svc := service.NewParticipantService(p, c, anyNats())
	require.NoError(t, svc.RemoveParticipant(context.Background(), uuid.New(), "self", "self"))
}

func TestRemoveParticipant_OwnerCannotRemoveSelf(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	assert.ErrorIs(t, svc.RemoveParticipant(context.Background(), uuid.New(), "owner", "owner"), errs.ErrCannotRemoveOwner)
}

func TestRemoveParticipant_TargetNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(nil, gorm.ErrRecordNotFound)

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.ErrorIs(t, err, errs.ErrNotParticipant)
}

func TestRemoveParticipant_TargetDBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(nil, errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestRemoveParticipant_ClearSquaresFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil).Once()
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db")).Once()
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.Error(t, err)
}

func TestRemoveParticipant_WithSquaresToClear(t *testing.T) {
	contestWithSquare := &model.Contest{
		Status:  model.ContestStatusActive,
		Squares: []model.Square{{Owner: "target"}},
	}
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(contestWithSquare, nil)
	c.EXPECT().ClearSquare(mock.Anything, mock.Anything).Return(&model.Square{}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().Delete(mock.Anything, mock.Anything, "target").Return(nil)

	svc := service.NewParticipantService(p, c, anyNats())
	require.NoError(t, svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner"))
}

func TestRemoveParticipant_ClearSquareFails(t *testing.T) {
	contestWithSquare := &model.Contest{
		Status:  model.ContestStatusActive,
		Squares: []model.Square{{Owner: "target"}},
	}
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(contestWithSquare, nil)
	c.EXPECT().ClearSquare(mock.Anything, mock.Anything).Return(nil, errors.New("clear failed"))
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.Error(t, err)
}

func TestRemoveParticipant_DeleteFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "owner").Return(&model.ContestParticipant{Role: model.ParticipantRoleOwner}, nil)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, "target").Return(&model.ContestParticipant{Role: model.ParticipantRoleParticipant}, nil)
	p.EXPECT().Delete(mock.Anything, mock.Anything, "target").Return(errors.New("db"))

	svc := service.NewParticipantService(p, c, anyNats())
	err := svc.RemoveParticipant(context.Background(), uuid.New(), "target", "owner")
	assert.Error(t, err)
}
