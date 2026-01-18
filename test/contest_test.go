package test

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContest_FullLifecycle(t *testing.T) {
	var contestID uuid.UUID
	const (
		name     = "Super Bowl 2025"
		homeTeam = "Chiefs"
		awayTeam = "Eagles"
	)
	t.Run("1_CreateContest", func(t *testing.T) {
		contest, status := CreateContest(router, authToken, oidcUser, name, homeTeam, awayTeam)

		assert.NotNil(t, contest)
		assert.Equal(t, status, http.StatusOK)
		assert.NotEqual(t, uuid.Nil, contest.ID)
		assert.Equal(t, name, contest.Name)
		assert.Equal(t, oidcUser, contest.Owner)
		assert.Equal(t, homeTeam, contest.HomeTeam)
		assert.Equal(t, awayTeam, contest.AwayTeam)
		assert.Equal(t, model.ContestStatusActive, contest.Status)

		contestID = contest.ID
		require.NotEqual(t, uuid.Nil, contestID, "contestID must be set for subsequent tests")
	})

	t.Run("2_GetContestByID", func(t *testing.T) {
		contest, status := GetContestByID(router, contestID)

		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, contestID, contest.ID)
		assert.Equal(t, name, contest.Name)
		assert.Equal(t, status, http.StatusOK)
	})

	t.Run("3_GetContestsByUser", func(t *testing.T) {
		response, status := GetContestsByUser(router, oidcUser, authToken)

		assert.Equal(t, status, http.StatusOK)
		assert.GreaterOrEqual(t, len(response.Contests), 1)
		assert.Equal(t, 1, response.Page)
		assert.True(t, response.Total >= 1)

		found := false
		for _, c := range response.Contests {
			if c.ID == contestID {
				found = true
				break
			}
		}

		assert.True(t, found, "Created contest should be in user's contest list")
	})

	t.Run("4_UpdateContest", func(t *testing.T) {
		newName := "Super Bowl LX"
		newHomeTeam := "49ers"
		updateReq := model.UpdateContestRequest{
			Name:     &newName,
			HomeTeam: &newHomeTeam,
		}

		contest, status := UpdateContest(router, contestID, authToken, updateReq)

		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, "Super Bowl LX", contest.Name)
		assert.Equal(t, "49ers", contest.HomeTeam)
		assert.Equal(t, "Eagles", contest.AwayTeam)
	})

	t.Run("5_FillAllSquares", func(t *testing.T) {
		contest, _ := GetContestByID(router, contestID)
		require.Len(t, contest.Squares, 100)

		for i, square := range contest.Squares {
			squareValue := fmt.Sprintf("U%d", i%100)
			updateSquareReq := model.UpdateSquareRequest{
				Value: squareValue,
				Owner: oidcUser,
			}

			status := UpdateSquare(router, contestID, square.ID, authToken, updateSquareReq)
			assert.Equal(t, status, http.StatusOK)
		}

		contest, _ = GetContestByID(router, contestID)
		for _, square := range contest.Squares {
			assert.NotEmpty(t, square.Value, "Square at row=%d, col=%d should have a value", square.Row, square.Col)
			assert.NotEmpty(t, square.Owner, "Square at row=%d, col=%d should have an owner", square.Row, square.Col)
		}
	})

	t.Run("6_StartContestAndSubmitResults", func(t *testing.T) {
		contest, status := StartContest(router, contestID, authToken)
		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, model.ContestStatusQ1, contest.Status)

		SubmitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
		assert.Equal(t, status, http.StatusOK)

		SubmitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 14, AwayTeamScore: 10})
		assert.Equal(t, status, http.StatusOK)

		SubmitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 21, AwayTeamScore: 17})
		assert.Equal(t, status, http.StatusOK)

		SubmitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 28, AwayTeamScore: 24})
		assert.Equal(t, status, http.StatusOK)

		contest, _ = GetContestByID(router, contestID)
		assert.Equal(t, model.ContestStatusFinished, contest.Status)
		assert.Len(t, contest.QuarterResults, 4)
	})

	t.Run("7_DeleteContest", func(t *testing.T) {
		DeleteContest(router, contestID, authToken)
		_, status := GetContestByID(router, contestID)
		assert.Equal(t, status, http.StatusNotFound)
	})
}

func TestCreateContest_Validation(t *testing.T) {
	testCases := []struct {
		name           string
		request        model.CreateContestRequest
		expectedStatus int
	}{
		{
			name: "Missing_Owner",
			request: model.CreateContestRequest{
				Owner:    "",
				Name:     "Valid Name",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: 400,
		},
		{
			name: "Owner_Too_Long",
			request: model.CreateContestRequest{
				Owner: `ThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLong
	ThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLong
	ThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLongThisOwnerNameIsTooLong`,
				Name: "Valid Name",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: 400,
		},
		{
			name: "Name_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: 400,
		},
		{
			name: "Name_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "ThisNameIsWayTooLongForValidationCheck",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: 400,
		},
		{
			name: "Home_Team_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "Valid Name",
				HomeTeam: "ThisHomeTeamNameIsWayTooLongForValidation",
				AwayTeam: "Eagles",
			},
			expectedStatus: 400,
		},
		{
			name: "Away_Team_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "Valid Name",
				HomeTeam: "Chiefs",
				AwayTeam: "ThisAwayTeamNameIsWayTooLongForValidation",
			},
			expectedStatus: 400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contest, status := CreateContest(router, authToken, tc.request.Owner, tc.request.Name, tc.request.HomeTeam, tc.request.AwayTeam)
			slog.Info("negative contest result", "contest", contest)
			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}
