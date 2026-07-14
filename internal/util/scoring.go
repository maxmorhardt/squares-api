package util

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"math/big"
	"sort"

	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func RandomizedLabels() (xLabels, yLabels []byte, err error) {
	x, err := shuffledDigits()
	if err != nil {
		return nil, nil, err
	}
	y, err := shuffledDigits()
	if err != nil {
		return nil, nil, err
	}

	if xLabels, err = json.Marshal(x); err != nil {
		return nil, nil, err
	}
	if yLabels, err = json.Marshal(y); err != nil {
		return nil, nil, err
	}
	return xLabels, yLabels, nil
}

func InitialLabels() (xLabels, yLabels []byte) {
	labels := make([]int8, 10)
	for i := range labels {
		labels[i] = -1
	}

	// marshaling a fixed-size []int8 cannot fail, so the errors are unreachable
	xLabels, _ = json.Marshal(labels)
	yLabels, _ = json.Marshal(labels)
	return xLabels, yLabels
}

func shuffledDigits() ([]int8, error) {
	labels := make([]int8, 10)
	for i := range labels {
		labels[i] = int8(i)
	}

	// fisher-yates shuffle with a cryptographic source
	for i := len(labels) - 1; i > 0; i-- {
		n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return nil, err
		}
		j := int(n.Int64())
		labels[i], labels[j] = labels[j], labels[i]
	}
	return labels, nil
}

func AllSquaresClaimed(c *model.Contest) bool {
	if len(c.Squares) == 0 {
		return false
	}
	for i := range c.Squares {
		if c.Squares[i].Owner == "" {
			return false
		}
	}
	return true
}

func ParseLabels(c *model.Contest) (xLabels, yLabels []int8, err error) {
	if err := json.Unmarshal(c.XLabels, &xLabels); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(c.YLabels, &yLabels); err != nil {
		return nil, nil, err
	}
	return xLabels, yLabels, nil
}

func ComputeWinner(homeScore, awayScore int, xLabels, yLabels []int8) (row, col int, ok bool) {
	homeLastDigit := homeScore % 10
	awayLastDigit := awayScore % 10

	row, col = -1, -1
	for i, label := range yLabels {
		if int(label) == awayLastDigit {
			row = i
			break
		}
	}
	for i, label := range xLabels {
		if int(label) == homeLastDigit {
			col = i
			break
		}
	}

	if row == -1 || col == -1 {
		return 0, 0, false
	}
	return row, col, true
}

func winnerOwner(c *model.Contest, row, col int) (owner, ownerName string) {
	for i := range c.Squares {
		if c.Squares[i].Row == row && c.Squares[i].Col == col {
			return c.Squares[i].Owner, c.Squares[i].OwnerName
		}
	}
	return "", ""
}

func QuarterResultFor(c *model.Contest, quarter, homeScore, awayScore int) (*model.QuarterResult, error) {
	xLabels, yLabels, err := ParseLabels(c)
	if err != nil {
		return nil, err
	}

	row, col, ok := ComputeWinner(homeScore, awayScore, xLabels, yLabels)
	if !ok {
		return nil, errs.ErrWinnerNotDeterminable
	}

	owner, ownerName := winnerOwner(c, row, col)
	return &model.QuarterResult{
		ContestID:     c.ID,
		Quarter:       quarter,
		HomeTeamScore: homeScore,
		AwayTeamScore: awayScore,
		WinnerRow:     row,
		WinnerCol:     col,
		Winner:        owner,
		WinnerName:    ownerName,
	}, nil
}

func SynthesizeFromGame(c *model.Contest) {
	if c.Game == nil {
		return
	}

	results := make([]model.QuarterResult, 0, len(c.Game.Scores))
	for _, s := range c.Game.Scores {
		r, err := QuarterResultFor(c, s.Quarter, s.HomeScore, s.AwayScore)
		if err != nil {
			continue
		}
		results = append(results, *r)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Quarter < results[j].Quarter })
	c.QuarterResults = results
}
