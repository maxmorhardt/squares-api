package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
	"github.com/redis/go-redis/v9"
)

const (
	pingInterval     = 30 * time.Second
	pongTimeout      = 60 * time.Second
	writeDeadline    = 10 * time.Second
	jwtCheckInterval = 5 * time.Minute
)

type WebSocketService interface {
	HandleWebSocketConnection(ctx context.Context, contestID uuid.UUID, conn *websocket.Conn)
}

type websocketService struct{}

func NewWebSocketService() WebSocketService {
	return &websocketService{}
}

func (s *websocketService) HandleWebSocketConnection(ctx context.Context, contestID uuid.UUID, conn *websocket.Conn) {
	log := util.LoggerFromContext(ctx)

	// generate connection id and update context
	connectionID := uuid.New()
	log = log.With("connection_id", connectionID)
	ctx = context.WithValue(ctx, model.ConnectionIDKey, connectionID)

	// send initial connected message
	if err := sendWebSocketMessage(conn, log, model.NewConnectedMessage(contestID, connectionID)); err != nil {
		log.Error("failed to send connected message", "error", err)
	}

	// subscribe to redis channel for contest updates
	log.Info("subscribing to redis channel")
	contestChannel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestID)
	pubsub := config.RedisClient.Subscribe(ctx, contestChannel)
	defer func() {
		log.Info("closing redis subscription")
		if err := pubsub.Close(); err != nil {
			log.Error("failed to close redis subscription", "error", err)
		}
	}()
	redisChannel := pubsub.Channel()

	// setup ping/pong to keep connection alive
	pingChecker := time.NewTicker(pingInterval)
	defer pingChecker.Stop()

	// set read deadline and pong handler
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	// setup jwt token validation checker
	jwtChecker := time.NewTicker(jwtCheckInterval)
	defer jwtChecker.Stop()

	// start message handlers
	go s.handleIncomingMessages(conn)
	s.handleOutgoingMessages(ctx, conn, pingChecker, jwtChecker, contestID, connectionID, redisChannel, log)
}

// ignore incoming messages
func (s *websocketService) handleIncomingMessages(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

// main event loop
func (s *websocketService) handleOutgoingMessages(
	ctx context.Context,
	conn *websocket.Conn,
	pingChecker *time.Ticker,
	jwtChecker *time.Ticker,
	contestID uuid.UUID,
	connectionID uuid.UUID,
	redisChannel <-chan *redis.Message,
	log *slog.Logger,
) {
	for {
		select {
		// forward redis updates to websocket client
		case msg := <-redisChannel:
			var updateData model.WSUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
				log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
			}

			if err := sendWebSocketMessage(conn, log, &updateData); err != nil {
				log.Error("failed to send redis message to websocket client", "error", err)
			}

		// send periodic ping to keep connection alive
		case <-pingChecker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("failed to send ping", "error", err)
				_ = conn.Close()
				return
			}

		// validate jwt token periodically
		case <-jwtChecker.C:
			if shouldCloseConnection(ctx, log) {
				log.Warn("closing connection due to token validation failure")
				if err := sendWebSocketMessage(conn, log, model.NewDisconnectedMessage(contestID, connectionID)); err != nil {
					log.Error("failed to send disconnected message", "error", err)
				}
				_ = conn.Close()
				return
			}
		// handle client disconnection
		case <-ctx.Done():
			log.Info("websocket client disconnected")
			return
		}
	}
}

func sendWebSocketMessage(conn *websocket.Conn, log *slog.Logger, data *model.WSUpdate) error {
	// marshal update to json
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("failed to marshal websocket message", "error", err, "type", data.Type)
		return err
	}

	// set write deadline and send message to client
	_ = conn.SetWriteDeadline(time.Now().Add(writeDeadline))
	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		log.Error("failed to write websocket message", "error", err, "type", data.Type)
		return err
	}

	log.Info("sent websocket message",
		"ws_type", data.Type,
		"ws_contest_id", data.ContestID,
		"ws_updated_by", data.UpdatedBy,
		"ws_timestamp", data.Timestamp,
		"ws_square", data.Square,
		"ws_contest", data.Contest,
		"ws_quarter_result", data.QuarterResult,
	)
	return nil
}

func shouldCloseConnection(ctx context.Context, log *slog.Logger) bool {
	// get claims from context
	claims, ok := ctx.Value(model.ClaimsKey).(*model.Claims)
	if !ok || claims == nil {
		log.Warn("claims not in ctx for websocket connection")
		return true
	}

	// check if token is expired
	now := time.Now().Unix()
	if claims.Expire < now {
		log.Info("token expired for websocket connection", "expire", claims.Expire, "now", now)
		return true
	}

	return false
}
