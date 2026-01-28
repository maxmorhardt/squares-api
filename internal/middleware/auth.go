package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const authErrorMessage = "Authentication required. Please log in to continue"

func AuthMiddleware() gin.HandlerFunc {
	// verify token from authorization header
	return func(c *gin.Context) {
		claims := verifyToken(c, false)
		authMiddleware(c, claims)
	}
}

func AuthMiddlewareWS() gin.HandlerFunc {
	// verify token from websocket protocol header
	return func(c *gin.Context) {
		claims := verifyToken(c, true)
		authMiddleware(c, claims)
	}
}

func authMiddleware(c *gin.Context, claims *model.Claims) {
	// abort if token verification failed
	if claims == nil {
		c.Abort()
		return
	}

	// add user and claims to context
	util.SetGinContextValue(c, model.UserKey, claims.Subject)
	util.SetGinContextValue(c, model.ClaimsKey, claims)

	// add user to logger
	log := util.LoggerFromGinContext(c)
	log = log.With("user", claims.Subject)
	util.SetGinContextValue(c, model.LoggerKey, log)

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
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return nil
		}

		token = wsProtocol
	} else {
		// http uses authorization bearer header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return nil
		}

		token = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// verify token with oidc provider
	idToken, err := config.OIDCVerifier.Verify(c.Request.Context(), token)
	if err != nil {
		log.Warn("failed to verify token", "error", err)
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
		}

		return nil
	}

	// extract claims from token
	claims := model.Claims{}
	if err := idToken.Claims(&claims); err != nil {
		log.Warn("failed to parse claims", "err", err)
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
		}

		return nil
	}

	return &claims
}
