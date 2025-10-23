package model

import (
	"time"

	"github.com/google/uuid"
)

type ContestSwagger struct {
	ID        uuid.UUID `json:"id" description:"Unique contest identifier"`
	Name      string    `json:"name" example:"My Contest" description:"Contest name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	XLabels   []int8    `json:"xLabels" description:"X-axis labels (0-9 randomized)"`
	YLabels   []int8    `json:"yLabels" description:"Y-axis labels (0-9 randomized)"`
	HomeTeam  string    `json:"homeTeam,omitempty" example:"Home Team" description:"Home team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	AwayTeam  string    `json:"awayTeam,omitempty" example:"Away Team" description:"Away team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	Squares   []Square  `json:"squares" description:"100 squares in 10x10 grid"`
	Owner     string    `json:"owner" description:"Username of the contest owner"`
	Status    string    `json:"status" example:"ACTIVE" description:"Contest status"`
	CreatedAt time.Time `json:"createdAt" description:"When the contest was created"`
	UpdatedAt time.Time `json:"updatedAt" description:"When the contest was last updated"`
	CreatedBy string    `json:"createdBy" description:"Username who created the contest"`
	UpdatedBy string    `json:"updatedBy" description:"Username who last updated the contest"`
}

type PaginatedContestResponseSwagger struct {
	Contests    []ContestSwagger `json:"contests"`
	Page        int              `json:"page"`
	Limit       int              `json:"limit"`
	Total       int64            `json:"total"`
	TotalPages  int              `json:"totalPages"`
	HasNext     bool             `json:"hasNext"`
	HasPrevious bool             `json:"hasPrevious"`
}
