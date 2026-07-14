package handler

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, _ string) (*model.Claims, error) {
	return nil, nil
}

func init() {
	gin.SetMode(gin.TestMode)
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		_ = v.RegisterValidation("safestring", func(fl validator.FieldLevel) bool {
			s := fl.Field().String()
			return s == "" || util.IsSafeString(s)
		})
	}
}

func TestHealth_Liveness(t *testing.T) {
	h := NewHealthHandler(nil, nil, nil)
	r := gin.New()
	r.GET("/health/live", h.Liveness)

	req, _ := http.NewRequest(http.MethodGet, "/health/live", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "UP")
}

func TestHealth_Readiness_AllDown(t *testing.T) {
	h := NewHealthHandler(nil, nil, nil)
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "DOWN")
}

func TestHealth_Readiness_OIDCVerifierPresent(t *testing.T) {
	h := NewHealthHandler(nil, nil, stubVerifier{})
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"oidc":"UP"`)
	assert.Contains(t, w.Body.String(), `"database":"DOWN"`)
	assert.Contains(t, w.Body.String(), `"nats":"DOWN"`)
}

func TestHealth_Readiness_NATSNotConnected(t *testing.T) {
	nc := new(nats.Conn)
	h := NewHealthHandler(nil, nc, nil)
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"nats":"DOWN"`)
}

func TestHealth_Readiness_DBConnPoolNil(t *testing.T) {
	gdb := &gorm.DB{Config: &gorm.Config{}}
	h := NewHealthHandler(gdb, nil, nil)
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"database":"DOWN"`)
}

func TestHealth_Readiness_DBPingSuccess(t *testing.T) {
	gdb := newTestGormDB(t)
	h := NewHealthHandler(gdb, nil, nil)
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), `"database":"UP"`)
}
func newTestGormDB(t *testing.T) *gorm.DB {
	t.Helper()

	sqlDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	gdb, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	return gdb
}

func TestHealth_Readiness_AllUp(t *testing.T) {
	gdb := newTestGormDB(t)

	addr := startFakeNATSServer(t)
	nc, err := nats.Connect(addr, nats.MaxReconnects(0), nats.Timeout(3*time.Second))
	require.NoError(t, err)
	t.Cleanup(func() { nc.Close() })

	h := NewHealthHandler(gdb, nc, stubVerifier{})
	r := gin.New()
	r.GET("/health/ready", h.Readiness)

	req, _ := http.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
	w := doRequest(r, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"UP"`)
	assert.Contains(t, w.Body.String(), `"database":"UP"`)
	assert.Contains(t, w.Body.String(), `"nats":"UP"`)
	assert.Contains(t, w.Body.String(), `"oidc":"UP"`)
}

func startFakeNATSServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

		fmt.Fprintf(conn,
			"INFO {\"server_id\":\"test\",\"version\":\"2.10.0\",\"go\":\"go1.21\",\"host\":\"127.0.0.1\",\"port\":%d,\"max_payload\":1048576,\"proto\":1,\"client_id\":1,\"auth_required\":false,\"tls_required\":false}\r\n",
			port)

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return
			}
			if scanner.Text() == "PING" {
				_, _ = fmt.Fprint(conn, "PONG\r\n")
				time.Sleep(300 * time.Millisecond)
				return
			}
		}
	}()

	return fmt.Sprintf("nats://127.0.0.1:%d", port)
}
