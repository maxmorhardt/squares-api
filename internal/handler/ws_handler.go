package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/metrics"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

func newUpgrader() websocket.Upgrader {
	originSet := make(map[string]bool, len(config.Env().Server.AllowedOrigins))
	for _, o := range config.Env().Server.AllowedOrigins {
		originSet[o] = true
	}

	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return originSet[r.Header.Get("Origin")]
		},
	}
}

type WebSocketHandler interface {
	ContestWSConnection(c *gin.Context)
}

type websocketHandler struct {
	websocketService   service.WebSocketService
	contestRepo        repository.ContestRepository
	participantService service.ParticipantService
	upgrader           websocket.Upgrader
	natsAvailable      func() bool
}

func NewWebSocketHandler(websocketService service.WebSocketService, contestRepo repository.ContestRepository, participantService service.ParticipantService) WebSocketHandler {
	return &websocketHandler{
		websocketService:   websocketService,
		contestRepo:        contestRepo,
		participantService: participantService,
		upgrader:           newUpgrader(),
		natsAvailable: func() bool {
			nc := config.NATS()
			return nc != nil && nc.IsConnected()
		},
	}
}

// @Summary Connect to WebSocket for real-time contest updates
// @Description Establishes a persistent WebSocket connection to receive real-time updates for a specific contest
// @Tags ws
// @Param owner path string true "Contest Owner"
// @Param name path string true "Contest Name"
// @Success 101 {string} string "WebSocket connection upgraded"
// @Failure 400 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /ws/contests/owner/{owner}/name/{name} [get]
func (h *websocketHandler) ContestWSConnection(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse path vars
	owner := c.Param("owner")
	if owner == "" {
		log.Warn("contest owner not provided")
		metrics.RecordWSConnectionResult(model.WSResultBadRequest)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest Owner is required", c))
		return
	}

	name := c.Param("name")
	if name == "" {
		log.Warn("contest name not provided")
		metrics.RecordWSConnectionResult(model.WSResultBadRequest)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest Name is required", c))
		return
	}

	// extract websocket protocol token from headers
	token := c.Request.Header.Get("Sec-WebSocket-Protocol")
	responseHeader := http.Header{}
	responseHeader.Set("Sec-WebSocket-Protocol", token)

	// upgrade http connection to websocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		metrics.RecordWSConnectionResult(model.WSResultUpgradeFailed)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to upgrade connection", c))
		return
	}

	// validate contest exists and check status
	contest, err := h.contestRepo.GetByOwnerAndName(c.Request.Context(), owner, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("contest not found, closing websocket")
			metrics.RecordWSConnectionResult(model.WSResultNotFound)
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4404, "Contest not found"))
			_ = conn.Close()
			return
		}

		log.Warn("failed to validate websocket request", "error", err)
		metrics.RecordWSConnectionResult(model.WSResultInternalError)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4500, "Failed to get contest"))
		_ = conn.Close()
		return
	}

	// check if user has permission to view this contest
	user := c.GetString(model.UserKey)
	if authErr := h.participantService.Authorize(c.Request.Context(), contest.ID, user, service.ActionView); authErr != nil {
		log.Warn("user not authorized for websocket", "user", user, "contest_id", contest.ID)
		metrics.RecordWSConnectionResult(model.WSResultUnauthorized)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4403, "Not authorized"))
		_ = conn.Close()
		return
	}

	log = log.With("contest_id", contest.ID)
	util.SetGinContextValue(c, model.LoggerKey, log)

	// verify NATS is available before handing off to the service
	if !h.natsAvailable() {
		log.Error("NATS connection not available, rejecting websocket")
		metrics.RecordWSConnectionResult(model.WSResultUnavailable)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4503, "Real-time updates unavailable"))
		_ = conn.Close()
		return
	}

	// fetch participants to include in the connected message
	participants, err := h.participantService.GetParticipants(c.Request.Context(), contest.ID, user)
	if err != nil {
		log.Error("failed to fetch participants for websocket connected message", "error", err)
		metrics.RecordWSConnectionResult(model.WSResultInternalError)
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4500, "Failed to load contest"))
		_ = conn.Close()
		return
	}

	// hand off to service which records the final connection result once the
	// connection is fully initialized (NATS subscribed and connected message sent)
	h.websocketService.HandleWebSocketConnection(c.Request.Context(), contest, participants, conn)
}
