package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContestParticipant struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID   uuid.UUID `json:"contestId" gorm:"type:uuid;index;not null"`
	Username    string    `json:"username" gorm:"type:varchar(255);index;not null"`
	SquareLimit int       `json:"squareLimit" gorm:"type:int;default:0;not null" description:"Maximum number of squares user can claim (0 = unlimited)"`
	JoinedAt    time.Time `json:"joinedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (cp *ContestParticipant) BeforeCreate(tx *gorm.DB) (err error) {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}

	if cp.JoinedAt.IsZero() {
		cp.JoinedAt = time.Now()
	}

	return
}
