package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContest_FullLifecycle(t *testing.T) {
	var contestID uuid.UUID

	t.Run("create contest", func(t *testing.T) {
		contest, status := createContest(t, ownerToken, ownerUser, "Super Bowl", "Chiefs", "Eagles", 50)
		require.Equal(t, http.StatusOK, status)
		require.NotEqual(t, uuid.Nil, contest.ID)
		assert.Equal(t, "Super Bowl", contest.Name)
		assert.Equal(t, ownerUser, contest.Owner)
		assert.Equal(t, model.ContestStatusActive, contest.Status)
		contestID = contest.ID
	})

	t.Run("get contests by owner", func(t *testing.T) {
		resp, status := getContestsByOwner(t, ownerUser, ownerToken)
		require.Equal(t, http.StatusOK, status)
		assert.GreaterOrEqual(t, len(resp.Contests), 1)
		found := false
		for _, c := range resp.Contests {
			if c.ID == contestID {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("update contest", func(t *testing.T) {
		awayTeam := "49ers"
		contest, status := updateContest(t, contestID, ownerToken, model.UpdateContestRequest{AwayTeam: &awayTeam})
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "49ers", contest.AwayTeam)
	})

	t.Run("websocket connect receives connected message", func(t *testing.T) {
		require.NotEqual(t, uuid.Nil, contestID)

		server := httptest.NewServer(router)
		defer server.Close()

		wsURL := "ws://" + server.Listener.Addr().String() +
			"/ws/contests/" + contestID.String()

		header := http.Header{}
		header.Set("Origin", "http://localhost:3000")
		header.Set("Sec-WebSocket-Protocol", ownerToken)

		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err)
		defer conn.Close()

		_, msgBytes, err := conn.ReadMessage()
		require.NoError(t, err)

		var msg model.WSUpdate
		require.NoError(t, json.Unmarshal(msgBytes, &msg))
		assert.Equal(t, model.ConnectedType, msg.Type)
		require.NotNil(t, msg.Contest)
		assert.Equal(t, contestID, msg.Contest.ID)
	})

	var inviteToken string

	t.Run("create invite link", func(t *testing.T) {
		invite, status := createInvite(t, contestID, ownerToken, model.CreateInviteRequest{
			MaxSquares: 50,
			Role:       "participant",
		})
		require.Equal(t, http.StatusOK, status)
		require.NotEmpty(t, invite.Token)
		inviteToken = invite.Token
	})

	t.Run("preview invite", func(t *testing.T) {
		preview, status := getInvitePreview(t, inviteToken)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "Super Bowl", preview.ContestName)
		assert.Equal(t, ownerUser, preview.Owner)
	})

	t.Run("list invites", func(t *testing.T) {
		invites, status := getInvites(t, contestID, ownerToken)
		require.Equal(t, http.StatusOK, status)
		assert.GreaterOrEqual(t, len(invites), 1)
	})

	t.Run("redeem invite as member", func(t *testing.T) {
		status := redeemInvite(t, inviteToken, memberToken)
		assert.Equal(t, http.StatusCreated, status)
	})

	t.Run("list participants includes member", func(t *testing.T) {
		participants, status := getParticipants(t, contestID, ownerToken)
		require.Equal(t, http.StatusOK, status)
		found := false
		for _, p := range participants {
			if p.UserID == memberUser {
				found = true
				break
			}
		}
		assert.True(t, found, "member should appear in participants after redeeming invite")
	})

	t.Run("fill all squares", func(t *testing.T) {
		contest, status := getContest(t, contestID)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, contest.Squares, 100, "contest must have 100 squares")

		// alternate between owner and member so both have squares
		for i, sq := range contest.Squares {
			if sq.Owner != "" {
				continue
			}
			claimer := ownerToken
			claimerUser := ownerUser
			if i%2 == 0 {
				claimer = memberToken
				claimerUser = memberUser
			}
			_, sqStatus := updateSquare(t, contestID, sq.ID, claimer, claimerUser, fmt.Sprintf("V%d", i))
			require.Equal(t, http.StatusOK, sqStatus, "square %d claim should succeed", i)
		}

		// confirm no empty squares remain
		filled, status2 := getContest(t, contestID)
		require.Equal(t, http.StatusOK, status2)
		for _, sq := range filled.Squares {
			assert.NotEmpty(t, sq.Owner, "square row=%d col=%d should be claimed", sq.Row, sq.Col)
		}
	})

	t.Run("start contest transitions to Q1", func(t *testing.T) {
		contest, status := startContest(t, contestID, ownerToken)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, model.ContestStatusQ1, contest.Status)
	})

	t.Run("record Q1 result transitions to Q2", func(t *testing.T) {
		status := submitQuarterResult(t, contestID, model.QuarterResultRequest{HomeTeamScore: 7, AwayTeamScore: 3})
		require.Equal(t, http.StatusOK, status)
		contest, s := getContest(t, contestID)
		require.Equal(t, http.StatusOK, s)
		assert.Equal(t, model.ContestStatusQ2, contest.Status)
		require.Len(t, contest.QuarterResults, 1)
		assert.Equal(t, 1, contest.QuarterResults[0].Quarter)
	})

	t.Run("record Q2 result transitions to Q3", func(t *testing.T) {
		status := submitQuarterResult(t, contestID, model.QuarterResultRequest{HomeTeamScore: 14, AwayTeamScore: 10})
		require.Equal(t, http.StatusOK, status)
		contest, s := getContest(t, contestID)
		require.Equal(t, http.StatusOK, s)
		assert.Equal(t, model.ContestStatusQ3, contest.Status)
		require.Len(t, contest.QuarterResults, 2)
	})

	t.Run("record Q3 result transitions to Q4", func(t *testing.T) {
		status := submitQuarterResult(t, contestID, model.QuarterResultRequest{HomeTeamScore: 21, AwayTeamScore: 17})
		require.Equal(t, http.StatusOK, status)
		contest, s := getContest(t, contestID)
		require.Equal(t, http.StatusOK, s)
		assert.Equal(t, model.ContestStatusQ4, contest.Status)
		require.Len(t, contest.QuarterResults, 3)
	})

	t.Run("record Q4 result transitions to FINISHED", func(t *testing.T) {
		status := submitQuarterResult(t, contestID, model.QuarterResultRequest{HomeTeamScore: 28, AwayTeamScore: 24})
		require.Equal(t, http.StatusOK, status)
		contest, s := getContest(t, contestID)
		require.Equal(t, http.StatusOK, s)
		assert.Equal(t, model.ContestStatusFinished, contest.Status)
		require.Len(t, contest.QuarterResults, 4)
		assert.Equal(t, 4, contest.QuarterResults[3].Quarter)
		for _, qr := range contest.QuarterResults {
			assert.NotEmpty(t, qr.Winner, "quarter %d should have a winner", qr.Quarter)
		}
	})

	t.Run("finished contest cannot be deleted", func(t *testing.T) {
		status := deleteContest(t, contestID, ownerToken)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("active contest can be deleted", func(t *testing.T) {
		c2, status := createContest(t, ownerToken, ownerUser, "Cleanup Bowl", "Chiefs", "Eagles", 4)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, http.StatusNoContent, deleteContest(t, c2.ID, ownerToken))
	})
}

