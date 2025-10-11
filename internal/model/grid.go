package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Grid struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name"`
	XLabels   datatypes.JSON `json:"xLabels"`
	YLabels   datatypes.JSON `json:"yLabels"`
	Cells     []GridCell     `json:"cells" gorm:"foreignKey:GridID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy string         `json:"createdBy"`
	UpdatedBy string         `json:"updatedBy"`
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