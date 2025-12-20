package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Contest struct {
	ID             uuid.UUID       `json:"id" gorm:"primaryKey"`
	Name           string          `json:"name"`
	XLabels        datatypes.JSON  `json:"xLabels"`
	YLabels        datatypes.JSON  `json:"yLabels"`
	HomeTeam       string          `json:"homeTeam,omitempty"`
	AwayTeam       string          `json:"awayTeam,omitempty"`
	Squares        []Square        `json:"squares" gorm:"foreignKey:ContestID;constraint:OnDelete:CASCADE"`
	QuarterResults []QuarterResult `json:"quarterResults,omitempty" gorm:"foreignKey:ContestID;constraint:OnDelete:CASCADE"`
	Owner          string          `json:"owner"`
	Status         ContestStatus   `json:"status"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	CreatedBy      string          `json:"createdBy"`
	UpdatedBy      string          `json:"updatedBy"`
}

func (c *Contest) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		c.CreatedBy = user
		c.UpdatedBy = user
	}

	return
}

func (c *Contest) BeforeUpdate(tx *gorm.DB) (err error) {
	if user, ok := tx.Statement.Context.Value(UserKey).(string); ok {
		c.UpdatedBy = user
	}

	return
}
