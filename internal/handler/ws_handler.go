package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler interface {
	ContestWSConnection(c *gin.Context)
}

type websocketHandler struct {
	websocketService service.WebSocketService
	validationService service.ValidationService
}

func NewWebSocketHandler(websocketService service.WebSocketService, validationService service.ValidationService) WebSocketHandler {
	return &websocketHandler{
		websocketService: websocketService,
	}
}

// @Summary Connect to WebSocket for real-time contest updates
// @Description Establishes a persistent WebSocket connection to receive real-time updates for a specific contest
// @Tags ws
// @Param contestId path string true "Contest ID to listen for updates" format(uuid)
// @Success 101 {string} string "WebSocket connection upgraded"
// @Failure 400 {object} model.APIError
// @Failure 401 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /ws/contests/{contestId} [get]
func (h *websocketHandler) ContestWSConnection(c *gin.Context) {
	log := util.LoggerFromContext(c)

	contestIDParam := c.Param("contestId")
	if contestIDParam == "" {
		log.Error("contest id is missing")
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Contest ID is required", c))
		return
	}

	contestID, err := uuid.Parse(contestIDParam)
	if err != nil {
		log.Error("invalid or missing contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid Contest ID", c))
		return
	}

	log.With("contest_id", contestID)

	token := c.Request.Header.Get("Sec-WebSocket-Protocol")
	responseHeader := http.Header{}
	responseHeader.Set("Sec-WebSocket-Protocol", token)

	if err := h.validationService.ValidateWSRequest(c.Request.Context(), contestID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
			return
		}

		log.Error("failed to validate websocket request", "error", err)
		c.JSON(http.StatusInternalServerError, model.NewAPIError(http.StatusInternalServerError, "Invalid Contest ID", c))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Failed to upgrade connection", c))
		return
	}
	
	h.websocketService.HandleWebSocketConnection(c.Request.Context(), contestID, conn)
}