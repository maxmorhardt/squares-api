package service

import (
	"context"
	"slices"

	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type AuthService interface{
	IsDeclaredUser(ctx context.Context, user string) bool
	IsInGroup(ctx context.Context, group string) bool
	IsAdmin(ctx context.Context) bool
	IsContestOwner(ctx context.Context, contestOwner, user string) bool
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) IsDeclaredUser(ctx context.Context, user string) bool {
	if s.IsAdmin(ctx) {
		return true
	}

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

func (s *authService) IsAdmin(ctx context.Context) bool {
	return s.IsInGroup(ctx, model.SquaresAdminGroup)
}

func (s *authService) IsContestOwner(ctx context.Context, contestOwner, user string) bool {
	if s.IsAdmin(ctx) {
		return true
	}

	return contestOwner == user
}