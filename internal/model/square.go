package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Square struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID      uuid.UUID `json:"contestId" gorm:"type:uuid;index"`
	Row            int       `json:"row" example:"0" description:"Square row position (0-9)"`
	Col            int       `json:"col" example:"0" description:"Square column position (0-9)"`
	Value          string    `json:"value" example:"MRM" description:"Square value (1-3 uppercase letters/numbers only)"`
	Owner          string    `json:"owner" description:"Owner's username"`
	OwnerFirstName string    `json:"ownerFirstName" description:"Owner's first name"`
	OwnerLastName  string    `json:"ownerLastName" description:"Owner's last name"`
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
