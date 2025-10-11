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
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/middleware"
	"github.com/maxmorhardt/squares-api/internal/model"
	"github.com/maxmorhardt/squares-api/internal/repository"
)

// @Summary Connect to Server-Sent Events stream for real-time grid updates
// @Description Establishes a persistent SSE connection to receive real-time updates for a specific grid
// @Tags events
// @Produce text/event-stream
// @Param gridId path string true "Grid ID to listen for updates" format(uuid)
// @Success 200 {object} model.GridChannelResponse
// @Failure 400 {object} model.APIError
// @Failure 401 {object} model.APIError
// @Security BearerAuth
// @Router /events/{gridId} [get]
func SSEHandler(c *gin.Context) {
	log := middleware.FromContext(c)

	claims, gridId := validateSSERequest(c, log)
	if claims == nil || gridId == uuid.Nil {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	log.Info("sending connected message", "user", claims.Username, "gridId", gridId)
	if err := sendSSEMessage(c, log, model.NewConnectedMessage(gridId, claims.Username)); err != nil {
		log.Error("failed to send connected message", "error", err)
		return
	}

	handleSSEConnection(c, log, gridId, claims.Username)
}

func validateSSERequest(c *gin.Context, log *slog.Logger) (*model.Claims, uuid.UUID) {
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

	log.Info("sse client validated", "user", claims.Username, "gridId", gridId)
	return claims, gridId
}

func handleSSEConnection(c *gin.Context, log *slog.Logger, gridId uuid.UUID, username string) {
	ctx := c.Request.Context()

	gridChannel := fmt.Sprintf("%s:%s", model.GridChannelPrefix, gridId.String())
	log.Info("subscribing to redis channel", "channel", gridChannel)

	pubsub := config.RedisClient.Subscribe(ctx, gridChannel)
	defer func() {
		log.Info("closing redis subscription", "channel", gridChannel)
		pubsub.Close()
	}()

	redisChannel := pubsub.Channel()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-redisChannel:
			log.Info("received redis message", "channel", msg.Channel)
			if err := handleRedisMessage(c, log, msg); err != nil {
				log.Warn("failed to handle redis message - closing connection", "error", err)
				return
			}

		case <-ticker.C:
			if err := sendSSEMessage(c, log, model.NewKeepAliveMessage(gridId)); err != nil {
				log.Warn("failed to send keepalive - closing connection", "error", err)
				return
			}

		case <-ctx.Done():
			log.Info("sse client disconnected", "user", username, "gridId", gridId)
			return
		}
	}
}

func handleRedisMessage(c *gin.Context, log *slog.Logger, msg *redis.Message) error {
	var updateData model.GridChannelResponse
	if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
		log.Error("failed to unmarshal redis message", "error", err, "payload", msg.Payload)
		return nil
	}

	log.Info("sending redis update to client", "type", updateData.Type, "gridId", updateData.GridID, "cellId", updateData.CellID)
	return sendSSEMessage(c, log, &updateData)
}

func sendSSEMessage(c *gin.Context, log *slog.Logger, data *model.GridChannelResponse) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error("failed to marshal sse message", "error", err, "type", data.Type)
		return err
	}

	_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
	if err != nil {
		log.Error("failed to write sse message", "error", err)
		return err
	}

	c.Writer.Flush()
	log.Debug("sse message sent successfully", "type", data.Type, "size", len(jsonData))
	return err
}
