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
)

func newUserService(t *testing.T) (service.UserService, *mocks.UserRepository) {
	t.Helper()
	repo := mocks.NewUserRepository(t)
	return service.NewUserService(repo, anyNats()), repo
}

func TestUserService_IsTokenValid(t *testing.T) {
	future := time.Now().Add(time.Hour).Unix()
	valid := &model.Claims{Email: "a@b.com", EmailVerified: true, IssuedAt: 100, Expire: future}

	t.Run("valid and not revoked", func(t *testing.T) {
		svc, repo := newUserService(t)
		repo.EXPECT().IsTokenRevoked(mock.Anything, "a@b.com", int64(100)).Return(false, nil)

		ok, err := svc.IsTokenValid(context.Background(), valid)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("revoked by deletion", func(t *testing.T) {
		svc, repo := newUserService(t)
		repo.EXPECT().IsTokenRevoked(mock.Anything, "a@b.com", int64(100)).Return(true, nil)

		ok, err := svc.IsTokenValid(context.Background(), valid)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("expired short-circuits before db", func(t *testing.T) {
		svc, _ := newUserService(t)
		expired := &model.Claims{Email: "a@b.com", EmailVerified: true, Expire: time.Now().Add(-time.Hour).Unix()}

		ok, err := svc.IsTokenValid(context.Background(), expired)
		require.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("nil claims", func(t *testing.T) {
		svc, _ := newUserService(t)
		ok, err := svc.IsTokenValid(context.Background(), nil)
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestUserService_GetProfile_Success(t *testing.T) {
	svc, repo := newUserService(t)
	// initials are seeded from the display name on first visit
	repo.EXPECT().GetOrCreate(mock.Anything, "a@b.com", "Max Morhardt", "MM").
		Return(&model.User{Email: "a@b.com", DisplayName: "Max Morhardt", DefaultInitials: "MM"}, nil)

	user, err := svc.GetProfile(context.Background(), "a@b.com", "Max Morhardt")

	require.NoError(t, err)
	assert.Equal(t, "a@b.com", user.Email)
	assert.Equal(t, "MM", user.DefaultInitials)
}

func TestUserService_GetProfile_Error(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().GetOrCreate(mock.Anything, "a@b.com", "Max", "M").
		Return(nil, errors.New("db down"))

	user, err := svc.GetProfile(context.Background(), "a@b.com", "Max")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, user)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	svc, repo := newUserService(t)
	squares := []model.Square{{ID: uuid.New(), ContestID: uuid.New(), Owner: "a@b.com", Value: "MM"}}
	repo.EXPECT().UpdateProfile(mock.Anything, "a@b.com", "MM").
		Return(&model.User{Email: "a@b.com", DefaultInitials: "MM"}, squares, nil)

	user, err := svc.UpdateProfile(context.Background(), "a@b.com", "MM")

	require.NoError(t, err)
	assert.Equal(t, "MM", user.DefaultInitials)
}

func TestUserService_UpdateProfile_Error(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().UpdateProfile(mock.Anything, "a@b.com", "MM").
		Return(nil, nil, errors.New("db down"))

	user, err := svc.UpdateProfile(context.Background(), "a@b.com", "MM")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, user)
}

func TestUserService_GetStats_Success(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(&model.UserStatsResponse{ContestsCreated: 2, QuarterWins: 1}, nil)

	stats, err := svc.GetStats(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.ContestsCreated)
	assert.Equal(t, int64(1), stats.QuarterWins)
}

func TestUserService_GetStats_Error(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().GetStats(mock.Anything, "a@b.com").
		Return(nil, errors.New("db down"))

	stats, err := svc.GetStats(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, stats)
}

func TestUserService_DeleteAccount_Success(t *testing.T) {
	svc, repo := newUserService(t)

	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return([]model.UserActiveContest{}, nil)
	repo.EXPECT().ScrubUserData(mock.Anything, "a@b.com").Return(nil)

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.NoError(t, err)
}

func TestUserService_DeleteAccount_BlockedByActiveContests(t *testing.T) {
	svc, repo := newUserService(t)
	blockers := []model.UserActiveContest{
		{ID: uuid.NewString(), Name: "test", Owner: "a@b.com", Role: "owner"},
		{ID: uuid.NewString(), Name: "pool", Owner: "other@b.com", Role: "participant"},
	}

	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return(blockers, nil)

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrAccountActiveContests)
}

func TestUserService_DeleteAccount_ListError(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return(nil, errors.New("db down"))

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUserService_DeleteAccount_ScrubError(t *testing.T) {
	svc, repo := newUserService(t)

	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return([]model.UserActiveContest{}, nil)
	repo.EXPECT().ScrubUserData(mock.Anything, "a@b.com").Return(errors.New("scrub failed"))

	err := svc.DeleteAccount(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
}

func TestUserService_GetActiveContests_Success(t *testing.T) {
	svc, repo := newUserService(t)
	active := []model.UserActiveContest{{ID: uuid.NewString(), Name: "pool", Owner: "a@b.com", Role: "owner"}}
	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return(active, nil)

	got, err := svc.GetActiveContests(context.Background(), "a@b.com")

	require.NoError(t, err)
	assert.Equal(t, active, got)
}

func TestUserService_GetActiveContests_Error(t *testing.T) {
	svc, repo := newUserService(t)
	repo.EXPECT().GetActiveContests(mock.Anything, "a@b.com").Return(nil, errors.New("db down"))

	got, err := svc.GetActiveContests(context.Background(), "a@b.com")

	require.ErrorIs(t, err, errs.ErrDatabaseUnavailable)
	assert.Nil(t, got)
}
