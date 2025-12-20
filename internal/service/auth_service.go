package service

import (
	"context"
	"slices"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type AuthService interface {
	IsDeclaredUser(ctx context.Context, user string) bool
	IsInGroup(ctx context.Context, group string) bool
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
