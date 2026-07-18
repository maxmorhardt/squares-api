package util

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabels(t *testing.T) {
	c := startedContest(model.ContestStatusQ1)
	x, y, err := ParseLabels(c)
	require.NoError(t, err)
	assert.Equal(t, identityLabels(), x)
	assert.Equal(t, identityLabels(), y)
}

func identityLabels() []int8 { return []int8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9} }

func startedContest(status model.ContestStatus) *model.Contest {
	labels, _ := json.Marshal(identityLabels())
	squares := make([]model.Square, 0, 100)
	for r := 0; r < 10; r++ {
		for c := 0; c < 10; c++ {
			squares = append(squares, model.Square{Row: r, Col: c, Owner: "u", OwnerName: "U"})
		}
	}
	return &model.Contest{ID: uuid.New(), Status: status, XLabels: labels, YLabels: labels, Squares: squares}
}

func TestComputeWinner(t *testing.T) {
	labels := identityLabels()
	// home 17 -> last digit 7 (col), away 23 -> last digit 3 (row)
	row, col, ok := ComputeWinner(17, 23, labels, labels)
	require.True(t, ok)
	assert.Equal(t, 3, row)
	assert.Equal(t, 7, col)
}

func TestComputeWinner_MissingDigit(t *testing.T) {
	short := []int8{0, 1, 2}
	_, _, ok := ComputeWinner(17, 23, short, short)
	assert.False(t, ok)
}

func TestQuarterResultFor(t *testing.T) {
	c := startedContest(model.ContestStatusQ1)
	r, err := QuarterResultFor(c, 1, 7, 3)
	require.NoError(t, err)
	assert.Equal(t, 1, r.Quarter)
	assert.Equal(t, 7, r.HomeTeamScore)
	assert.Equal(t, "u", r.Winner)
}

func TestSynthesizeFromGame(t *testing.T) {
	c := startedContest(model.ContestStatusQ2)
	c.Game = &model.Game{ID: uuid.New(), Scores: []model.GameScore{
		{Quarter: 2, HomeScore: 14, AwayScore: 10},
		{Quarter: 1, HomeScore: 7, AwayScore: 3},
	}}

	SynthesizeFromGame(c)

	require.Len(t, c.QuarterResults, 2)
	assert.Equal(t, 1, c.QuarterResults[0].Quarter)
	assert.Equal(t, 2, c.QuarterResults[1].Quarter)
}

func TestSynthesizeFromGame_NoGame(t *testing.T) {
	c := startedContest(model.ContestStatusQ1)
	SynthesizeFromGame(c)
	assert.Empty(t, c.QuarterResults)
}

func TestRandomizedLabels(t *testing.T) {
	xJSON, yJSON, err := RandomizedLabels()
	require.NoError(t, err)

	var x, y []int8
	require.NoError(t, json.Unmarshal(xJSON, &x))
	require.NoError(t, json.Unmarshal(yJSON, &y))

	// each axis is a permutation of 0-9
	for _, labels := range [][]int8{x, y} {
		require.Len(t, labels, 10)
		for d := int8(0); d < 10; d++ {
			assert.True(t, slices.Contains(labels, d), "missing digit %d", d)
		}
	}
}

func TestAllSquaresClaimed(t *testing.T) {
	full := startedContest(model.ContestStatusActive)
	assert.True(t, AllSquaresClaimed(full))

	full.Squares[0].Owner = ""
	assert.False(t, AllSquaresClaimed(full))

	assert.False(t, AllSquaresClaimed(&model.Contest{}))
}

const scoreboardJSON = `{
  "events": [
    {
      "id": "401",
      "date": "2026-01-04T18:00Z",
      "season": {"year": 2025, "type": 2},
      "week": {"number": 18},
      "competitions": [
        {
          "status": {"period": 4, "type": {"state": "post", "completed": true}},
          "competitors": [
            {"homeAway": "home", "score": "24", "team": {"displayName": "Chiefs", "abbreviation": "KC"},
             "linescores": [{"value": 7}, {"value": 3}, {"value": 7}, {"value": 7}]},
            {"homeAway": "away", "score": "20", "team": {"displayName": "Eagles", "abbreviation": "PHI"},
             "linescores": [{"value": 3}, {"value": 7}, {"value": 3}, {"value": 7}]}
          ]
        }
      ]
    },
    {"id": "empty", "competitions": []}
  ]
}`

