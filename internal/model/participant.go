package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ParticipantRole string

const (
	ParticipantRoleOwner       ParticipantRole = "owner"
	ParticipantRoleParticipant ParticipantRole = "participant"
	ParticipantRoleViewer      ParticipantRole = "viewer"
)

type ContestParticipant struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID  uuid.UUID       `json:"contestId" gorm:"type:uuid;index;not null"`
	UserID     string          `json:"userId" gorm:"not null;index"`
	Role       ParticipantRole `json:"role" gorm:"not null"`
	MaxSquares int             `json:"maxSquares" gorm:"not null;default:0"`
	InviteID   *uuid.UUID      `json:"inviteId,omitempty" gorm:"type:uuid"`
	JoinedAt   time.Time       `json:"joinedAt"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func (p *ContestParticipant) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now()
	}
	return
}
