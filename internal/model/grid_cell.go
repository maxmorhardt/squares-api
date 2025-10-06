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
	Owner     string    `json:"owner"`
}

func (gc *GridCell) BeforeCreate(tx *gorm.DB) (err error) {
	if gc.ID == uuid.Nil {
		gc.ID = uuid.New()
	}

	return
}