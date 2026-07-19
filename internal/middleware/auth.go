package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const authErrorMessage = "Authentication required. Please log in to continue"

func AuthMiddleware(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := verifyToken(c, userService, false)
		authMiddleware(c, claims)
	}
}

func AuthMiddlewareWS(userService service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := verifyToken(c, userService, true)
		authMiddleware(c, claims)
	}
}

func authMiddleware(c *gin.Context, claims *model.Claims) {
	// abort if token verification failed
	if claims == nil {
		if !c.IsAborted() {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
		}
		return
	}

	// add user and claims to context
	util.SetGinContextValue(c, model.UserKey, claims.Email)
	util.SetGinContextValue(c, model.ClaimsKey, claims)

	// add user to logger
	log := util.LoggerFromGinContext(c)
	log = log.With("user", claims.Email)
	util.SetGinContextValue(c, model.LoggerKey, log)

	c.Next()
}

func verifyToken(c *gin.Context, userService service.UserService, isWebSocket bool) *model.Claims {
	log := util.LoggerFromGinContext(c)

	// extract token from appropriate header
	var token string
	if isWebSocket {
		// websocket uses sec-websocket-protocol header
		wsProtocol := c.Request.Header.Get("Sec-WebSocket-Protocol")
		if wsProtocol == "" {
			log.Warn("missing sec-websocket-protocol header")
			metrics.RecordAuthFailure(model.AuthFailureMissingHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return nil
		}

		token = wsProtocol
	} else {
		// http uses authorization bearer header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("missing authorization header")
			metrics.RecordAuthFailure(model.AuthFailureMissingHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return nil
		}

		token = strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Warn("empty bearer token")
			metrics.RecordAuthFailure(model.AuthFailureMissingHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return nil
		}
	}

	claims, err := userService.VerifyToken(c.Request.Context(), token)
	if err != nil {
		log.Warn("failed to verify token", "error", err)
		if errors.Is(err, errs.ErrClaimsParse) {
			metrics.RecordAuthFailure(model.AuthFailureClaimsParse)
		} else {
			metrics.RecordAuthFailure(model.AuthFailureVerifyFailed)
		}
		if isWebSocket {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
		}

		return nil
	}

	return claims
}
