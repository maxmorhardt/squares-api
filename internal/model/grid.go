package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)


type Grid struct {
	ID        uuid.UUID      `gorm:"primaryKey"`
	Name      string
	Data      datatypes.JSON
	XLabels   datatypes.JSON
	YLabels   datatypes.JSON
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy string
	UpdatedBy string
}

type GridSwagger struct {
	ID        uuid.UUID
	Name      string
	Data      [][]string
	XLabels   []int8
	YLabels   []int8
	CreatedAt string
	UpdatedAt string
	CreatedBy string
	UpdatedBy string
}

func (g *Grid) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(ContextUserKey).(string); ok {
		g.CreatedBy = user
		g.UpdatedBy = user
	}

	return
}

func (g *Grid) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(ContextUserKey).(string); ok {
		g.UpdatedBy = user
	}

	return
}