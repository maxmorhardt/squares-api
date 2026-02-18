package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:3000" || origin == "https://squares.maxstash.io"
	},
}

type WebSocketHandler interface {
	ContestWSConnection(c *gin.Context)
}

type websocketHandler struct {
	websocketService service.WebSocketService
	contestRepo      repository.ContestRepository
}

func NewWebSocketHandler(websocketService service.WebSocketService, contestRepo repository.ContestRepository) WebSocketHandler {
	return &websocketHandler{
		websocketService: websocketService,
		contestRepo:      contestRepo,
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
// @Router /ws/contests/{owner}/{name} [get]
func (h *websocketHandler) ContestWSConnection(c *gin.Context) {
	log := util.LoggerFromGinContext(c)

	// parse path vars
	owner := c.Param("owner")
	if owner == "" {
		log.Warn("contest owner not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest Owner is required", c))
		return
	}

	name := c.Param("name")
	if owner == "" {
		log.Warn("contest name not provided")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest Name is required", c))
		return
	}

	// extract websocket protocol token from headers
	token := c.Request.Header.Get("Sec-WebSocket-Protocol")
	responseHeader := http.Header{}
	responseHeader.Set("Sec-WebSocket-Protocol", token)

	// validate contest exists and check status
	contest, err := h.contestRepo.GetByOwnerAndName(c.Request.Context(), owner, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
			return
		}

		log.Warn("failed to validate websocket request", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to get contest", c))
		return
	}

	log = log.With("contest_id", contest.ID)
	util.SetGinContextValue(c, model.LoggerKey, log)

	// reject connection for finished or deleted contests
	if contest.Status == model.ContestStatusFinished || contest.Status == model.ContestStatusDeleted {
		log.Warn("websocket connection rejected for contest status", "status", contest.Status)
		c.JSON(http.StatusForbidden, model.NewAPIError(http.StatusBadRequest, "Cannot connect to contest in FINISHED or DELETED state", c))
		return
	}

	// upgrade http connection to websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Failed to upgrade connection", c))
		return
	}

	// handle websocket connection lifecycle
	h.websocketService.HandleWebSocketConnection(c.Request.Context(), contest.ID, conn)
}
