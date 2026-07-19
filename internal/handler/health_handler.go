package handler

import (
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type HealthHandler interface {
	Liveness(c *gin.Context)
	Readiness(c *gin.Context)
}

type healthHandler struct {
	db           *gorm.DB
	natsConn     *nats.Conn
	oidcVerifier *oidc.IDTokenVerifier
}

func NewHealthHandler(db *gorm.DB, natsConn *nats.Conn, oidcVerifier *oidc.IDTokenVerifier) HealthHandler {
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
// @Success 200 {object} model.LivenessResponse
// @Router /health/live [get]
func (h *healthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, model.LivenessResponse{Status: "UP"})
}

// Readiness godoc
// @Summary Readiness check
// @Description Checks database, NATS, and OIDC provider connectivity
// @Tags health
// @Produce json
// @Success 200 {object} model.ReadinessResponse
// @Failure 503 {object} model.ReadinessResponse
// @Router /health/ready [get]
func (h *healthHandler) Readiness(c *gin.Context) {
	checks := model.ReadinessResponse{}
	ready := true

	// database
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err == nil {
			err = sqlDB.Ping()
		}
		if err != nil {
			checks.Database = "DOWN"
			ready = false
		} else {
			checks.Database = "UP"
		}
	} else {
		checks.Database = "DOWN"
		ready = false
	}

	// nats
	if h.natsConn != nil && h.natsConn.IsConnected() {
		checks.Nats = "UP"
	} else {
		checks.Nats = "DOWN"
		ready = false
	}

	// oidc
	if h.oidcVerifier != nil {
		checks.OIDC = "UP"
	} else {
		checks.OIDC = "DOWN"
		ready = false
	}

	if ready {
		checks.Status = "UP"
		c.JSON(http.StatusOK, checks)
	} else {
		checks.Status = "DOWN"
		c.JSON(http.StatusServiceUnavailable, checks)
	}
}
