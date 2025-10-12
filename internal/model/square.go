package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Square struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID uuid.UUID `json:"contestId" gorm:"type:uuid;index"`
	Row       int       `json:"row" example:"0" description:"Square row position (0-9)"`
	Col       int       `json:"col" example:"0" description:"Square column position (0-9)"`
	Value     string    `json:"value" example:"MRM" description:"Square value (1-3 uppercase letters/numbers only)"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Owner     string    `json:"owner"`
}

func (s *Square) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	return
}
