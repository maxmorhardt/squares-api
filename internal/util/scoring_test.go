package util

import (
	"encoding/json"
	"slices"
	"testing"

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
