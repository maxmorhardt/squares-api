package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/squares-api/internal/model"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newRateLimiter(r rate.Limit, b int) *rateLimiter {
	rl := &rateLimiter{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, l := range rl.limiters {
			if time.Since(l.lastSeen) > 24*time.Hour {
				delete(rl.limiters, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limiters[ip]
	if !ok {
		l = &ipLimiter{limiter: rate.NewLimiter(rl.r, rl.b)}
		rl.limiters[ip] = l
	}
	l.lastSeen = time.Now()
	return l.limiter
}

var contactRateLimiter = newRateLimiter(rate.Every(24*time.Hour/10), 10)

func ContactRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !contactRateLimiter.get(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, model.NewAPIError(http.StatusTooManyRequests, "Too many requests", c))
			return
		}
		c.Next()
	}
}
