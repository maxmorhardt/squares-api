package model

import (
	"time"

	"github.com/google/uuid"
)

type ContestSwagger struct {
	ID             uuid.UUID       `json:"id" description:"Unique contest identifier"`
	Name           string          `json:"name" example:"Super Bowl 2025" description:"Contest name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	XLabels        []int8          `json:"xLabels" description:"X-axis labels (0-9 randomized when transitioning to Q1)"`
	YLabels        []int8          `json:"yLabels" description:"Y-axis labels (0-9 randomized when transitioning to Q1)"`
	HomeTeam       string          `json:"homeTeam,omitempty" example:"Chiefs" description:"Home team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	AwayTeam       string          `json:"awayTeam,omitempty" example:"49ers" description:"Away team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	Squares        []Square        `json:"squares" description:"100 squares in 10x10 grid"`
	QuarterResults []QuarterResult `json:"quarterResults,omitempty" description:"Results for each quarter (Q1-Q4)"`
	Owner          string          `json:"owner" description:"Username of the contest owner"`
	Status         string          `json:"status" example:"ACTIVE" enums:"ACTIVE,Q1,Q2,Q3,Q4,FINISHED,DELETED" description:"Contest status (ACTIVE→Q1→Q2→Q3→Q4→FINISHED, or DELETED at any time)"`
	CreatedAt      time.Time       `json:"createdAt" description:"When the contest was created"`
	UpdatedAt      time.Time       `json:"updatedAt" description:"When the contest was last updated"`
	CreatedBy      string          `json:"createdBy" description:"Username who created the contest"`
	UpdatedBy      string          `json:"updatedBy" description:"Username who last updated the contest"`
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