func TestScoreboardToGames(t *testing.T) {
	var resp model.ScoreboardResponse
	require.NoError(t, json.Unmarshal([]byte(scoreboardJSON), &resp))

	games := ScoreboardToGames(&resp)

	// the event with no competitions is skipped
	require.Len(t, games, 1)
	g := games[0]
	assert.Equal(t, "401", g.ESPNID)
	assert.Equal(t, 2025, g.Season)
	assert.Equal(t, 2, g.SeasonType)
	assert.Equal(t, 18, g.Week)
	assert.Equal(t, 4, g.Period)
	assert.True(t, g.Completed)
	assert.Equal(t, "post", g.State)

	assert.Equal(t, "Chiefs", g.HomeTeam)
	assert.Equal(t, "KC", g.HomeAbbr)
	assert.Equal(t, 24, g.HomeScore)
	assert.Equal(t, []int{7, 3, 7, 7}, g.HomeLine)

	assert.Equal(t, "Eagles", g.AwayTeam)
	assert.Equal(t, "PHI", g.AwayAbbr)
	assert.Equal(t, 20, g.AwayScore)
	assert.Equal(t, []int{3, 7, 3, 7}, g.AwayLine)

	assert.False(t, g.GameTime.IsZero())
}

func TestESPNGameToGame(t *testing.T) {
	eg := &model.ESPNGame{
		ESPNID:     "401",
		HomeTeam:   "Chiefs",
		AwayTeam:   "Eagles",
		HomeAbbr:   "KC",
		AwayAbbr:   "PHI",
		GameTime:   time.Date(2026, 1, 4, 18, 0, 0, 0, time.UTC),
		Week:       18,
		Season:     2025,
		SeasonType: 2,
		State:      "in",
		Period:     3,
		HomeScore:  17,
		AwayScore:  13,
	}

	g := ESPNGameToGame(eg)

	assert.Equal(t, "401", g.ESPNID)
	assert.Equal(t, "Chiefs", g.HomeTeam)
	assert.Equal(t, "PHI", g.AwayAbbr)
	assert.Equal(t, 18, g.Week)
	assert.Equal(t, model.GameStatusInProgress, g.Status)
	assert.Equal(t, 3, g.Period)
	assert.Equal(t, 17, g.HomeScore)
	assert.Equal(t, 13, g.AwayScore)
}

func TestCompletedQuarters_Completed(t *testing.T) {
	eg := &model.ESPNGame{
		Period:    4,
		Completed: true,
		HomeScore: 24,
		AwayScore: 20,
		HomeLine:  []int{7, 3, 7, 7},
		AwayLine:  []int{3, 7, 3, 7},
	}

	quarters := CompletedQuarters(eg)

	// q1-q3 use cumulative line scores; q4 uses the final score (covers overtime)
	require.Len(t, quarters, 4)
	assert.Equal(t, model.QuarterScore{Quarter: 1, Home: 7, Away: 3}, quarters[0])
	assert.Equal(t, model.QuarterScore{Quarter: 2, Home: 10, Away: 10}, quarters[1])
	assert.Equal(t, model.QuarterScore{Quarter: 3, Home: 17, Away: 13}, quarters[2])
	assert.Equal(t, model.QuarterScore{Quarter: 4, Home: 24, Away: 20}, quarters[3])
}

func TestCompletedQuarters_InProgress(t *testing.T) {
	// mid-third-quarter: only quarters play has moved past are counted
	eg := &model.ESPNGame{
		Period:   3,
		HomeLine: []int{7, 3, 7},
		AwayLine: []int{3, 7, 0},
	}

	quarters := CompletedQuarters(eg)

	require.Len(t, quarters, 2)
	assert.Equal(t, model.QuarterScore{Quarter: 1, Home: 7, Away: 3}, quarters[0])
	assert.Equal(t, model.QuarterScore{Quarter: 2, Home: 10, Away: 10}, quarters[1])
}

func TestCompletedQuarters_Scheduled(t *testing.T) {
	assert.Empty(t, CompletedQuarters(&model.ESPNGame{}))
}

func TestGameStatusFromState(t *testing.T) {
	assert.Equal(t, model.GameStatusInProgress, gameStatusFromState("in"))
	assert.Equal(t, model.GameStatusFinal, gameStatusFromState("post"))
	assert.Equal(t, model.GameStatusScheduled, gameStatusFromState("pre"))
	assert.Equal(t, model.GameStatusScheduled, gameStatusFromState(""))
}

func TestParseESPNTime(t *testing.T) {
	rfc := parseESPNTime("2026-01-04T18:00:00Z")
	assert.Equal(t, time.Date(2026, 1, 4, 18, 0, 0, 0, time.UTC), rfc.UTC())

	// espn also emits a minute-precision layout without seconds
	short := parseESPNTime("2026-01-04T18:00Z")
	assert.Equal(t, time.Date(2026, 1, 4, 18, 0, 0, 0, time.UTC), short.UTC())

	assert.True(t, parseESPNTime("not-a-time").IsZero())
}
