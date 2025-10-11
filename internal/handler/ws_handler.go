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

var upgrader = websocket.Upgrader{}

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

	jwtChecker := time.NewTicker(5 * time.Minute)
	defer jwtChecker.Stop()

	go handleIncomingMessages(conn)

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

		case <-jwtChecker.C:
			if shouldCloseConnection(c, log, username) {
				log.Info("closing websocket connection - token validation failed", "user", username)
				if err := sendWebSocketMessage(conn, log, model.NewClosedConnectionMessage(gridId, username)); err != nil {
					log.Warn("failed to send closed connection message", "error", err)
				}

				conn.Close()
				return
			}

		case <-ctx.Done():
			log.Info("websocket client disconnected", "user", username, "gridId", gridId)
			return
		}
	}
}

func handleIncomingMessages(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
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

	return nil
}

func shouldCloseConnection(c *gin.Context, log *slog.Logger, username string) bool {
	claims := middleware.VerifyToken(c, config.OIDCVerifier, log)

	if claims == nil {
		log.Info("token validation failed for websocket connection", "user", username)
		return true
	}

	if claims.Username != username {
		log.Warn("username mismatch in token", "expected", username, "actual", claims.Username)
		return true
	}

	return false
}
