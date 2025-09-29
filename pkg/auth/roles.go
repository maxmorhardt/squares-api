package auth

import (
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
)

func RoleMiddleware(verifier *oidc.IDTokenVerifier, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idToken := verifyToken(c, verifier)
		if idToken == nil {
			return
		}

		validateRoles(c, idToken, allowedRoles...)
	}
}

func verifyToken(c *gin.Context, verifier *oidc.IDTokenVerifier) *oidc.IDToken {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return nil
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	idToken, err := verifier.Verify(c.Request.Context(), token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return nil
	}

	return idToken
}

func validateRoles(c *gin.Context, idToken *oidc.IDToken, allowedRoles ...string) {
	var claims struct {
		Username string   `json:"preferred_username"`
		Roles    []string `json:"roles"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Failed to parse claims"})
		return
	}

	if len(allowedRoles) == 0 {
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
		return
	}

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
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	c.Set("username", claims.Username)
	c.Set("roles", claims.Roles)

	c.Next()
}