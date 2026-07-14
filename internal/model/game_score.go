package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GameScore struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	GameID    uuid.UUID `json:"gameId" gorm:"type:uuid;index;not null"`
	Quarter   int       `json:"quarter" gorm:"not null"`
	HomeScore int       `json:"homeScore"`
	AwayScore int       `json:"awayScore"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (gs *GameScore) BeforeCreate(tx *gorm.DB) (err error) {
	if gs.ID == uuid.Nil {
		gs.ID = uuid.New()
	}
	return
}
