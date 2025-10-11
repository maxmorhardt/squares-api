package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/config"
	"github.com/maxmorhardt/squares-api/internal/middleware"
)

func SSEHandler(c *gin.Context) {
	log := middleware.FromContext(c)
	claims := middleware.VerifyToken(c, config.OIDCVerifier, log)
	if claims == nil {
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	welcomeMsg := map[string]interface{}{
		"type":      "connected",
		"message":   "Connected to updates",
		"user":      claims.Username,
		"timestamp": time.Now().Unix(),
	}
	sendSSEMessage(c, welcomeMsg)

	ctx := c.Request.Context()
	pubsub := config.RedisClient.Subscribe(ctx, "grid-updates")
	defer pubsub.Close()

	ch := pubsub.Channel()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg := <-ch:
			var updateData map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &updateData); err != nil {
				log.Error("Failed to unmarshal Redis message", "error", err)
				continue
			}

			if !sendSSEMessage(c, updateData) {
				return
			}

		case <-ticker.C:
			keepalive := map[string]interface{}{
				"type":      "keepalive",
				"timestamp": time.Now().Unix(),
			}
			if !sendSSEMessage(c, keepalive) {
				return
			}

		case <-ctx.Done():
			log.Info("SSE client disconnected", "user", claims.Username)
			return
		}
	}
}

func sendSSEMessage(c *gin.Context, data any) bool {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false
	}

	_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", string(jsonData))
	if err != nil {
		return false
	}

	c.Writer.Flush()
	return true
}
