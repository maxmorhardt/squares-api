package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := newRateLimiter(2)

	assert.True(t, rl.allow("1.1.1.1"))
	assert.True(t, rl.allow("1.1.1.1"))
	assert.False(t, rl.allow("1.1.1.1"), "third request over limit of 2")
	assert.True(t, rl.allow("2.2.2.2"), "different ip is tracked independently")
}

func TestContactRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/contact", ContactRateLimitMiddleware(1), func(c *gin.Context) { c.Status(http.StatusOK) })

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost, "/contact", http.NoBody))
	assert.Equal(t, http.StatusOK, w1.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodPost, "/contact", http.NoBody))
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestContactRateLimitMiddleware_NonPositiveLimitDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/contact", ContactRateLimitMiddleware(0), func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/contact", http.NoBody))
	assert.Equal(t, http.StatusOK, w.Code)
}
