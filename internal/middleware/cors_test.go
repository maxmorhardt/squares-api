package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set("Origin", "http://allowed.com")

	w := httptest.NewRecorder()
	corsRouter().ServeHTTP(w, req)

	assert.Equal(t, "http://allowed.com", w.Header().Get("Access-Control-Allow-Origin"))
}

func corsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware([]string{"http://allowed.com"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	req.Header.Set("Origin", "http://evil.com")

	w := httptest.NewRecorder()
	corsRouter().ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}
