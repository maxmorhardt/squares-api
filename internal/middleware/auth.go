package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

func AuthMiddleware(allowedGroups ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := verifyToken(c, false)
		authMiddleware(c, claims, allowedGroups...)
	}
}

func AuthMiddlewareWS(allowedGroups ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := verifyToken(c, true)
		authMiddleware(c, claims, allowedGroups...)
	}
}

func authMiddleware(c *gin.Context, claims *model.Claims, allowedGroups ...string) {
	if claims == nil {
		c.Abort()
		return
	}

	util.SetGinContextValue(c, model.UserKey, claims.Username)
	util.SetGinContextValue(c, model.ClaimsKey, claims)

	log := util.LoggerFromGinContext(c)
	log = log.With("user", claims.Username)
	util.SetGinContextValue(c, model.LoggerKey, log)

	if len(allowedGroups) == 0 {
		c.Next()
		return
	}

	validateGroups(c, claims, allowedGroups...)
	c.Next()
}

func verifyToken(c *gin.Context, isWebSocket bool) *model.Claims {
	log := util.LoggerFromGinContext(c)

	var token string
	if isWebSocket {
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

	idToken, err := config.OIDCVerifier.Verify(c.Request.Context(), token)
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

func validateGroups(c *gin.Context, claims *model.Claims, allowedGroups ...string) {
	log := util.LoggerFromGinContext(c)

	allowedSet := make(map[string]struct{}, len(allowedGroups))
	for _, group := range allowedGroups {
		allowedSet[group] = struct{}{}
	}

	hasGroup := false
	for _, group := range claims.Groups {
		if _, allowedGroup := allowedSet[group]; allowedGroup {
			hasGroup = true
			break
		}
	}

	if !hasGroup {
		log.Warn("user forbidden", "groups", claims.Groups, "allowed_groups", allowedGroups)
		c.AbortWithStatusJSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, "Forbidden", c))
		return
	}

	log.Info("user authorized", "groups", claims.Groups)
}
