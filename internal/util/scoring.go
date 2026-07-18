package util

import (
	cryptorand "crypto/rand"
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"time"

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

func ScoreboardToGames(r *model.ScoreboardResponse) []model.ESPNGame {
	games := make([]model.ESPNGame, 0, len(r.Events))
	for _, e := range r.Events {
		if len(e.Competitions) == 0 {
			continue
		}
		comp := e.Competitions[0]

		g := model.ESPNGame{
			ESPNID:     e.ID,
			GameTime:   parseESPNTime(e.Date),
			Week:       e.Week.Number,
			Season:     e.Season.Year,
			SeasonType: e.Season.Type,
			State:      comp.Status.Type.State,
			Period:     comp.Status.Period,
			Completed:  comp.Status.Type.Completed,
		}

		for _, c := range comp.Competitors {
			score, _ := strconv.Atoi(c.Score)
			line := make([]int, 0, len(c.LineScores))
			for _, ls := range c.LineScores {
				line = append(line, int(ls.Value))
			}

			if c.HomeAway == "home" {
				g.HomeTeam = c.Team.DisplayName
				g.HomeAbbr = c.Team.Abbreviation
				g.HomeScore = score
				g.HomeLine = line
			} else {
				g.AwayTeam = c.Team.DisplayName
				g.AwayAbbr = c.Team.Abbreviation
				g.AwayScore = score
				g.AwayLine = line
			}
		}

		games = append(games, g)
	}

	return games
}

func ESPNGameToGame(e *model.ESPNGame) *model.Game {
	return &model.Game{
		ESPNID:     e.ESPNID,
		HomeTeam:   e.HomeTeam,
		AwayTeam:   e.AwayTeam,
		HomeAbbr:   e.HomeAbbr,
		AwayAbbr:   e.AwayAbbr,
		GameTime:   e.GameTime,
		Week:       e.Week,
		Season:     e.Season,
		SeasonType: e.SeasonType,
		Status:     gameStatusFromState(e.State),
		Period:     e.Period,
		HomeScore:  e.HomeScore,
		AwayScore:  e.AwayScore,
	}
}

func CompletedQuarters(e *model.ESPNGame) []model.QuarterScore {
	out := make([]model.QuarterScore, 0, 4)

	// accumulate the running totals squares are scored against
	homeTotal, awayTotal := 0, 0
	n := min(len(e.HomeLine), len(e.AwayLine))
	for q := 1; q <= 3 && q-1 < n; q++ {
		// sum each quarter's line score into the cumulative total
		homeTotal += e.HomeLine[q-1]
		awayTotal += e.AwayLine[q-1]
		// only count a quarter once play has moved past it or the game is over
		if e.Period > q || e.Completed {
			out = append(out, model.QuarterScore{Quarter: q, Home: homeTotal, Away: awayTotal})
		}
	}

	// use the final score for Q4 so it reflects any overtime, not just the fourth period
	if e.Completed {
		out = append(out, model.QuarterScore{Quarter: 4, Home: e.HomeScore, Away: e.AwayScore})
	}

	return out
}

func gameStatusFromState(state string) model.GameStatus {
	switch state {
	case "in":
		return model.GameStatusInProgress
	case "post":
		return model.GameStatusFinal
	default:
		return model.GameStatusScheduled
	}
}

func parseESPNTime(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
