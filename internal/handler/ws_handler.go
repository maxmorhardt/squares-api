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
		return true
	},
}

// @Summary Connect to WebSocket for real-time contest updates
// @Description Establishes a persistent WebSocket connection to receive real-time updates for a specific contest
// @Tags events
// @Param contestId path string true "Contest ID to listen for updates" format(uuid)
// @Success 101 {string} string "WebSocket connection upgraded"
// @Failure 400 {object} model.APIError
// @Failure 401 {object} model.APIError
// @Failure 404 {object} model.APIError
// @Failure 500 {object} model.APIError
// @Security BearerAuth
// @Router /ws/contests/{contestId} [get]
func WebSocketHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	claims, contestId := validateWebSocketRequest(c, log)
	if claims == nil || contestId == uuid.Nil {
		return
	}

	token := c.Request.Header.Get("Sec-WebSocket-Protocol")
	responseHeader := http.Header{}
	if token != "" {
		responseHeader.Set("Sec-WebSocket-Protocol", token)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Error("failed to upgrade connection to websocket", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Failed to upgrade to WebSocket", c))

		return
	}
	defer conn.Close()

	log.Info("websocket connection established", "user", claims.Username, "contestId", contestId)
	if err := sendWebSocketMessage(conn, log, model.NewConnectedMessage(contestId, claims.Username)); err != nil {
		log.Error("failed to send connected message", "error", err)
		return
	}

	handleWebSocketConnection(conn, c, log, contestId, claims.Username)
}

func validateWebSocketRequest(c *gin.Context, log *slog.Logger) (*model.Claims, uuid.UUID) {
	claims := middleware.VerifyToken(c, config.OIDCVerifier, log, true)
	if claims == nil {
		return nil, uuid.Nil
	}

	contestId, err := uuid.Parse(c.Param("contestId"))
	if err != nil || contestId == uuid.Nil {
		log.Error("invalid or missing contest id", "error", err)
		c.JSON(http.StatusBadRequest, model.NewAPIError(http.StatusBadRequest, "Invalid or missing Contest ID", c))
		return nil, uuid.Nil
	}

	contestRepo := repository.NewContestRepository()
	_, err = contestRepo.GetByID(c.Request.Context(), contestId.String())
	if err != nil {
		log.Error("contest not found", "contestId", contestId)
		c.JSON(http.StatusNotFound, model.NewAPIError(http.StatusNotFound, "Contest not found", c))
		return nil, uuid.Nil
	}

	log.Info("websocket client validated", "user", claims.Username, "contestId", contestId)
	return claims, contestId
}

func handleWebSocketConnection(conn *websocket.Conn, c *gin.Context, log *slog.Logger, contestId uuid.UUID, username string) {
	ctx := c.Request.Context()

	contestChannel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestId)
	log.Info("subscribing to redis channel", "channel", contestChannel)

	pubsub := config.RedisClient.Subscribe(ctx, contestChannel)
	defer func() {
		log.Info("closing redis subscription", "channel", contestChannel)
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
			if err := sendWebSocketMessage(conn, log, model.NewKeepAliveMessage(contestId)); err != nil {
				log.Warn("failed to send keepalive - closing connection", "error", err)
				return
			}

		case <-jwtChecker.C:
			if shouldCloseConnection(c, log, username) {
				log.Info("closing websocket connection - token validation failed", "user", username)
				if err := sendWebSocketMessage(conn, log, model.NewClosedConnectionMessage(contestId, username)); err != nil {
					log.Warn("failed to send closed connection message", "error", err)
				}
				conn.Close()
				return
			}

		case <-ctx.Done():
			log.Info("websocket client disconnected", "user", username, "contestId", contestId)
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
	var updateData model.ContestChannelResponse
	if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
		log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
		return nil
	}

	log.Info("sending redis update to client", "type", updateData.Type, "contestId", updateData.ContestID, "squareId", updateData.SquareID)
	return sendWebSocketMessage(conn, log, &updateData)
}

func sendWebSocketMessage(conn *websocket.Conn, log *slog.Logger, data *model.ContestChannelResponse) error {
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
	claims := middleware.VerifyToken(c, config.OIDCVerifier, log, true)

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
