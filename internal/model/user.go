package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const GhostUser = "ghost"

type User struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Email       string    `json:"email" gorm:"not null;uniqueIndex:idx_users_email"`
	DisplayName string    `json:"displayName" gorm:"not null;default:''"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	return nil
}
