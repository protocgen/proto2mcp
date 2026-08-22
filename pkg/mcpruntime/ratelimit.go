package mcpruntime

import (
	"context"
	"math"
	"sync"
	"time"
)

const maxBuckets = 10000

type bucketKey struct {
	tenant string
	tool   string
}

// RateLimiter is a per-tool token bucket rate limiter middleware.
// It limits the rate of tool calls per tenant.
//
// EXPERIMENTAL: This API may change.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[bucketKey]*tokenBucket
	rate    float64 // tokens per second
	burst   int     // max tokens (burst capacity)
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

// NewRateLimiter creates a rate limiter with the given rate (calls/sec) and burst.
func NewRateLimiter(ratePerSecond float64, burst int) *RateLimiter {
	if math.IsNaN(ratePerSecond) || math.IsInf(ratePerSecond, 0) || ratePerSecond < 0 {
		panic("mcpruntime: NewRateLimiter requires a finite non-negative rate")
	}
	if burst <= 0 {
		panic("mcpruntime: NewRateLimiter requires a positive burst")
	}
	return &RateLimiter{
		buckets: make(map[bucketKey]*tokenBucket),
		rate:    ratePerSecond,
		burst:   burst,
	}
}

// evictStale removes buckets idle for more than 5 minutes.
func (r *RateLimiter) evictStale(now time.Time) {
	staleThreshold := now.Add(-5 * time.Minute)
	for k, b := range r.buckets {
		if b.last.Before(staleThreshold) {
			delete(r.buckets, k)
		}
	}
}

func (r *RateLimiter) allow(key bucketKey) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	b, ok := r.buckets[key]
	if !ok {
		// Evict stale entries if at capacity
		if len(r.buckets) >= maxBuckets {
			r.evictStale(now)
		}
		b = &tokenBucket{tokens: float64(r.burst), last: now}
		r.buckets[key] = b
	}

	// Replenish tokens
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * r.rate
	if b.tokens > float64(r.burst) {
		b.tokens = float64(r.burst)
	}
	b.last = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// HandleToolCall implements ToolInterceptor.
func (r *RateLimiter) HandleToolCall(ctx context.Context, req ToolRequest, next HandlerFunc) (*CallToolResult, error) {
	// Rate limit by tenant+tool
	key := bucketKey{tenant: TenantFromContext(ctx), tool: req.ToolName}
	if !r.allow(key) {
		return NewErrorResultWithDetails("rate limit exceeded, please slow down", "RESOURCE_EXHAUSTED", nil), nil
	}
	return next(ctx, req)
}

// FilterTools implements DiscoveryInterceptor (no-op — rate limiting doesn't affect discovery).
func (r *RateLimiter) FilterTools(_ context.Context, tools []ToolDefinition) []ToolDefinition {
	return tools
}

var _ Middleware = (*RateLimiter)(nil)
