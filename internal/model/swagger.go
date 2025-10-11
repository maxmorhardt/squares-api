package model

import (
	"time"

	"github.com/google/uuid"
)

type ContestSwagger struct {
	ID        uuid.UUID
	Name      string
	XLabels   []int
	YLabels   []int
	Squares   []Square
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}
