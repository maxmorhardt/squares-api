package service

import (
	"slices"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

type AuthService interface{
	IsDeclaredUser(c *gin.Context, user string) bool
	IsInGroup(c *gin.Context, group string) bool
	IsAdmin(c *gin.Context) bool
}

type authService struct{}

func NewAuthService() AuthService {
	return &authService{}
}

func (s *authService) IsDeclaredUser(c *gin.Context, user string) bool {
	if s.IsAdmin(c) {
		return true
	}

	ctxUser := c.GetString(model.UserKey)
	return ctxUser == user
}

func (s *authService) IsInGroup(c *gin.Context, group string) bool {
	claims := util.ClaimsFromContext(c)
	if claims == nil {
		return false
	}

	return slices.Contains(claims.Groups, group)
}

func (s *authService) IsAdmin(c *gin.Context) bool {
	return s.IsInGroup(c, model.SquaresAdminGroup)
}