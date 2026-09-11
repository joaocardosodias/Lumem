package http

import (
	"testing"
	"time"
)

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    1.0,
		burst:   2.0,
	}

	rl.buckets["old-ip"] = &tokenBucket{
		tokens:     1.0,
		capacity:   2.0,
		rate:       1.0,
		lastRefill: time.Now().Add(-15 * time.Minute),
	}

	rl.buckets["fresh-ip"] = &tokenBucket{
		tokens:     1.0,
		capacity:   2.0,
		rate:       1.0,
		lastRefill: time.Now(),
	}

	rl.cleanup()

	if _, exists := rl.buckets["old-ip"]; exists {
		t.Fatalf("esperava que old-ip fosse removido no cleanup")
	}

	if _, exists := rl.buckets["fresh-ip"]; !exists {
		t.Fatalf("esperava que fresh-ip fosse mantido no cleanup")
	}
}

func TestRateLimiter_AllowBranches(t *testing.T) {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    10.0,
		burst:   5.0,
	}

	if !rl.allow("1.1.1.1") {
		t.Fatalf("esperava allow true para novo ip")
	}

	rl.mu.Lock()
	rl.buckets["1.1.1.1"].tokens = 0.5
	rl.buckets["1.1.1.1"].lastRefill = time.Now()
	rl.mu.Unlock()

	if rl.allow("1.1.1.1") {
		t.Fatalf("esperava allow false quando tokens insuficientes")
	}

	rl.mu.Lock()
	rl.buckets["1.1.1.1"].tokens = 1.0
	rl.buckets["1.1.1.1"].lastRefill = time.Now().Add(-10 * time.Second)
	rl.mu.Unlock()

	if !rl.allow("1.1.1.1") {
		t.Fatalf("esperava allow true apos refill")
	}
}
