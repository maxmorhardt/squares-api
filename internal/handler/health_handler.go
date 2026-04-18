package handler

import (
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type HealthHandler interface {
	Liveness(c *gin.Context)
	Readiness(c *gin.Context)
}

type healthHandler struct {
	db           *gorm.DB
	natsConn     func() *nats.Conn
	oidcVerifier func() *oidc.IDTokenVerifier
}

func NewHealthHandler(db *gorm.DB, natsConn func() *nats.Conn, oidcVerifier func() *oidc.IDTokenVerifier) HealthHandler {
	return &healthHandler{
		db:           db,
		natsConn:     natsConn,
		oidcVerifier: oidcVerifier,
	}
}

// Liveness godoc
// @Summary Liveness check
// @Description Returns UP if the service process is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/live [get]
func (h *healthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "UP"})
}

// Readiness godoc
// @Summary Readiness check
// @Description Checks database, NATS, and OIDC provider connectivity
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /health/ready [get]
func (h *healthHandler) Readiness(c *gin.Context) {
	checks := gin.H{}
	ready := true

	// database
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err == nil {
			err = sqlDB.Ping()
		}
		if err != nil {
			checks["database"] = "DOWN"
			ready = false
		} else {
			checks["database"] = "UP"
		}
	} else {
		checks["database"] = "DOWN"
		ready = false
	}

	// nats
	nc := h.natsConn()
	if nc != nil && nc.IsConnected() {
		checks["nats"] = "UP"
	} else {
		checks["nats"] = "DOWN"
		ready = false
	}

	// oidc
	if h.oidcVerifier() != nil {
		checks["oidc"] = "UP"
	} else {
		checks["oidc"] = "DOWN"
		ready = false
	}

	if ready {
		checks["status"] = "UP"
		c.JSON(http.StatusOK, checks)
	} else {
		checks["status"] = "DOWN"
		c.JSON(http.StatusServiceUnavailable, checks)
	}
}