// --- request helpers ---

func createContest(t *testing.T, token, owner, name, homeTeam, awayTeam string, maxSquares int) (contest *model.Contest, status int) {
	t.Helper()
	body, _ := json.Marshal(model.CreateContestRequest{
		Owner:      owner,
		Name:       name,
		HomeTeam:   homeTeam,
		AwayTeam:   awayTeam,
		MaxSquares: maxSquares,
	})
	code, resp := doRequest(t, http.MethodPut, "/contests", token, body)
	var c model.Contest
	_ = json.Unmarshal(resp, &c)
	return &c, code
}

func getContestsByOwner(t *testing.T, owner, token string) (resp model.PaginatedContestResponse, status int) {
	t.Helper()
	code, respBody := doRequest(t, http.MethodGet, fmt.Sprintf("/contests/owner/%s?page=1&limit=25", owner), token, nil)
	_ = json.Unmarshal(respBody, &resp)
	return resp, code
}

func getContest(t *testing.T, contestID uuid.UUID) (contest *model.Contest, status int) {
	t.Helper()
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws://" + server.Listener.Addr().String() +
		"/ws/contests/" + contestID.String()

	header := http.Header{}
	header.Set("Origin", "http://localhost:3000")
	header.Set("Sec-WebSocket-Protocol", ownerToken)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	defer conn.Close()

	_, msgBytes, err := conn.ReadMessage()
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	var msg model.WSUpdate
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		return nil, http.StatusInternalServerError
	}

	if msg.Contest == nil {
		return nil, http.StatusNotFound
	}

	return msg.Contest, http.StatusOK
}

