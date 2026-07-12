package model

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

type UserProfileResponse struct {
	Email       string `json:"email" example:"user@example.com"`
	DisplayName string `json:"displayName" example:"Max"`
	CreatedAt   string `json:"createdAt" example:"2026-07-11T00:00:00Z"`
}

type UserStatsResponse struct {
	ContestsCreated int64 `json:"contestsCreated" example:"3"`
	ContestsJoined  int64 `json:"contestsJoined" example:"7"`
	SquaresClaimed  int64 `json:"squaresClaimed" example:"42"`
	QuarterWins     int64 `json:"quarterWins" example:"5"`
}

type InvitePreviewResponse struct {
	ContestName string `json:"contestName"`
	Owner       string `json:"owner"`
	Role        string `json:"role"`
	MaxSquares  int    `json:"maxSquares"`
}

type InviteResponse struct {
	InviteURL string `json:"inviteUrl"`
	Token     string `json:"token"`
}
