package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestSizeLimit_SmallBodyPasses(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("field=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	sizeRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func sizeRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestSizeLimitMiddleware())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestRequestSizeLimit_TooLargeRejected(t *testing.T) {
	body := "field=" + strings.Repeat("a", (1<<20)+16)
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	sizeRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}
