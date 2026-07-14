package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestCreateInvite_ContestNotFound(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	_, err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{}, "u")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func inviteSvc(inv *mocks.InviteRepository, p *mocks.ParticipantRepository, c *mocks.ContestRepository, pSvc *mocks.ParticipantService) service.InviteService {
	return service.NewInviteService(inv, p, c, pSvc, anyNats())
}

func TestCreateInvite_DBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("boom"))

	_, err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{}, "u")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestCreateInvite_Terminal(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusFinished}, nil)

	_, err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{}, "u")
	assert.ErrorIs(t, err, errs.ErrContestFinalized)
}

func TestCreateInvite_Unauthorized(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errs.ErrInsufficientRole)

	_, err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, pSvc).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{}, "u")
	assert.ErrorIs(t, err, errs.ErrInsufficientRole)
}

func TestCreateInvite_RepoCreateFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db write failed"))

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, pSvc).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{MaxSquares: 5, Role: "participant"}, "u")
	require.Error(t, err)
}

func TestCreateInvite_ParticipantZeroSquares(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, pSvc).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{MaxSquares: 0, Role: "participant"}, "u")
	assert.ErrorIs(t, err, errs.ErrInvalidSquareCount)
}

func TestCreateInvite_ViewerForcesZeroSquares(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	got, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, pSvc).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{MaxSquares: 50, Role: "viewer"}, "owner")
	require.NoError(t, err)
	assert.Equal(t, model.ParticipantRoleViewer, got.Role)
	assert.Equal(t, 0, got.MaxSquares)
}

func TestCreateInvite_Success(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	got, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, pSvc).
		CreateInvite(context.Background(), uuid.New(), &model.CreateInviteRequest{MaxSquares: 5, Role: "participant", ExpiresIn: 60}, "owner")
	require.NoError(t, err)
	assert.Equal(t, model.ParticipantRole("participant"), got.Role)
	assert.NotNil(t, got.ExpiresAt)
}

func TestGetInvitePreview_NotFound(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	assert.ErrorIs(t, err, errs.ErrInviteNotFound)
}

func TestGetInvitePreview_DBError(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGetInvitePreview_Expired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ExpiresAt: &past}, nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	assert.ErrorIs(t, err, errs.ErrInviteExpired)
}

func TestGetInvitePreview_MaxUsesReached(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{MaxUses: 1, Uses: 1}, nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	assert.ErrorIs(t, err, errs.ErrInviteMaxUsesReached)
}

func TestGetInvitePreview_ContestFetchFails(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: uuid.New()}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestGetInvitePreview_Success(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID, Role: model.ParticipantRoleParticipant, MaxSquares: 5}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{ID: contestID, Name: "Pool", Owner: "owner"}, nil)

	got, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		GetInvitePreview(context.Background(), "tok")
	require.NoError(t, err)
	assert.Equal(t, contestID, got.ContestID)
	assert.Equal(t, "Pool", got.ContestName)
	assert.Equal(t, "owner", got.Owner)
}

func TestRedeemInvite_DBError(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestRedeemInvite_Expired(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ExpiresAt: &past}, nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrInviteExpired)
}

func TestRedeemInvite_MaxUsesReached(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{MaxUses: 2, Uses: 2}, nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrInviteMaxUsesReached)
}

func TestRedeemInvite_ContestTerminal(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusFinished}, nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrContestFinalized)
}

func TestRedeemInvite_AlreadyParticipant(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(&model.ContestParticipant{}, nil)

	_, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrAlreadyParticipant)
}

func TestRedeemInvite_ParticipantCheckDBError(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	_, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestRedeemInvite_GetTotalAllocatedSquaresError(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID, MaxSquares: 10}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(0, errors.New("db error"))

	_, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestRedeemInvite_NotEnoughSquares(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID, MaxSquares: 50}, nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(60, nil)

	_, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).RedeemInvite(context.Background(), "tok", "u")
	assert.ErrorIs(t, err, errs.ErrNotEnoughSquares)
}

func TestRedeemInvite_RepoFails(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID, MaxSquares: 10}, nil)
	inv.EXPECT().RedeemInvite(mock.Anything, mock.Anything, mock.Anything).Return(errors.New("tx failed"))
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(0, nil)

	_, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).
		RedeemInvite(context.Background(), "tok", "u")
	require.Error(t, err)
}

func TestRedeemInvite_Success(t *testing.T) {
	contestID := uuid.New()
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetByToken(mock.Anything, mock.Anything).Return(&model.ContestInvite{ContestID: contestID, Role: model.ParticipantRoleParticipant, MaxSquares: 10}, nil)
	inv.EXPECT().RedeemInvite(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	p := mocks.NewParticipantRepository(t)
	p.EXPECT().GetByContestAndUser(mock.Anything, mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	p.EXPECT().GetTotalAllocatedSquares(mock.Anything, mock.Anything).Return(0, nil)

	got, err := inviteSvc(inv, p, c, mocks.NewParticipantService(t)).RedeemInvite(context.Background(), "tok", "u")
	require.NoError(t, err)
	assert.Equal(t, "u", got.UserID)
	assert.Equal(t, 10, got.MaxSquares)
}

func TestGetInvitesByContestID_Success(t *testing.T) {
	want := []model.ContestInvite{{Token: "t1"}}
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetAllByContestID(mock.Anything, mock.Anything).Return(want, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	got, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), pSvc).
		GetInvitesByContestID(context.Background(), uuid.New(), "owner")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetInvitesByContestID_RepoError(t *testing.T) {
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().GetAllByContestID(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := inviteSvc(inv, mocks.NewParticipantRepository(t), mocks.NewContestRepository(t), pSvc).
		GetInvitesByContestID(context.Background(), uuid.New(), "owner")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestDeleteInvite_Terminal(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusFinished}, nil)

	err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		DeleteInvite(context.Background(), uuid.New(), uuid.New(), "u")
	assert.ErrorIs(t, err, errs.ErrContestFinalized)
}

func TestDeleteInvite_DBError(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, mocks.NewParticipantService(t)).
		DeleteInvite(context.Background(), uuid.New(), uuid.New(), "u")
	assert.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestDeleteInvite_AuthorizeFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errs.ErrInsufficientRole)

	err := inviteSvc(mocks.NewInviteRepository(t), mocks.NewParticipantRepository(t), c, pSvc).
		DeleteInvite(context.Background(), uuid.New(), uuid.New(), "u")
	assert.ErrorIs(t, err, errs.ErrInsufficientRole)
}

func TestDeleteInvite_RepoDeleteFails(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().Delete(mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, pSvc).
		DeleteInvite(context.Background(), uuid.New(), uuid.New(), "u")
	require.Error(t, err)
}

func TestDeleteInvite_Success(t *testing.T) {
	c := mocks.NewContestRepository(t)
	c.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&model.Contest{Status: model.ContestStatusActive}, nil)
	pSvc := mocks.NewParticipantService(t)
	pSvc.EXPECT().Authorize(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	inv := mocks.NewInviteRepository(t)
	inv.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

	err := inviteSvc(inv, mocks.NewParticipantRepository(t), c, pSvc).
		DeleteInvite(context.Background(), uuid.New(), uuid.New(), "u")
	require.NoError(t, err)
}
