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

// CategoryRateLimiter manages multiple independent rate limiters by category.
type CategoryRateLimiter struct {
	limiters map[string]*RateLimiter
}

// RateCategoryConfig defines the limit and window for a rate limit category.
type RateCategoryConfig struct {
	Max    int
	Window time.Duration
}

// NewCategoryRateLimiter creates a set of categorized rate limiters.
func NewCategoryRateLimiter(categories map[string]RateCategoryConfig) *CategoryRateLimiter {
	limiters := make(map[string]*RateLimiter, len(categories))
	for name, cfg := range categories {
		limiters[name] = NewRateLimiter(cfg.Max, cfg.Window)
	}
	return &CategoryRateLimiter{limiters: limiters}
}

// LimitCategory returns a middleware handler for a specific category.
// The key function uses user ID if available, falling back to IP.
func (c *CategoryRateLimiter) LimitCategory(category string) func(e *core.RequestEvent) error {
	limiter, ok := c.limiters[category]
	if !ok {
		// No limiter for this category, pass through.
		return func(e *core.RequestEvent) error { return e.Next() }
	}

	return func(e *core.RequestEvent) error {
		// Use authenticated user ID if available, otherwise IP.
		key := e.RealIP()
		if e.Auth != nil {
			key = e.Auth.Id
		}

		now := time.Now()

		limiter.mu.Lock()
		timestamps := limiter.requests[key]

		cutoff := now.Add(-limiter.window)
		valid := timestamps[:0]
		for _, t := range timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}

		if len(valid) >= limiter.max {
			limiter.requests[key] = valid
			limiter.mu.Unlock()

			retryAfter := valid[0].Add(limiter.window).Sub(now)
			e.Response.Header().Set("Retry-After", retryAfter.Truncate(time.Second).String())

			return e.JSON(http.StatusTooManyRequests, map[string]string{
				"error": "rate limit exceeded, try again later",
			})
		}

		limiter.requests[key] = append(valid, now)
		limiter.mu.Unlock()

		return e.Next()
	}
}

// Cleanup removes stale entries from all category limiters.
func (c *CategoryRateLimiter) Cleanup() {
	for _, rl := range c.limiters {
		rl.Cleanup()
	}
}
