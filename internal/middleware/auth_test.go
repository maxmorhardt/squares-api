package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/errs"
	"github.com/maxmorhardt/squares-api/internal/mocks"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

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

func userServiceReturning(t *testing.T, claims *model.Claims, err error) *mocks.UserService {
	m := mocks.NewUserService(t)
	m.EXPECT().VerifyToken(mock.Anything, mock.Anything).Return(claims, err).Maybe()
	return m
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(userServiceReturning(t, nil, nil)))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached, "handler should not run without auth")
}

func TestAuthMiddleware_EmptyToken(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(userServiceReturning(t, nil, nil)))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_BadPrefix(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(userServiceReturning(t, nil, nil)))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Basic abc123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_VerifyError(t *testing.T) {
	r, reached := buildRouter(AuthMiddleware(userServiceReturning(t, nil, errors.New("invalid token"))))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer badtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_ClaimsParseError(t *testing.T) {
	claimsErr := fmt.Errorf("%w: %w", errs.ErrClaimsParse, errors.New("json: cannot unmarshal"))
	r, reached := buildRouter(AuthMiddleware(userServiceReturning(t, nil, claimsErr)))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer validtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddleware_Success(t *testing.T) {
	us := userServiceReturning(t, &model.Claims{Email: "alice@example.com", EmailVerified: true}, nil)
	r, reached := buildRouter(AuthMiddleware(us))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer goodtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *reached, "handler should run on valid auth")
	assert.Equal(t, "alice@example.com", w.Body.String(), "authenticated user should be in context")
}

func TestAuthMiddlewareWS_MissingProtocolHeader(t *testing.T) {
	r, reached := buildRouter(AuthMiddlewareWS(userServiceReturning(t, nil, nil)))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, *reached)
}

func TestAuthMiddlewareWS_VerifyError_Aborts(t *testing.T) {
	r, reached := buildRouter(AuthMiddlewareWS(userServiceReturning(t, nil, errors.New("invalid token"))))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Sec-WebSocket-Protocol", "badtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// WS path does not write a JSON error body, but it must still abort with a 401
	assert.False(t, *reached, "handler should not run when ws token is invalid")
	assert.Equal(t, http.StatusUnauthorized, w.Code, "response status should be 401, not gin's default 200")
	assert.Equal(t, 0, w.Body.Len(), "no response body should be written for ws verify failures")
}

func TestAuthMiddlewareWS_Success(t *testing.T) {
	us := userServiceReturning(t, &model.Claims{Email: "bob@example.com", EmailVerified: true}, nil)
	r, reached := buildRouter(AuthMiddlewareWS(us))

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Sec-WebSocket-Protocol", "goodtoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *reached)
	assert.Equal(t, "bob@example.com", w.Body.String())
}
