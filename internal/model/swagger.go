package model

import (
	"time"

	"github.com/google/uuid"
)

type ContestSwagger struct {
	ID             uuid.UUID       `json:"id"`
	Name           string          `json:"name"`
	XLabels        []int8          `json:"xLabels"`
	YLabels        []int8          `json:"yLabels"`
	HomeTeam       string          `json:"homeTeam,omitempty"`
	AwayTeam       string          `json:"awayTeam,omitempty"`
	Squares        []Square        `json:"squares"`
	QuarterResults []QuarterResult `json:"quarterResults,omitempty"`
	Owner          string          `json:"owner"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	CreatedBy      string          `json:"createdBy"`
	UpdatedBy      string          `json:"updatedBy"`
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
