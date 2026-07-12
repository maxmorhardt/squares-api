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
)

func newUserService(t *testing.T) (service.UserService, *mocks.UserRepository, *mocks.ContestService) {
	t.Helper()
	repo := mocks.NewUserRepository(t)
	contestSvc := mocks.NewContestService(t)
	return service.NewUserService(repo, contestSvc), repo, contestSvc
}

func TestUserService_GetProfile_Success(t *testing.T) {
	svc, repo, _ := newUserService(t)
	repo.EXPECT().GetOrCreate(mock.Anything, "a@b.com", "Max").
		Return(&model.User{Email: "a@b.com", DisplayName: "Max"}, nil)

	user, err := svc.GetProfile(context.Background(), "a@b.com", "Max")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
}

func TestUserService_GetProfile_Error(t *testing.T) {
	svc, repo, _ := newUserService(t)
	repo.EXPECT().GetOrCreate(mock.Anything, "a@b.com", "Max").
		Return(nil, errors.New("db down"))

	user, err := svc.GetProfile(context.Background(), "a@b.com", "Max")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, user)
}

func TestUserService_GetStats_Success(t *testing.T) {
	svc, repo, _ := newUserService(t)
	repo.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(&model.UserStatsResponse{ContestsCreated: 2, QuarterWins: 1}, nil)

	stats, err := svc.GetStats(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.ContestsCreated)
	assert.Equal(t, int64(1), stats.QuarterWins)
}

func TestUserService_GetStats_Error(t *testing.T) {
	svc, repo, _ := newUserService(t)
	repo.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(nil, errors.New("db down"))

	stats, err := svc.GetStats(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, stats)
}

func TestUserService_DeleteAccount_Success(t *testing.T) {
	svc, repo, contestSvc := newUserService(t)
	id1, id2 := uuid.New(), uuid.New()

	repo.EXPECT().GetOwnedActiveContestIDs(mock.Anything, "a@b.com").Return([]uuid.UUID{id1, id2}, nil)
	contestSvc.EXPECT().DeleteContest(mock.Anything, id1, "a@b.com").Return(nil)
	contestSvc.EXPECT().DeleteContest(mock.Anything, id2, "a@b.com").Return(nil)
	repo.EXPECT().ScrubUserData(mock.Anything, "a@b.com").Return(nil)

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.NoError(t, err)
}

func TestUserService_DeleteAccount_ListError(t *testing.T) {
	svc, repo, _ := newUserService(t)
	repo.EXPECT().GetOwnedActiveContestIDs(mock.Anything, "a@b.com").Return(nil, errors.New("db down"))

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUserService_DeleteAccount_DeleteContestError(t *testing.T) {
	svc, repo, contestSvc := newUserService(t)
	id1 := uuid.New()
	deleteErr := errs.ErrContestFinalized

	repo.EXPECT().GetOwnedActiveContestIDs(mock.Anything, "a@b.com").Return([]uuid.UUID{id1}, nil)
	contestSvc.EXPECT().DeleteContest(mock.Anything, id1, "a@b.com").Return(deleteErr)

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, deleteErr)
}

func TestUserService_DeleteAccount_ScrubError(t *testing.T) {
	svc, repo, _ := newUserService(t)

	repo.EXPECT().GetOwnedActiveContestIDs(mock.Anything, "a@b.com").Return([]uuid.UUID{}, nil)
	repo.EXPECT().ScrubUserData(mock.Anything, "a@b.com").Return(errors.New("scrub failed"))

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}
