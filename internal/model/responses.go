package model

type HealthResponse struct {
	Status string `json:"status" example:"UP"`
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

type ContactResponse struct {
	Message string `json:"message"`
}

type StatsResponse struct {
	ContestsCreatedToday int64 `json:"contestsCreatedToday" example:"5"`
	SquaresClaimedToday  int64 `json:"squaresClaimedToday" example:"42"`
	TotalActiveContests  int64 `json:"totalActiveContests" example:"12"`
}
