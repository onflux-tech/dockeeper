package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// RateLimiter implements a fixed-window IP-based rate limiter.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	max      int
	window   time.Duration
}

// NewRateLimiter creates a rate limiter allowing max requests per window.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

// Limit returns a middleware handler that enforces the rate limit.
func (rl *RateLimiter) Limit(e *core.RequestEvent) error {
	ip := e.RealIP()
	now := time.Now()

	rl.mu.Lock()
	timestamps := rl.requests[ip]

	// Remove entries outside the window.
	cutoff := now.Add(-rl.window)
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.max {
		rl.requests[ip] = valid
		rl.mu.Unlock()

		retryAfter := valid[0].Add(rl.window).Sub(now)
		e.Response.Header().Set("Retry-After", retryAfter.Truncate(time.Second).String())

		return e.JSON(http.StatusTooManyRequests, map[string]string{
			"error": "rate limit exceeded, try again later",
		})
	}

	rl.requests[ip] = append(valid, now)
	rl.mu.Unlock()

	return e.Next()
}

// Cleanup removes stale entries. Call periodically via a cron job.
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-rl.window)
	for ip, timestamps := range rl.requests {
		valid := timestamps[:0]
		for _, t := range timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = valid
		}
	}
}
