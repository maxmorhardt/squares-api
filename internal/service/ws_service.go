package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/util"
	"github.com/nats-io/nats.go"
)

const (
	pingInterval      = 30 * time.Second
	pongTimeout       = 60 * time.Second
	writeDeadline     = 10 * time.Second
	jwtCheckInterval  = 5 * time.Minute
	natsCheckInterval = 10 * time.Second
	maxChatMessageLen = 255
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

	// subscribe to NATS subject for contest updates before notifying client
	log.Info("subscribing to NATS subject")
	contestSubject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID.String())

	natsChan := make(chan *nats.Msg, 64)
	natsConn := config.NATS()
	if natsConn == nil || !natsConn.IsConnected() {
		log.Error("NATS connection not available")
		_ = conn.Close()
		return
	}

	sub, err := natsConn.ChanSubscribe(contestSubject, natsChan)
	if err != nil {
		log.Error("failed to subscribe to NATS", "error", err)
		return
	}
	
	defer func() {
		log.Info("closing NATS subscription")
		if err := sub.Unsubscribe(); err != nil {
			log.Error("failed to unsubscribe from NATS", "error", err)
		}
	}()

	// send connected message only after NATS subscription is established
	if err := sendWebSocketMessage(conn, log, model.NewConnectedMessage(contestID, connectionID)); err != nil {
		log.Error("failed to send connected message", "error", err)
		return
	}

	// setup ping/pong to keep connection alive
	pingChecker := time.NewTicker(pingInterval)
	defer pingChecker.Stop()

	// set read limit to prevent oversized frames
	conn.SetReadLimit(1024)

	// set read deadline and pong handler
	_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	// setup jwt token validation checker
	jwtChecker := time.NewTicker(jwtCheckInterval)
	defer jwtChecker.Stop()

	// setup NATS connection checker
	natsChecker := time.NewTicker(natsCheckInterval)
	defer natsChecker.Stop()

	// start message handlers
	go s.handleIncomingMessages(ctx, conn, contestID, log)
	s.handleOutgoingMessages(ctx, conn, pingChecker, jwtChecker, natsChecker, contestID, connectionID, natsChan, log, sub)
}

// handle incoming messages from websocket client
func (s *websocketService) handleIncomingMessages(ctx context.Context, conn *websocket.Conn, contestID uuid.UUID, log *slog.Logger) {
	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var chatMsg model.WSChatMessage
		if err := json.Unmarshal(rawMsg, &chatMsg); err != nil {
			log.Warn("failed to unmarshal incoming ws message", "error", err)
			continue
		}

		s.handleChatMessage(ctx, contestID, chatMsg.Message, log)
	}
}

func (s *websocketService) handleChatMessage(ctx context.Context, contestID uuid.UUID, message string, log *slog.Logger) {
	message = strings.TrimSpace(message)
	if message == "" || len(message) > maxChatMessageLen {
		return
	}

	if !util.IsSafeString(message) {
		log.Warn("chat message contains unsafe characters")
		return
	}

	claims, ok := ctx.Value(model.ClaimsKey).(*model.Claims)
	if !ok || claims == nil {
		log.Warn("no claims in context for chat message")
		return
	}

	chatMsg := model.NewChatMessage(contestID, claims.Username, message)
	jsonData, err := json.Marshal(chatMsg)
	if err != nil {
		log.Error("failed to marshal chat message", "error", err)
		return
	}

	contestSubject := fmt.Sprintf("%s.%s", model.ContestChannelPrefix, contestID.String())
	natsConn := config.NATS()
	if natsConn == nil || !natsConn.IsConnected() {
		log.Warn("NATS not available for chat message")
		return
	}

	if err := natsConn.Publish(contestSubject, jsonData); err != nil {
		log.Error("failed to publish chat message to NATS", "error", err)
		return
	}

	log.Info("chat message sent", "sender", claims.Username, "message", message)
}

// main event loop
func (s *websocketService) handleOutgoingMessages(
	ctx context.Context,
	conn *websocket.Conn,
	pingChecker *time.Ticker,
	jwtChecker *time.Ticker,
	natsChecker *time.Ticker,
	contestID uuid.UUID,
	connectionID uuid.UUID,
	natsChan <-chan *nats.Msg,
	log *slog.Logger,
	sub *nats.Subscription,
) {
	for {
		select {
		// forward NATS updates to websocket client
		case msg, ok := <-natsChan:
			if !ok {
				log.Warn("NATS channel closed, closing websocket connection")
				if err := sendWebSocketMessage(conn, log, model.NewDisconnectedMessage(contestID, connectionID)); err != nil {
					log.Error("failed to send disconnected message", "error", err)
				}
				_ = conn.Close()
				return
			}

			var updateData model.WSUpdate
			if err := json.Unmarshal(msg.Data, &updateData); err != nil {
				log.Error("failed to unmarshal NATS message", "error", err, "data", string(msg.Data))
				continue
			}

			if err := sendWebSocketMessage(conn, log, &updateData); err != nil {
				log.Error("failed to send NATS message to websocket client", "error", err)
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

		// check NATS connection periodically
		case <-natsChecker.C:
			natsConn := config.NATS()
			if natsConn == nil || !natsConn.IsConnected() || !sub.IsValid() {
				log.Warn("NATS connection lost, closing websocket")
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

	log.Info("sent websocket message", "ws_type", data.Type)
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
