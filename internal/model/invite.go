package model

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContestInvite struct {
	ID         uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	ContestID  uuid.UUID       `json:"contestId" gorm:"type:uuid;index;not null"`
	Token      string          `json:"token" gorm:"uniqueIndex;not null"`
	MaxSquares int             `json:"maxSquares" gorm:"not null"`
	Role       ParticipantRole `json:"role" gorm:"not null;default:participant"`
	CreatedBy  string          `json:"createdBy" gorm:"not null"`
	ExpiresAt  *time.Time      `json:"expiresAt,omitempty"`
	MaxUses    int             `json:"maxUses" gorm:"not null;default:0"`
	Uses       int             `json:"uses" gorm:"not null;default:0"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

func (i *ContestInvite) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Token == "" {
		i.Token, err = generateToken()
	}
	return
}

func (i *ContestInvite) IsExpired() bool {
	if i.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*i.ExpiresAt)
}

func (i *ContestInvite) HasUsesRemaining() bool {
	if i.MaxUses == 0 {
		return true
	}
	return i.Uses < i.MaxUses
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
