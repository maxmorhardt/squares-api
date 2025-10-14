package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/service"
	"github.com/maxmorhardt/squares-api/internal/util"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
func WebSocketHandler(c *gin.Context) {
	log := util.LoggerFromContext(c)

	contestId := service.ValidateWebSocketRequest(c)
	if contestId == uuid.Nil {
		return
	}

	log.With("contest_id", contestId)
	user := c.GetString(model.UserKey)

	// need header in response
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

	log.Info("websocket connection established")
	if err := sendWebSocketMessage(conn, log, model.NewConnectedMessage(contestId)); err != nil {
		log.Error("failed to send connected message", "error", err)
		return
	}

	handleWebSocketConnection(conn, c, log, contestId, user)
}

func handleWebSocketConnection(conn *websocket.Conn, c *gin.Context, log *slog.Logger, contestId uuid.UUID, username string) {
	ctx := c.Request.Context()

	contestChannel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestId)
	log.With("contest_channel", contestChannel)
	log.Info("subscribing to redis channel")

	pubsub := config.RedisClient.Subscribe(ctx, contestChannel)
	defer func() {
		log.Info("closing redis subscription")
		pubsub.Close()
	}()

	redisChannel := pubsub.Channel()

	pingChecker := time.NewTicker(30 * time.Second)
	defer pingChecker.Stop()

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	jwtChecker := time.NewTicker(5 * time.Minute)
	defer jwtChecker.Stop()

	go handleIncomingMessages(conn)

	// main event loop
	for {
		select {
		case msg := <-redisChannel:
			var updateData model.WSUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
				log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
				return
			}

			if err := sendWebSocketMessage(conn, log, &updateData); err != nil {
				log.Error("failed to send redis message to websocket client", "error", err)
				return
			}

		case <-pingChecker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("failed to send ping", "error", err)
				return
			}

		case <-jwtChecker.C:
			if shouldCloseConnection(c, log, username) {
				log.Warn("closing connection due to token validation failure")
				if err := sendWebSocketMessage(conn, log, model.NewDisconnectedMessage(contestId)); err != nil {
					log.Error("failed to send disconnected message", "error", err)
				}
				conn.Close()
				return
			}

		case <-ctx.Done():
			log.Info("websocket client disconnected")
			return
		}
	}
}

// ignore incoming messages
func handleIncomingMessages(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func sendWebSocketMessage(conn *websocket.Conn, log *slog.Logger, data *model.WSUpdate) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("failed to marshal websocket message", "error", err, "type", data.Type)
		return err
	}

	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Error("failed to write websocket message", "error", err)
		return err
	}

	log.Info("sending websocket message", "type", data.Type, "updated_by", data.UpdatedBy)
	return nil
}

func shouldCloseConnection(c *gin.Context, log *slog.Logger, username string) bool {
	claims := util.ClaimsFromContext(c)		
	if claims == nil {
		log.Warn("claims not in ctx for websocket connection")
		return true
	}

	now := time.Now().Unix()
	if claims.Expire < now {
		log.Info("token expired for websocket connection", "expire", claims.Expire, "now", now)
		return true
	}

	return false
}
