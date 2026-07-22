package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeaderboard_PublicAccess(t *testing.T) {
	code, body := doRequest(t, http.MethodGet, "/leaderboard", "", nil)

	require.Equal(t, http.StatusOK, code)

	var resp model.LeaderboardResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.NotNil(t, resp.Entries)

	// invariants that must hold no matter what other tests have written
	for i, entry := range resp.Entries {
		assert.GreaterOrEqual(t, entry.Rank, 1)
		assert.GreaterOrEqual(t, entry.QuarterWins, int64(1))
		assert.NotEmpty(t, entry.DisplayName)
		assert.NotContains(t, entry.DisplayName, "@", "display name must not leak an email")

		// you cannot win more quarters than you played, so the win rate can never exceed 100%.
		// catches a denominator computed too narrowly; over-counting would only understate the rate
		assert.LessOrEqual(t, entry.QuarterWins, entry.QuartersPlayed,
			"%s won more quarters than they played", entry.DisplayName)

		if i > 0 {
			assert.LessOrEqual(t, entry.QuarterWins, resp.Entries[i-1].QuarterWins)
			assert.GreaterOrEqual(t, entry.Rank, resp.Entries[i-1].Rank)
		}
	}
}

func TestLeaderboard_RejectsInvalidLimit(t *testing.T) {
	for _, limit := range []string{"abc", "0", "-1", "101"} {
		code, _ := doRequest(t, http.MethodGet, "/leaderboard?limit="+limit, "", nil)
		assert.Equal(t, http.StatusBadRequest, code, "limit=%s should be rejected", limit)
	}
}

func TestLeaderboard_AcceptsValidLimit(t *testing.T) {
	code, body := doRequest(t, http.MethodGet, "/leaderboard?limit=5", "", nil)

	require.Equal(t, http.StatusOK, code)

	var resp model.LeaderboardResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	assert.LessOrEqual(t, len(resp.Entries), 5)
}

func TestLeaderboard_MyRankRequiresAuth(t *testing.T) {
	code, _ := doRequest(t, http.MethodGet, "/leaderboard/me", "", nil)

	assert.Equal(t, http.StatusUnauthorized, code)
}

func TestLeaderboard_TotalRankedMatchesTheVisibleBoard(t *testing.T) {
	// the board is well under the limit, so every ranked player must be on it. a winner counted
	// in totalRanked but missing from the board is what produced a bogus "#2 of 2" for one entry
	code, body := doRequest(t, http.MethodGet, "/leaderboard?limit=100", "", nil)
	require.Equal(t, http.StatusOK, code)

	var board model.LeaderboardResponse
	require.NoError(t, json.Unmarshal(body, &board))

	code, body = doRequest(t, http.MethodGet, "/leaderboard/me", ownerToken, nil)
	require.Equal(t, http.StatusOK, code)

	var rank model.LeaderboardRankResponse
	require.NoError(t, json.Unmarshal(body, &rank))

	assert.Equal(t, int64(len(board.Entries)), rank.TotalRanked)
	assert.LessOrEqual(t, rank.Rank, len(board.Entries))
}

func TestLeaderboard_MyRankUnrankedUser(t *testing.T) {
	// a fresh identity keeps this independent of wins other tests create
	token := mintToken("no-wins@squares.test")

	code, body := doRequest(t, http.MethodGet, "/leaderboard/me", token, nil)

	require.Equal(t, http.StatusOK, code)

	var resp model.LeaderboardRankResponse
	require.NoError(t, json.Unmarshal(body, &resp))
	// a user who has never won a quarter must not be reported as rank 1
	assert.Equal(t, 0, resp.Rank)
	assert.False(t, resp.Ranked)
	assert.Equal(t, int64(0), resp.QuarterWins)
}
