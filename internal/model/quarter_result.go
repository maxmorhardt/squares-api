package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type QuarterResult struct {
	ID              uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID       uuid.UUID `json:"contestId" gorm:"type:uuid;index;not null"`
	Quarter         int       `json:"quarter" gorm:"type:int;not null" example:"1" description:"Quarter number (1, 2, 3, or 4)"`
	HomeTeamScore   int       `json:"homeTeamScore" example:"14" description:"Home team score at end of quarter"`
	AwayTeamScore   int       `json:"awayTeamScore" example:"7" description:"Away team score at end of quarter"`
	WinnerRow       int       `json:"winnerRow" example:"4" description:"Row of the winning square (0-9)"`
	WinnerCol       int       `json:"winnerCol" example:"7" description:"Column of the winning square (0-9)"`
	Winner          string    `json:"winner" description:"Username of the winner"`
	WinnerFirstName string    `json:"winnerFirstName" description:"First name of the winner"`
	WinnerLastName  string    `json:"winnerLastName" description:"Last name of the winner"`
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
