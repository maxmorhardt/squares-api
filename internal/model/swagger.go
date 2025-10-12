package model

import (
	"time"

	"github.com/google/uuid"
)

type ContestSwagger struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name" example:"My Contest" description:"Contest name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	XLabels   []int     `json:"xLabels" description:"X-axis labels (0-9 randomized)"`
	YLabels   []int     `json:"yLabels" description:"Y-axis labels (0-9 randomized)"`
	HomeTeam  string    `json:"homeTeam" example:"Home Team" description:"Home team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	AwayTeam  string    `json:"awayTeam" example:"Away Team" description:"Away team name (1-20 characters, letters, numbers, spaces, hyphens, underscores only)"`
	Squares   []Square  `json:"squares" description:"100 squares in 10x10 grid"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedBy string    `json:"updatedBy"`
}
