package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuarterResult struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID       uuid.UUID `json:"contestId" gorm:"type:uuid;index;not null"`
	Quarter         int       `json:"quarter" gorm:"type:int;not null"`
	HomeTeamScore   int       `json:"homeTeamScore"`
	AwayTeamScore   int       `json:"awayTeamScore"`
	WinnerRow       int       `json:"winnerRow"`
	WinnerCol       int       `json:"winnerCol"`
	Winner          string    `json:"winner"`
	WinnerName      string    `json:"winnerName"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CreatedBy       string    `json:"createdBy"`
	UpdatedBy       string    `json:"updatedBy"`
}

func (q *QuarterResult) BeforeCreate(tx *gorm.DB) (err error) {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		q.CreatedBy = user
		q.UpdatedBy = user
	}

	return
}

func (q *QuarterResult) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		q.UpdatedBy = user
	}

	return
}
