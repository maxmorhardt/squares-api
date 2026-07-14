package model

import (
	"strconv"
	"time"
)

type ESPNGame struct {
	ESPNID     string
	HomeTeam   string
	AwayTeam   string
	HomeAbbr   string
	AwayAbbr   string
	GameTime   time.Time
	Week       int
	Season     int
	SeasonType int
	State      string
	Period     int
	Completed  bool
	HomeScore  int
	AwayScore  int
	HomeLine   []int
	AwayLine   []int
}

type QuarterScore struct {
	Quarter int
	Home    int
	Away    int
}

type ScoreboardResponse struct {
	Events []scoreboardEvent `json:"events"`
}

type scoreboardEvent struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Season struct {
		Year int `json:"year"`
		Type int `json:"type"`
	} `json:"season"`
	Week struct {
		Number int `json:"number"`
	} `json:"week"`
	Competitions []struct {
		Status      scoreboardStatus      `json:"status"`
		Competitors []scoreboardTeamScore `json:"competitors"`
	} `json:"competitions"`
}

type scoreboardStatus struct {
	Period int `json:"period"`
	Type   struct {
		State     string `json:"state"`
		Completed bool   `json:"completed"`
	} `json:"type"`
}

type scoreboardTeamScore struct {
	HomeAway string `json:"homeAway"`
	Score    string `json:"score"`
	Team     struct {
		DisplayName  string `json:"displayName"`
		Abbreviation string `json:"abbreviation"`
	} `json:"team"`
	LineScores []struct {
		Value float64 `json:"value"`
	} `json:"linescores"`
}

func (r *ScoreboardResponse) ToGames() []ESPNGame {
	games := make([]ESPNGame, 0, len(r.Events))
	for _, e := range r.Events {
		if len(e.Competitions) == 0 {
			continue
		}
		comp := e.Competitions[0]

		g := ESPNGame{
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

func (e *ESPNGame) ToGame() *Game {
	return &Game{
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

// CompletedQuarters converts ESPN's per-quarter line scores into the cumulative
// running totals squares are scored against. A quarter only counts once play has
// moved past it (or the game is over). Q1-Q3 sum the line scores; Q4 uses the
// final total so it reflects any overtime rather than just the fourth period.
func (e *ESPNGame) CompletedQuarters() []QuarterScore {
	out := make([]QuarterScore, 0, 4)

	homeTotal, awayTotal := 0, 0
	n := min(len(e.HomeLine), len(e.AwayLine))
	for q := 1; q <= 3 && q-1 < n; q++ {
		homeTotal += e.HomeLine[q-1]
		awayTotal += e.AwayLine[q-1]
		if e.Period > q || e.Completed {
			out = append(out, QuarterScore{Quarter: q, Home: homeTotal, Away: awayTotal})
		}
	}

	if e.Completed {
		out = append(out, QuarterScore{Quarter: 4, Home: e.HomeScore, Away: e.AwayScore})
	}

	return out
}

func gameStatusFromState(state string) GameStatus {
	switch state {
	case "in":
		return GameStatusInProgress
	case "post":
		return GameStatusFinal
	default:
		return GameStatusScheduled
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