func updateContest(t *testing.T, contestID uuid.UUID, token string, req model.UpdateContestRequest) (contest model.Contest, status int) {
	t.Helper()
	body, _ := json.Marshal(req)
	code, resp := doRequest(t, http.MethodPatch, fmt.Sprintf("/contests/%s", contestID), token, body)
	var c model.Contest
	_ = json.Unmarshal(resp, &c)
	return c, code
}

func createInvite(t *testing.T, contestID uuid.UUID, token string, req model.CreateInviteRequest) (resp model.ContestInvite, status int) {
	t.Helper()
	body, _ := json.Marshal(req)
	code, respBody := doRequest(t, http.MethodPost, fmt.Sprintf("/contests/%s/invites", contestID), token, body)
	_ = json.Unmarshal(respBody, &resp)
	return resp, code
}

func getInvitePreview(t *testing.T, token string) (resp model.InvitePreviewResponse, status int) {
	t.Helper()
	code, respBody := doRequest(t, http.MethodGet, fmt.Sprintf("/invites/%s", token), "", nil)
	_ = json.Unmarshal(respBody, &resp)
	return resp, code
}

func getInvites(t *testing.T, contestID uuid.UUID, token string) (invites []model.ContestInvite, status int) {
	t.Helper()
	code, resp := doRequest(t, http.MethodGet, fmt.Sprintf("/contests/%s/invites", contestID), token, nil)
	_ = json.Unmarshal(resp, &invites)
	return invites, code
}

func redeemInvite(t *testing.T, inviteToken, authToken string) int {
	t.Helper()
	code, _ := doRequest(t, http.MethodPost, fmt.Sprintf("/invites/%s/redeem", inviteToken), authToken, nil)
	return code
}

func getParticipants(t *testing.T, contestID uuid.UUID, token string) (participants []model.ContestParticipant, status int) {
	t.Helper()
	code, resp := doRequest(t, http.MethodGet, fmt.Sprintf("/contests/%s/participants", contestID), token, nil)
	_ = json.Unmarshal(resp, &participants)
	return participants, code
}

func updateSquare(t *testing.T, contestID, squareID uuid.UUID, token, owner, value string) (square model.Square, status int) {
	t.Helper()
	body, _ := json.Marshal(model.UpdateSquareRequest{Owner: owner, Value: value})
	code, resp := doRequest(t, http.MethodPatch, fmt.Sprintf("/contests/%s/squares/%s", contestID, squareID), token, body)
	_ = json.Unmarshal(resp, &square)
	return square, code
}

func startContest(t *testing.T, contestID uuid.UUID, token string) (contest *model.Contest, status int) {
	t.Helper()
	code, resp := doRequest(t, http.MethodPost, fmt.Sprintf("/contests/%s/start", contestID), token, nil)
	var c model.Contest
	_ = json.Unmarshal(resp, &c)
	return &c, code
}

func submitQuarterResult(t *testing.T, contestID uuid.UUID, req model.QuarterResultRequest) int {
	t.Helper()
	body, _ := json.Marshal(req)
	code, _ := doRequest(t, http.MethodPost, fmt.Sprintf("/contests/%s/quarter-result", contestID), ownerToken, body)
	return code
}

func deleteContest(t *testing.T, contestID uuid.UUID, token string) int {
	t.Helper()
	code, _ := doRequest(t, http.MethodDelete, fmt.Sprintf("/contests/%s", contestID), token, nil)
	return code
}

func doRequest(t *testing.T, method, path, token string, body []byte) (code int, respBody []byte) {
	t.Helper()
	r, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	if token != "" {
		r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	if len(body) > 0 {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w.Code, w.Body.Bytes()
}
