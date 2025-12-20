package model

type CreateContestRequest struct {
	Owner    string `json:"owner" binding:"required" description:"Owner's username" validate:"required"`
	Name     string `json:"name" binding:"required" example:"My Contest" validate:"required,max=20,min=1"`
	HomeTeam string `json:"homeTeam,omitempty" example:"Home Team" validate:"max=20"`
	AwayTeam string `json:"awayTeam,omitempty" example:"Away Team" validate:"max=20"`
}

type UpdateSquareRequest struct {
	Value string `json:"value" binding:"required" example:"MRM" validate:"required,max=3,min=1" description:"Square value (required)"`
	Owner string `json:"owner" binding:"required" example:"username" validate:"required" description:"Square owner (required)"`
}

type ClearSquareRequest struct{}

type UpdateContestRequest struct {
	Name     *string `json:"name,omitempty" example:"Updated Contest Name" validate:"omitempty,max=20,min=1"`
	HomeTeam *string `json:"homeTeam,omitempty" example:"Updated Home Team" validate:"omitempty,max=20"`
	AwayTeam *string `json:"awayTeam,omitempty" example:"Updated Away Team" validate:"omitempty,max=20"`
}

type RecordQuarterResultRequest struct {
	Quarter       int `json:"quarter" binding:"required,min=1,max=4" example:"1" description:"Quarter number (1-4)"`
	HomeTeamScore int `json:"homeTeamScore" binding:"required,min=0" example:"14" description:"Home team score"`
	AwayTeamScore int `json:"awayTeamScore" binding:"required,min=0" example:"7" description:"Away team score"`
}

type PaginatedContestResponse struct {
	Contests    []Contest `json:"contests"`
	Page        int       `json:"page"`
	Limit       int       `json:"limit"`
	Total       int64     `json:"total"`
	TotalPages  int       `json:"totalPages"`
	HasNext     bool      `json:"hasNext"`
	HasPrevious bool      `json:"hasPrevious"`
}
