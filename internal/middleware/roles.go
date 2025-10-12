package middleware

import (
	"net/http"
	"strings"

	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
)

func RoleMiddleware(verifier *oidc.IDTokenVerifier, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := FromContext(c)

		claims := VerifyToken(c, verifier, log, false)
		if claims == nil {
			return
		}

		if len(allowedRoles) == 0 {
			setContext(c, claims)
			return
		}

		ValidateRoles(c, claims, log, allowedRoles...)
	}
}

func VerifyToken(c *gin.Context, verifier *oidc.IDTokenVerifier, log *slog.Logger, isWebSocket bool) *model.Claims {
	var token string
	if  isWebSocket {
			wsProtocol := c.Request.Header.Get("Sec-WebSocket-Protocol")
			if wsProtocol == "" {
				log.Warn("missing sec-websocket-protocol header")
				c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Missing Sec-WebSocket-Protocol header", c))
				return nil
			}

			token = wsProtocol
	} else {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Missing Authorization header", c))
			return nil
		}

		token = strings.TrimPrefix(authHeader, "Bearer ")
	}
	
	idToken, err := verifier.Verify(c.Request.Context(), token)
	if err != nil {
		log.Warn("failed to verify token", "error", err)
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Invalid token", c))
		}
		return nil
	}

	claims := model.Claims{}
	if err := idToken.Claims(&claims); err != nil {
		log.Warn("failed to parse claims", "err", err)
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Invalid token claims", c))
		}
		return nil
	}

	return &claims
}

func ValidateRoles(c *gin.Context, claims *model.Claims, log *slog.Logger, allowedRoles ...string) {
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
		log.Warn("user forbidden", "user", claims.Username, "roles", claims.Roles, "allowedRoles", allowedRoles)
		c.AbortWithStatusJSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, "Forbidden", c))
		return
	}

	log.Info("user authorized", "user", claims.Username, "roles", claims.Roles)
	setContext(c, claims)
}

func setContext(c *gin.Context, claims *model.Claims) {
	c.Set(model.UserKey, claims.Username)
	c.Set(model.RolesKey, claims.Roles)
	c.Next()
}
