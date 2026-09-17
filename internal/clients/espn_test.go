package clients

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

	games, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), nil)
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

	_, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), []string{"20250905"})
	require.Error(t, err)
}

func TestESPNClient_FetchScoreboard_SendsUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(scoreboardBody))
	}))
	defer server.Close()

	_, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), []string{"20260911"})
	require.NoError(t, err)
	assert.Equal(t, userAgent, got)
}

func TestESPNClient_FetchScoreboard_RequestsOneDatePerCall(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Query().Get("dates"))
		_, _ = w.Write([]byte(scoreboardBody))
	}))
	defer server.Close()

	// espn rejects YYYYMMDD-YYYYMMDD ranges, so a window must become one request per day
	_, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), []string{"20260911", "20260912"})
	require.NoError(t, err)
	assert.Equal(t, []string{"20260911", "20260912"}, got)
}

func TestESPNClient_FetchScoreboard_OmitsDatesWhenEmpty(t *testing.T) {
	var hadParam bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hadParam = r.URL.Query()["dates"]
		_, _ = w.Write([]byte(scoreboardBody))
	}))
	defer server.Close()

	// no dates means espn returns the current week, which is all a live poll needs
	_, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), nil)
	require.NoError(t, err)
	assert.False(t, hadParam)
}

func TestESPNClient_FetchScoreboard_DedupesGamesAcrossDates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(scoreboardBody))
	}))
	defer server.Close()

	// a late game lands on both adjacent dates, so the same espn id must not ingest twice
	games, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), []string{"20260911", "20260912"})
	require.NoError(t, err)
	require.Len(t, games, 1)
	assert.Equal(t, "401", games[0].ESPNID)
}

func TestESPNClient_FetchScoreboard_StopsOnFirstDateFailure(t *testing.T) {
	var got []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.URL.Query().Get("dates"))
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := NewESPNClient(server.URL).FetchScoreboard(context.Background(), []string{"20260911", "20260912"})
	require.Error(t, err)

	// the failing date is named so a future espn parameter change is diagnosable from the log
	assert.Contains(t, err.Error(), "20260911")
	assert.NotContains(t, got, "20260912")
}
