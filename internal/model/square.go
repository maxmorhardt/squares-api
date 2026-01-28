package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Square struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID      uuid.UUID `json:"contestId" gorm:"type:uuid;index"`
	Row            int       `json:"row"`
	Col            int       `json:"col"`
	Value          string    `json:"value"`
	Owner          string    `json:"owner"`
	OwnerFirstName string    `json:"ownerFirstName"`
	OwnerLastName  string    `json:"ownerLastName"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	CreatedBy      string    `json:"createdBy"`
	UpdatedBy      string    `json:"updatedBy"`
}

func (s *Square) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		s.CreatedBy = user
		s.UpdatedBy = user
	}

	return
}

func (s *Square) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		s.UpdatedBy = user
	}

	return
}
