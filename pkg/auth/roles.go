package auth

import (
	"net/http"
	"strings"

	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/pkg/logger"
)

func RoleMiddleware(verifier *oidc.IDTokenVerifier, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := logger.FromContext(c)

		claims := verifyToken(c, verifier, logger)
		if claims == nil {
			return
		}

		if len(allowedRoles) == 0 {
			setContext(c, claims)
			return
		}

		validateRoles(c, claims, logger, allowedRoles...)
	}
}

func verifyToken(c *gin.Context, verifier *oidc.IDTokenVerifier, logger *slog.Logger) *Claims {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		logger.Warn("missing authorization header")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	idToken, err := verifier.Verify(c.Request.Context(), token)
	if err != nil {
		logger.Warn("failed to verify token", "err", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return nil
	}

	claims := Claims{}
	if err := idToken.Claims(&claims); err != nil {
		logger.Warn("failed to parse claims", "err", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return nil
	}

	logger.Info("token verified", "username", claims.Username, "roles", claims.Roles)
	return &claims
}

func validateRoles(c *gin.Context, claims *Claims, logger *slog.Logger, allowedRoles ...string) {
	allowedSet := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowedSet[role] = struct{}{}
	}

	hasRole := false
	for _, role := range claims.Roles {
		if _, allowedRole := allowedSet[role]; allowedRole {
			hasRole = true
			break
		}
	}

	if !hasRole {
		logger.Warn("user forbidden", "username", claims.Username, "roles", claims.Roles, "allowedRoles", allowedRoles)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	logger.Info("user authorized", "username", claims.Username, "roles", claims.Roles)
	setContext(c, claims)
}

func setContext(c *gin.Context, claims *Claims) {
	c.Set("username", claims.Username)
	c.Set("roles", claims.Roles)
	c.Next()
}