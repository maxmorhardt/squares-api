package model

import "github.com/google/uuid"

type LivenessResponse struct {
	Status string `json:"status"`
}

type ReadinessResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Nats     string `json:"nats"`
	OIDC     string `json:"oidc"`
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

type LeaderboardEntry struct {
	Rank           int    `json:"rank" example:"1"`
	DisplayName    string `json:"displayName" example:"Max"`
	QuarterWins    int64  `json:"quarterWins" example:"12"`
	SquaresClaimed int64  `json:"squaresClaimed" example:"48"`
	QuartersPlayed int64  `json:"quartersPlayed" example:"40"`
}

type LeaderboardResponse struct {
	Entries []LeaderboardEntry `json:"entries"`
}

type LeaderboardRankResponse struct {
	Rank        int   `json:"rank" example:"7"`
	TotalRanked int64 `json:"totalRanked" example:"143"`
	QuarterWins int64 `json:"quarterWins" example:"5"`
	Ranked      bool  `json:"ranked" example:"true"`
}

type UserProfileResponse struct {
	Email           string `json:"email" example:"user@example.com"`
	DisplayName     string `json:"displayName" example:"Max"`
	DefaultInitials string `json:"defaultInitials" example:"MM"`
	CreatedAt       string `json:"createdAt" example:"2026-07-11T00:00:00Z"`
}

type UserActiveContest struct {
	ID    string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name  string `json:"name" example:"test"`
	Owner string `json:"owner" example:"user@example.com"`
	Role  string `json:"role" example:"owner"`
}

type UserStatsResponse struct {
	ContestsCreated int64 `json:"contestsCreated" example:"3"`
	ContestsJoined  int64 `json:"contestsJoined" example:"7"`
	SquaresClaimed  int64 `json:"squaresClaimed" example:"42"`
	QuarterWins     int64 `json:"quarterWins" example:"5"`
	QuartersPlayed  int64 `json:"quartersPlayed" example:"20"`
}

type InvitePreviewResponse struct {
	ContestID   uuid.UUID `json:"contestId"`
	ContestName string    `json:"contestName"`
	Owner       string    `json:"owner"`
	Role        string    `json:"role"`
	MaxSquares  int       `json:"maxSquares"`
}
