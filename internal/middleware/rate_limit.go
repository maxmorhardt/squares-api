package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
)

const (
	rateLimitWindow       = 24 * time.Hour
	rateLimitCleanupEvery = time.Hour
)

type ipCounter struct {
	count       int
	windowStart time.Time
	lastSeen    time.Time
}

type rateLimiter struct {
	mu          sync.Mutex
	entries     map[string]*ipCounter
	limit       int
	lastCleanup time.Time
}

func newRateLimiter(limit int) *rateLimiter {
	return &rateLimiter{
		entries:     make(map[string]*ipCounter),
		limit:       limit,
		lastCleanup: time.Now(),
	}
}

func ContactRateLimitMiddleware(requestsPerDay int) gin.HandlerFunc {
	if requestsPerDay <= 0 {
		requestsPerDay = 1
	}

	rl := newRateLimiter(requestsPerDay)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.NewAPIError(http.StatusTooManyRequests, "Too many requests", c))
			return
		}
		c.Next()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	if now.Sub(rl.lastCleanup) > rateLimitCleanupEvery {
		for k, e := range rl.entries {
			if now.Sub(e.lastSeen) > rateLimitWindow {
				delete(rl.entries, k)
			}
		}
		rl.lastCleanup = now
	}

	e, ok := rl.entries[ip]
	if !ok || now.Sub(e.windowStart) >= rateLimitWindow {
		rl.entries[ip] = &ipCounter{count: 1, windowStart: now, lastSeen: now}
		return true
	}

	e.lastSeen = now
	if e.count >= rl.limit {
		return false
	}

	e.count++
	return true
}
