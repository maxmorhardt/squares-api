package model

type CreateContestRequest struct {
	Name     string `json:"name" binding:"required"`
	HomeTeam string `json:"homeTeam,omitempty"`
	AwayTeam string `json:"awayTeam,omitempty"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required"`
}
