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
		// verify token from authorization header
		claims := verifyToken(c, false)
		authMiddleware(c, claims, allowedGroups...)
	}
}

func AuthMiddlewareWS(allowedGroups ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// verify token from websocket protocol header
		claims := verifyToken(c, true)
		authMiddleware(c, claims, allowedGroups...)
	}
}

func authMiddleware(c *gin.Context, claims *model.Claims, allowedGroups ...string) {
	// abort if token verification failed
	if claims == nil {
		c.Abort()
		return
	}

	// add user and claims to context
	util.SetGinContextValue(c, model.UserKey, claims.Username)
	util.SetGinContextValue(c, model.ClaimsKey, claims)

	// add user to logger
	log := util.LoggerFromGinContext(c)
	log = log.With("user", claims.Username)
	util.SetGinContextValue(c, model.LoggerKey, log)

	// if no group restrictions, proceed
	if len(allowedGroups) == 0 {
		c.Next()
		return
	}

	// validate user has required group membership
	validateGroups(c, claims, allowedGroups...)
	c.Next()
}

func verifyToken(c *gin.Context, isWebSocket bool) *model.Claims {
	log := util.LoggerFromGinContext(c)

	// extract token from appropriate header
	var token string
	if isWebSocket {
		// websocket uses sec-websocket-protocol header
		wsProtocol := c.Request.Header.Get("Sec-WebSocket-Protocol")
		if wsProtocol == "" {
			log.Warn("missing sec-websocket-protocol header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Missing Sec-WebSocket-Protocol header", c))
			return nil
		}

		token = wsProtocol
	} else {
		// http uses authorization bearer header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Missing Authorization header", c))
			return nil
		}

		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// verify token with oidc provider
	idToken, err := config.OIDCVerifier.Verify(c.Request.Context(), token)
	if err != nil {
		log.Warn("failed to verify token", "error", err)
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, "Invalid token", c))
		}

		return nil
	}

	// extract claims from token
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

	// create set of allowed groups for fast lookup
	allowedSet := make(map[string]struct{}, len(allowedGroups))
	for _, group := range allowedGroups {
		allowedSet[group] = struct{}{}
	}

	// check if user has any allowed group
	hasGroup := false
	for _, group := range claims.Groups {
		if _, allowedGroup := allowedSet[group]; allowedGroup {
			hasGroup = true
			break
		}
	}

	// abort if user doesn't have required group
	if !hasGroup {
		log.Warn("user forbidden", "groups", claims.Groups, "allowed_groups", allowedGroups)
		c.AbortWithStatusJSON(http.StatusForbidden, model.NewAPIError(http.StatusForbidden, "Forbidden", c))
		return
	}

	log.Info("user authorized", "groups", claims.Groups)
}
