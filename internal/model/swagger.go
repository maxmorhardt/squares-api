package model

import (
	"time"

	"github.com/google/uuid"
)

type GridSwagger struct {
	ID        uuid.UUID
	Name      string
	XLabels   []int
	YLabels   []int
	Cells     []GridCell
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}