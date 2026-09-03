package http

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type tokenBucket struct {
	tokens     float64
	capacity   float64
	rate       float64
	lastRefill time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rate    float64
	burst   float64
}

func NewRateLimiter(rate float64, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    rate,
		burst:   float64(burst),
	}

	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[ip]
	if !exists {
		rl.buckets[ip] = &tokenBucket{
			tokens:     rl.burst - 1,
			capacity:   rl.burst,
			rate:       rl.rate,
			lastRefill: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, b := range rl.buckets {
		if b.lastRefill.Before(cutoff) {
			delete(rl.buckets, ip)
		}
	}
}

func RateLimiterMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.Header("Retry-After", "1")
			RespondProblem(c, http.StatusTooManyRequests, "ERR_RATE_LIMIT_EXCEEDED", "Limite de Requisicoes Excedido", "Voce ultrapassou a taxa permitida de requisicoes. Tente novamente em breve.")
			return
		}
		c.Next()
	}
}
