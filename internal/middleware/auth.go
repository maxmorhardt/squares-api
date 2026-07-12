package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
)

const authErrorMessage = "Authentication required. Please log in to continue"

var (
	errClaimsParse     = errors.New("claims parse failed")
	errEmailUnverified = errors.New("token has no verified email")
)

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*model.Claims, error)
}

type oidcTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCTokenVerifier(verifier *oidc.IDTokenVerifier) TokenVerifier {
	return &oidcTokenVerifier{verifier: verifier}
}

func (v *oidcTokenVerifier) Verify(ctx context.Context, token string) (*model.Claims, error) {
	idToken, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &model.Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("%w: %w", errClaimsParse, err)
	}

	// email is the identity key across providers, so it must be present and verified
	if claims.Email == "" || !claims.EmailVerified {
		return nil, errEmailUnverified
	}

	return claims, nil
}

func AuthMiddleware(verifier TokenVerifier) gin.HandlerFunc {
	// verify token from authorization header
	return func(c *gin.Context) {
		claims := verifyToken(c, verifier, false)
		authMiddleware(c, claims)
	}
}

func AuthMiddlewareWS(verifier TokenVerifier) gin.HandlerFunc {
	// verify token from websocket protocol header
	return func(c *gin.Context) {
		claims := verifyToken(c, verifier, true)
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
	util.SetGinContextValue(c, model.UserKey, claims.Email)
	util.SetGinContextValue(c, model.ClaimsKey, claims)

	// add user to logger
	log := util.LoggerFromGinContext(c)
	log = log.With("user", claims.Email)
	util.SetGinContextValue(c, model.LoggerKey, log)

	c.Next()
}

func verifyToken(c *gin.Context, verifier TokenVerifier, isWebSocket bool) *model.Claims {
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

	// verify token and extract claims via the injected verifier
	claims, err := verifier.Verify(c.Request.Context(), token)
	if err != nil {
		log.Warn("failed to verify token", "error", err)
		if errors.Is(err, errClaimsParse) {
			metrics.RecordAuthFailure(model.AuthFailureClaimsParse)
		} else {
			metrics.RecordAuthFailure(model.AuthFailureVerifyFailed)
		}
		if !isWebSocket {
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
		}

		return nil
	}

	return claims
}
