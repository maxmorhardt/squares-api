package model

type CreateContestRequest struct {
	Name     string `json:"name" binding:"required" example:"My Contest" validate:"required,max=20,min=1" description:"Contest name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	HomeTeam string `json:"homeTeam,omitempty" example:"Home Team" validate:"max=20" description:"Home team name (optional, 1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	AwayTeam string `json:"awayTeam,omitempty" example:"Away Team" validate:"max=20" description:"Away team name (optional, 1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" example:"MRM" validate:"max=3" description:"Square value (1-3 uppercase letters/numbers only)"`
}
