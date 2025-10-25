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

type UpdateContestRequest struct {
	Name     *string        `json:"name,omitempty" example:"Updated Contest Name" validate:"omitempty,max=20,min=1"`
	HomeTeam *string        `json:"homeTeam,omitempty" example:"Updated Home Team" validate:"omitempty,max=20"`
	AwayTeam *string        `json:"awayTeam,omitempty" example:"Updated Away Team" validate:"omitempty,max=20"`
	Status   *ContestStatus `json:"status,omitempty" example:"ACTIVE" validate:"omitempty"`
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
