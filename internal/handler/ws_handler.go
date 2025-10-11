package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "http://localhost:3000" || origin == "https://squares.maxstash.io"
	},
}

// @Summary Connect to WebSocket for real-time grid updates
// @Description Establishes a persistent WebSocket connection to receive real-time updates for a specific grid
// @Tags events
// @Param gridId path string true "Grid ID to listen for updates" format(uuid)
// @Success 101 {string} string "WebSocket connection upgraded"
// @Failure 400 {object} model.APIError
// @Failure 401 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /events/{gridId} [get]
func WebSocketHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	claims, gridId := validateWebSocketRequest(c, log)
	if claims == nil || gridId == uuid.Nil {
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("failed to upgrade connection to websocket", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Failed to upgrade to WebSocket", c))
		return
	}
	defer conn.Close()

	log.Info("websocket connection established", "user", claims.Username, "gridId", gridId)

	if err := sendWebSocketMessage(conn, log, model.NewConnectedMessage(gridId, claims.Username)); err != nil {
		log.Error("failed to send connected message", "error", err)
		return
	}

	handleWebSocketConnection(conn, c, log, gridId, claims.Username)
}

func validateWebSocketRequest(c *gin.Context, log *slog.Logger) (*model.Claims, uuid.UUID) {
	claims := middleware.VerifyToken(c, config.OIDCVerifier, log)
	if claims == nil {
		return nil, uuid.Nil
	}

	gridId, err := uuid.Parse(c.Param("gridId"))
	if err != nil || gridId == uuid.Nil {
		log.Error("invalid or missing grid id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid or missing Grid ID", c))
		return nil, uuid.Nil
	}

	gridRepo := repository.NewGridRepository()
	_, err = gridRepo.GetByID(c.Request.Context(), gridId.String())
	if err != nil {
		log.Error("grid not found", "gridId", gridId)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Grid not found", c))
		return nil, uuid.Nil
	}

	log.Info("websocket client validated", "user", claims.Username, "gridId", gridId)
	return claims, gridId
}

func handleWebSocketConnection(conn *websocket.Conn, c *gin.Context, log *slog.Logger, gridId uuid.UUID, username string) {
	ctx := c.Request.Context()

	gridChannel := fmt.Sprintf("%s:%s", model.GridChannelPrefix, gridId)
	log.Info("subscribing to redis channel", "channel", gridChannel)

	pubsub := config.RedisClient.Subscribe(ctx, gridChannel)
	defer func() {
		log.Info("closing redis subscription", "channel", gridChannel)
		pubsub.Close()
	}()

	redisChannel := pubsub.Channel()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Handle incoming WebSocket messages from client
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Info("websocket read error, client disconnected", "error", err)
				return
			}
			// For now, we just read and ignore client messages
			// In the future, we could handle client-to-server messages here
		}
	}()

	for {
		select {
		case msg := <-redisChannel:
			log.Info("received redis message", "channel", msg.Channel)
			if err := handleWebSocketRedisMessage(conn, log, msg); err != nil {
				log.Warn("failed to handle redis message - closing connection", "error", err)
				return
			}

		case <-ticker.C:
			if err := sendWebSocketMessage(conn, log, model.NewKeepAliveMessage(gridId)); err != nil {
				log.Warn("failed to send keepalive - closing connection", "error", err)
				return
			}

		case <-ctx.Done():
			log.Info("websocket client disconnected", "user", username, "gridId", gridId)
			return
		}
	}
}

func handleWebSocketRedisMessage(conn *websocket.Conn, log *slog.Logger, msg *redis.Message) error {
	var updateData model.GridChannelResponse
	if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
		log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
		return nil
	}

	log.Info("sending redis update to client", "type", updateData.Type, "gridId", updateData.GridID, "cellId", updateData.CellID)
	return sendWebSocketMessage(conn, log, &updateData)
}

func sendWebSocketMessage(conn *websocket.Conn, log *slog.Logger, data *model.GridChannelResponse) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("failed to marshal websocket message", "error", err, "type", data.Type)
		return err
	}

	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Error("failed to write websocket message", "error", err)
		return err
	}

	log.Debug("websocket message sent successfully", "type", data.Type, "size", len(jsonData))
	return nil
}
