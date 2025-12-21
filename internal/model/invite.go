package model

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// InviteTokenClaims represents the claims in a contest invite token
type InviteTokenClaims struct {
	ContestID   uuid.UUID `json:"contestId"`
	SquareLimit int       `json:"squareLimit"` // 0 = unlimited
	jwt.RegisteredClaims
}

// GenerateInviteLinkRequest is the request to generate a shareable invite link
type GenerateInviteLinkRequest struct {
	SquareLimit int `json:"squareLimit" binding:"min=0" example:"5" description:"Maximum squares user can claim (0 = unlimited)"`
}

// GenerateInviteLinkResponse contains the generated invite token
type GenerateInviteLinkResponse struct {
	Token     string `json:"token" example:"eyJhbGc..." description:"Invite token to append to contest URL"`
	ExpiresAt int64  `json:"expiresAt,omitempty" description:"Unix timestamp when token expires (if set)"`
}
