package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(&fakeVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached, "handler should not run without auth")
}

func buildRouter(mw gin.HandlerFunc) (router *gin.Engine, reached *bool) {
	gin.SetMode(gin.TestMode)
	didReach := false
	router = gin.New()
	router.GET("/protected", mw, func(c *gin.Context) {
		didReach = true
		c.String(http.StatusOK, c.GetString(model.UserKey))
	})
	return router, &didReach
}

type fakeVerifier struct {
	claims *model.Claims
	err    error
}

func (f *fakeVerifier) Verify(_ context.Context, _ string) (*model.Claims, error) {
	return f.claims, f.err
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(&fakeVerifier{claims: &model.Claims{Email: "alice@example.com", EmailVerified: true}}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_BadPrefix(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(&fakeVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_VerifyError(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(&fakeVerifier{err: errors.New("invalid token")}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer badtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_ClaimsParseError(t *testing.T) {
	claimsErr := fmt.Errorf("%w: %w", errs.ErrClaimsParse, errors.New("json: cannot unmarshal"))
	r, reached := buildRouter(AuthMiddleware(&fakeVerifier{err: claimsErr}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer validtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_Success(t *testing.T) {
	verifier := &fakeVerifier{claims: &model.Claims{Email: "alice@example.com", EmailVerified: true}}
	r, reached := buildRouter(AuthMiddleware(verifier))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer goodtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *reached, "handler should run on valid auth")
	assert.Equal(t, "alice@example.com", w.Body.String(), "authenticated user should be in context")
}

func TestAuthMiddlewareWS_MissingProtocolHeader(t *testing.T) {
	r, reached := buildRouter(AuthMiddlewareWS(&fakeVerifier{}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddlewareWS_VerifyError_Aborts(t *testing.T) {
	r, reached := buildRouter(AuthMiddlewareWS(&fakeVerifier{err: errors.New("invalid token")}))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Sec-WebSocket-Protocol", "badtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// WS path does not write a JSON error body, but it must still abort
	assert.False(t, *reached, "handler should not run when ws token is invalid")
}

func TestAuthMiddlewareWS_Success(t *testing.T) {
	verifier := &fakeVerifier{claims: &model.Claims{Email: "bob@example.com", EmailVerified: true}}
	r, reached := buildRouter(AuthMiddlewareWS(verifier))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Sec-WebSocket-Protocol", "goodtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *reached)
	assert.Equal(t, "bob@example.com", w.Body.String())
}
