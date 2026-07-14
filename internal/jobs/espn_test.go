package jobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scoreboardBody = `{
  "events": [{
    "id": "401",
    "date": "2025-09-05T00:20Z",
    "season": {"year": 2025, "type": 2},
    "week": {"number": 1},
    "competitions": [{
      "status": {"period": 2, "type": {"state": "in", "completed": false}},
      "competitors": [
        {"homeAway": "home", "score": "10", "team": {"displayName": "Chiefs", "abbreviation": "KC"}, "linescores": [{"value": 7}, {"value": 3}]},
        {"homeAway": "away", "score": "7", "team": {"displayName": "Eagles", "abbreviation": "PHI"}, "linescores": [{"value": 0}, {"value": 7}]}
      ]
    }]
  }]
}`

func TestESPNClient_FetchScoreboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(scoreboardBody))
	}))
	defer server.Close()

	games, err := newESPNClient(server.URL).FetchScoreboard(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, games, 1)

	g := games[0]
	assert.Equal(t, "401", g.ESPNID)
	assert.Equal(t, "Chiefs", g.HomeTeam)
	assert.Equal(t, "Eagles", g.AwayTeam)
	assert.Equal(t, 10, g.HomeScore)
	assert.Equal(t, "in", g.State)
	assert.Equal(t, 2, g.Period)
}

func TestESPNClient_FetchScoreboard_BadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := newESPNClient(server.URL).FetchScoreboard(context.Background(), "2025")
	require.Error(t, err)
}
