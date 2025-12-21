package service

import (
	"context"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type AuthService interface {
	IsDeclaredUser(ctx context.Context, user string) bool
	IsInGroup(ctx context.Context, group string) bool
	GenerateInviteToken(contestID uuid.UUID, squareLimit int) (string, error)
	ValidateInviteToken(tokenString string) (*model.InviteTokenClaims, error)
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) IsDeclaredUser(ctx context.Context, user string) bool {
	ctxUser := ctx.Value(model.UserKey).(string)
	return ctxUser == user
}

func (s *authService) IsInGroup(ctx context.Context, group string) bool {
	claims := util.ClaimsFromContext(ctx)
	if claims == nil {
		return false
	}

	return slices.Contains(claims.Groups, group)
}

func (s *authService) GenerateInviteToken(contestID uuid.UUID, squareLimit int) (string, error) {
	// create claims with contest ID and square limit
	claims := model.InviteTokenClaims{
		ContestID:   contestID,
		SquareLimit: squareLimit,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(90 * 24 * time.Hour)), // 90 days
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "squares-api",
		},
	}

	// create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// get JWT secret from config
	secret := config.GetJWTSecret()

	return token.SignedString([]byte(secret))
}

func (s *authService) ValidateInviteToken(tokenString string) (*model.InviteTokenClaims, error) {
	claims := &model.InviteTokenClaims{}

	// parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		secret := config.GetJWTSecret()
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
