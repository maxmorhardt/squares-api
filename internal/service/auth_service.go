package service

import (
	"context"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type AuthService interface {
	IsDeclaredUser(ctx context.Context, user string) bool
	HasRole(ctx context.Context, role string) bool
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) IsDeclaredUser(ctx context.Context, user string) bool {
	ctxUser := ctx.Value(model.UserKey).(string)
	return ctxUser == user
}

func (s *authService) HasRole(ctx context.Context, role string) bool {
	claims := util.ClaimsFromContext(ctx)
	if claims == nil {
		return false
	}

	for key := range claims.Roles {
		if key == role {
			return true
		}
	}

	return false
}
