package model

type CreateContestRequest struct {
	Owner    string `json:"owner" binding:"required" description:"Owner's username" validate:"required"`
	Name     string `json:"name" binding:"required" example:"My Contest" validate:"required,max=20,min=1"`
	HomeTeam string `json:"homeTeam,omitempty" example:"Home Team" validate:"max=20"`
	AwayTeam string `json:"awayTeam,omitempty" example:"Away Team" validate:"max=20"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" example:"MRM" validate:"max=3" description:"Square value"`
}
