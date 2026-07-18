package model

import "time"

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

type GameActivity struct {
	Live        bool
	NextKickoff time.Time
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
