package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
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

	log.Info("subscribing to redis channel")
	contestChannel := fmt.Sprintf("%s:%s", model.ContestChannelPrefix, contestID)
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

	go s.handleIncomingMessages(conn)
	s.handleOutgoingMessages(ctx, conn, pingChecker, jwtChecker, contestID, redisChannel, log)
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
	redisChannel <-chan *redis.Message,
	log *slog.Logger,
) {
	for {
		select {
		case msg := <-redisChannel:
			var updateData model.WSUpdate
			if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
				log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
			}

			if err := sendWebSocketMessage(conn, log, &updateData); err != nil {
				log.Error("failed to send redis message to websocket client", "error", err)
			}

		case <-pingChecker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn("failed to send ping", "error", err)
			}

		case <-jwtChecker.C:
			if shouldCloseConnection(ctx, log) {
				log.Warn("closing connection due to token validation failure")
				if err := sendWebSocketMessage(conn, log, model.NewDisconnectedMessage(contestID)); err != nil {
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

func shouldCloseConnection(ctx context.Context, log *slog.Logger) bool {
	claims, ok := ctx.Value(model.ClaimsKey).(*model.Claims)
	if !ok || claims == nil {
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
