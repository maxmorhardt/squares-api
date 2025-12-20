package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContactSubmission struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name      string    `json:"name" gorm:"type:varchar(100);not null"`
	Email     string    `json:"email" gorm:"type:varchar(255);not null"`
	Subject   string    `json:"subject" gorm:"type:varchar(200);not null"`
	Message   string    `json:"message" gorm:"type:text;not null"`
	IPAddress string    `json:"ipAddress" gorm:"type:varchar(45)"`
	Status    string    `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Response  string    `json:"response" gorm:"type:text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (cs *ContactSubmission) BeforeCreate(tx *gorm.DB) (err error) {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}

	if cs.Status == "" {
		cs.Status = "pending"
	}

	return
}
