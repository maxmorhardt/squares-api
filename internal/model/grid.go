package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Grid struct {
	ID        uuid.UUID      `gorm:"primaryKey" json:"id"`
	Name      string         `json:"name"`
	Data      datatypes.JSON `json:"data"`
	XLabels   datatypes.JSON `json:"xLabels"`
	YLabels   datatypes.JSON `json:"yLabels"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy string         `json:"createdBy"`
	UpdatedBy string         `json:"updatedBy"`
}

type GridSwagger struct {
	ID        uuid.UUID
	Name      string
	Data      [][]string
	XLabels   []int8
	YLabels   []int8
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}

type CreateGridRequest struct {
	Name string `json:"name" binding:"required"`
}

func (g *Grid) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		g.CreatedBy = user
		g.UpdatedBy = user
	}

	return
}

func (g *Grid) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		g.UpdatedBy = user
	}

	return
}
