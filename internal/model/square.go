package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Square struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID uuid.UUID `json:"contestId" gorm:"type:uuid;index"`
	Row       int       `json:"row"`
	Col       int       `json:"col"`
	Value     string    `json:"value"`
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