package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GameStatus string

const (
	GameStatusScheduled  GameStatus = "scheduled"
	GameStatusInProgress GameStatus = "in_progress"
	GameStatusFinal      GameStatus = "final"
)

type Game struct {
	ID         uuid.UUID   `json:"id" gorm:"type:uuid;primaryKey"`
	ESPNID     string      `json:"espnId" gorm:"column:espn_id;uniqueIndex;not null"`
	HomeTeam   string      `json:"homeTeam"`
	AwayTeam   string      `json:"awayTeam"`
	HomeAbbr   string      `json:"homeAbbr"`
	AwayAbbr   string      `json:"awayAbbr"`
	GameTime   time.Time   `json:"gameTime"`
	Week       int         `json:"week"`
	Season     int         `json:"season"`
	SeasonType int         `json:"seasonType" gorm:"column:season_type"`
	Status     GameStatus  `json:"status" gorm:"not null;default:scheduled"`
	Period     int         `json:"period"`
	HomeScore  int         `json:"homeScore"`
	AwayScore  int         `json:"awayScore"`
	Scores     []GameScore `json:"scores,omitempty" gorm:"foreignKey:GameID;constraint:OnDelete:CASCADE"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

func (g *Game) BeforeCreate(tx *gorm.DB) (err error) {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return
}
