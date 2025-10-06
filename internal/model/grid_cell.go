package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GridCell struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	GridID    uuid.UUID `json:"gridId" gorm:"type:uuid;index"`
	Row       int       `json:"row"`
	Col       int       `json:"col"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedBy string    `json:"updatedBy"`
}

func (gc *GridCell) BeforeCreate(tx *gorm.DB) (err error) {
	if gc.ID == uuid.Nil {
		gc.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		gc.CreatedBy = user
		gc.UpdatedBy = user
	}

	return
}

func (gc *GridCell) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		gc.UpdatedBy = user
	}
	return
}