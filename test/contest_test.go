package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
		contest, status := createContest(router, authToken, oidcUser, name, homeTeam, awayTeam)

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

	t.Run("2_GetContestByOwnerAndName", func(t *testing.T) {
		contest, status := getContestByOwnerAndName(router, oidcUser, name)

		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, name, contest.Name)
	})

	t.Run("3_GetContestsByUser", func(t *testing.T) {
		response, status := getContestsByOwner(router, oidcUser, authToken)

		assert.Equal(t, status, http.StatusOK)
		assert.GreaterOrEqual(t, len(response.Contests), 1)
		assert.Equal(t, 1, response.Page)
		assert.True(t, response.Total >= 1)

		found := false
		for _, c := range response.Contests {
			if c.Name == name {
				found = true
				break
			}
		}

		assert.True(t, found, "Created contest should be in user's contest list")
	})

	t.Run("4_UpdateContest", func(t *testing.T) {
		newHomeTeam := "49ers"
		updateReq := model.UpdateContestRequest{
			HomeTeam: &newHomeTeam,
		}

		contest, status := updateContest(router, contestID, authToken, updateReq)

		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, "49ers", contest.HomeTeam)
		assert.Equal(t, "Eagles", contest.AwayTeam)
	})

	t.Run("5_FillAllSquares", func(t *testing.T) {
		contest, _ := getContestByOwnerAndName(router, oidcUser, name)
		require.Len(t, contest.Squares, 100)

		for i, square := range contest.Squares {
			squareValue := fmt.Sprintf("U%d", i%100)
			updateSquareReq := model.UpdateSquareRequest{
				Value: squareValue,
				Owner: oidcUser,
			}

			status := updateSquare(router, contestID, square.ID, authToken, updateSquareReq)
			assert.Equal(t, status, http.StatusOK)
		}

		contest, _ = getContestByOwnerAndName(router, oidcUser, name)
		for _, square := range contest.Squares {
			assert.NotEmpty(t, square.Value, "Square at row=%d, col=%d should have a value", square.Row, square.Col)
			assert.NotEmpty(t, square.Owner, "Square at row=%d, col=%d should have an owner", square.Row, square.Col)
		}
	})

	t.Run("6_StartContestAndSubmitResults", func(t *testing.T) {
		contest, status := startContest(router, contestID, authToken)
		assert.Equal(t, status, http.StatusOK)
		assert.Equal(t, model.ContestStatusQ1, contest.Status)

		submitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
		assert.Equal(t, status, http.StatusOK)

		submitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 14, AwayTeamScore: 10})
		assert.Equal(t, status, http.StatusOK)

		submitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 21, AwayTeamScore: 17})
		assert.Equal(t, status, http.StatusOK)

		submitQuarterResult(router, contestID, authToken, model.QuarterResultRequest{HomeTeamScore: 28, AwayTeamScore: 24})
		assert.Equal(t, status, http.StatusOK)

		contest, _ = getContestByOwnerAndName(router, oidcUser, name)
		assert.Equal(t, model.ContestStatusFinished, contest.Status)
		assert.Len(t, contest.QuarterResults, 4)
	})

	t.Run("7_DeleteContest", func(t *testing.T) {
		deleteContest(router, contestID, authToken)
		_, status := getContestByOwnerAndName(router, oidcUser, name)
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
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Owner_Too_Long",
			request: model.CreateContestRequest{
				Owner:    strings.Repeat("A", 256),
				Name:     "Valid Name",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     strings.Repeat("A", 21),
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Home_Team_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "Valid Name",
				HomeTeam: strings.Repeat("A", 21),
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Away_Team_Too_Long",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "Valid Name",
				HomeTeam: "Chiefs",
				AwayTeam: strings.Repeat("A", 21),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Empty_String",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Name_Min_Length_1",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "A",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Name_Max_Length_20",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     strings.Repeat("A", 20),
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Name_Over_Max_21",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     strings.Repeat("A", 21),
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "HomeTeam_Max_Length_20",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "HomeTeam Test",
				HomeTeam: strings.Repeat("A", 20),
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "HomeTeam_Over_Max_21",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "HomeTeam Test 2",
				HomeTeam: strings.Repeat("A", 21),
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "AwayTeam_Max_Length_20",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "AwayTeam Test",
				HomeTeam: "Chiefs",
				AwayTeam: strings.Repeat("A", 20),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "AwayTeam_Over_Max_21",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "AwayTeam Test 2",
				HomeTeam: "Chiefs",
				AwayTeam: strings.Repeat("A", 21),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Owner_Max_Length_255",
			request: model.CreateContestRequest{
				Owner:    oidcUser,
				Name:     "Owner Test",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Owner_Over_Max_256",
			request: model.CreateContestRequest{
				Owner:    strings.Repeat("A", 256),
				Name:     "Owner Test 2",
				HomeTeam: "Chiefs",
				AwayTeam: "Eagles",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contest, status := createContest(router, authToken, tc.request.Owner, tc.request.Name, tc.request.HomeTeam, tc.request.AwayTeam)
			slog.Info("negative contest result", "contest", contest)
			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestUpdateSquare_Validation(t *testing.T) {
	contestName := "Square Val Test"
	contest, status := createContest(router, authToken, oidcUser, contestName, "Home", "Away")
	require.Equal(t, http.StatusOK, status)
	require.NotEqual(t, uuid.Nil, contest.ID)

	contest, status = getContestByOwnerAndName(router, oidcUser, contestName)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, contest.Squares, 100)

	squareID := contest.Squares[0].ID

	t.Run("Successful_Square_Update", func(t *testing.T) {
		updateReq := model.UpdateSquareRequest{
			Value: "ABC",
			Owner: oidcUser,
		}
		status := updateSquare(router, contest.ID, squareID, authToken, updateReq)
		assert.Equal(t, http.StatusOK, status)
	})

	testCases := []struct {
		name           string
		request        model.UpdateSquareRequest
		expectedStatus int
	}{
		{
			name: "Value_Too_Short",
			request: model.UpdateSquareRequest{
				Value: "",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Value_Too_Long",
			request: model.UpdateSquareRequest{
				Value: "ABCD",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Value_Not_Uppercase",
			request: model.UpdateSquareRequest{
				Value: "abc",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Value_Not_Alphanumeric",
			request: model.UpdateSquareRequest{
				Value: "A$B",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Value_Contains_Dangerous_Chars",
			request: model.UpdateSquareRequest{
				Value: "A<B",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Owner_Missing",
			request: model.UpdateSquareRequest{
				Value: "ABC",
				Owner: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Owner_Contains_Dangerous_Chars",
			request: model.UpdateSquareRequest{
				Value: "ABC",
				Owner: "user<script>",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Valid_Single_Char",
			request: model.UpdateSquareRequest{
				Value: "A",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid_Two_Chars",
			request: model.UpdateSquareRequest{
				Value: "AB",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid_Three_Chars",
			request: model.UpdateSquareRequest{
				Value: "XYZ",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Valid_With_Numbers",
			request: model.UpdateSquareRequest{
				Value: "A1B",
				Owner: oidcUser,
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := updateSquare(router, contest.ID, squareID, authToken, tc.request)
			assert.Equal(t, tc.expectedStatus, status, "Test case: %s", tc.name)
		})
	}

	deleteContest(router, contest.ID, authToken)
}

func TestQuarterResult_Validation(t *testing.T) {
	setupContest := func(name string) (uuid.UUID, *model.Contest) {
		contest, status := createContest(router, authToken, oidcUser, name, "Home", "Away")
		require.Equal(t, http.StatusOK, status)
		require.NotEqual(t, uuid.Nil, contest.ID)

		contest, status = getContestByOwnerAndName(router, oidcUser, name)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, contest.Squares, 100)

		for _, square := range contest.Squares {
			updateReq := model.UpdateSquareRequest{
				Value: "AAA",
				Owner: oidcUser,
			}
			status := updateSquare(router, contest.ID, square.ID, authToken, updateReq)
			require.Equal(t, http.StatusOK, status)
		}

		_, status = startContest(router, contest.ID, authToken)
		require.Equal(t, http.StatusOK, status)

		return contest.ID, contest
	}

	t.Run("Successful_Quarter_Result", func(t *testing.T) {
		contestID, _ := setupContest("QR Success")
		defer deleteContest(router, contestID, authToken)

		quarterReq := model.QuarterResultRequest{
			HomeTeamScore: 14,
			AwayTeamScore: 7,
		}
		status := submitQuarterResult(router, contestID, authToken, quarterReq)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("Both_Scores_Zero", func(t *testing.T) {
		contestID, _ := setupContest("QR Zero Both")
		defer deleteContest(router, contestID, authToken)

		quarterReq := model.QuarterResultRequest{
			HomeTeamScore: 0,
			AwayTeamScore: 0,
		}
		status := submitQuarterResult(router, contestID, authToken, quarterReq)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("Home_Score_Zero", func(t *testing.T) {
		contestID, _ := setupContest("QR Zero Home")
		defer deleteContest(router, contestID, authToken)

		quarterReq := model.QuarterResultRequest{
			HomeTeamScore: 0,
			AwayTeamScore: 21,
		}
		status := submitQuarterResult(router, contestID, authToken, quarterReq)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("Away_Score_Zero", func(t *testing.T) {
		contestID, _ := setupContest("QR Zero Away")
		defer deleteContest(router, contestID, authToken)

		quarterReq := model.QuarterResultRequest{
			HomeTeamScore: 28,
			AwayTeamScore: 0,
		}
		status := submitQuarterResult(router, contestID, authToken, quarterReq)
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("Max_Valid_Scores", func(t *testing.T) {
		contestID, _ := setupContest("QR Max Scores")
		defer deleteContest(router, contestID, authToken)

		quarterReq := model.QuarterResultRequest{
			HomeTeamScore: 9999,
			AwayTeamScore: 9999,
		}
		status := submitQuarterResult(router, contestID, authToken, quarterReq)
		assert.Equal(t, http.StatusOK, status)
	})

	contestID, _ := setupContest("QR Validation")
	defer deleteContest(router, contestID, authToken)

	testCases := []struct {
		name           string
		request        model.QuarterResultRequest
		expectedStatus int
	}{
		{
			name: "Home_Score_Negative",
			request: model.QuarterResultRequest{
				HomeTeamScore: -1,
				AwayTeamScore: 14,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Away_Score_Negative",
			request: model.QuarterResultRequest{
				HomeTeamScore: 14,
				AwayTeamScore: -1,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Both_Scores_Negative",
			request: model.QuarterResultRequest{
				HomeTeamScore: -5,
				AwayTeamScore: -10,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Home_Score_Too_High",
			request: model.QuarterResultRequest{
				HomeTeamScore: 10000,
				AwayTeamScore: 14,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Away_Score_Too_High",
			request: model.QuarterResultRequest{
				HomeTeamScore: 14,
				AwayTeamScore: 10000,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status := submitQuarterResult(router, contestID, authToken, tc.request)
			assert.Equal(t, tc.expectedStatus, status, "Test case: %s", tc.name)
		})
	}
}

func createContest(router http.Handler, authToken, oidcUser, name, homeTeam, awayTeam string) (*model.Contest, int) {
	reqBody := model.CreateContestRequest{
		Owner:    oidcUser,
		Name:     name,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPut, "/contests", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func getContestByOwnerAndName(router http.Handler, owner, name string) (*model.Contest, int) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/owner/%s/name/%s", owner, name), nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func getContestsByOwner(router http.Handler, oidcUser, authToken string) (model.PaginatedContestResponse, int) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/contests/owner/%s?page=1&limit=10", oidcUser), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response model.PaginatedContestResponse
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	return response, w.Code
}

func updateContest(router http.Handler, contestID uuid.UUID, authToken string, updateReq model.UpdateContestRequest) (model.Contest, int) {
	body, _ := json.Marshal(updateReq)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s", contestID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return contest, w.Code
}

func updateSquare(router http.Handler, contestID uuid.UUID, squareID uuid.UUID, authToken string, updateReq model.UpdateSquareRequest) int {
	body, _ := json.Marshal(updateReq)

	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", contestID, squareID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Code
}

func startContest(router http.Handler, contestID uuid.UUID, authToken string) (*model.Contest, int) {
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/start", contestID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var contest model.Contest
	_ = json.Unmarshal(w.Body.Bytes(), &contest)

	return &contest, w.Code
}

func submitQuarterResult(router http.Handler, contestID uuid.UUID, authToken string, reqBody model.QuarterResultRequest) int {
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", contestID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Code
}

func deleteContest(router http.Handler, contestID uuid.UUID, authToken string) int {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/contests/%s", contestID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w.Code
}
