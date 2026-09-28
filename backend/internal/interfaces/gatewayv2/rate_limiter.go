// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"sync"
	"time"
)

// RateLimiterImpl implements a token bucket rate limiter with sliding window.
type RateLimiterImpl struct {
	mu              sync.RWMutex
	buckets         map[string]*tokenBucket
	clock           func() time.Time
	cleanupInterval time.Duration
	stopCh          chan struct{}
}

type tokenBucket struct {
	key        string
	limit      int
	window     time.Duration
	tokens     float64
	lastRefill time.Time
	refillRate float64 // tokens per second
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiterImpl {
	rl := &RateLimiterImpl{
		buckets:         make(map[string]*tokenBucket),
		clock:           time.Now,
		cleanupInterval: 5 * time.Minute,
		stopCh:          make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Check performs a rate limit check and consumes a token if allowed.
func (r *RateLimiterImpl) Check(ctx context.Context, key string, limit int, window time.Duration) RateLimitResult {
	bucket := r.getOrCreateBucket(key, limit, window)
	return bucket.take()
}

// getOrCreateBucket retrieves or creates a token bucket for the key.
func (r *RateLimiterImpl) getOrCreateBucket(key string, limit int, window time.Duration) *tokenBucket {
	r.mu.RLock()
	bucket, ok := r.buckets[key]
	r.mu.RUnlock()

	if ok {
		return bucket
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, ok := r.buckets[key]; ok {
		return bucket
	}

	refillRate := float64(limit) / window.Seconds()
	bucket = &tokenBucket{
		key:        key,
		limit:      limit,
		window:     window,
		tokens:     float64(limit),
		lastRefill: time.Now(),
		refillRate: refillRate,
	}
	r.buckets[key] = bucket
	return bucket
}

// take attempts to consume one token from the bucket.
func (b *tokenBucket) take() RateLimitResult {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	b.tokens += b.refillRate * elapsed.Seconds()
	if b.tokens > float64(b.limit) {
		b.tokens = float64(b.limit)
	}
	b.lastRefill = now

	allowed := b.tokens >= 1
	var remaining int
	if allowed {
		b.tokens--
		remaining = int(b.tokens)
	} else {
		remaining = 0
	}

	resetAt := now.Add(b.window)
	retryAfter := time.Duration(0)
	if !allowed {
		retryAfter = time.Duration((1 - b.tokens) / b.refillRate * float64(time.Second))
	}

	return RateLimitResult{
		Allowed:    allowed,
		Limit:      b.limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
		Scope:      "credential", // default, overridden by caller
		Key:        b.key,
	}
}

// cleanupLoop periodically removes idle buckets.
func (r *RateLimiterImpl) cleanupLoop() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopCh:
			return
		}
	}
}

// cleanup removes buckets that haven't been used recently.
func (r *RateLimiterImpl) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for key, bucket := range r.buckets {
		bucket.mu.Lock()
		idle := now.Sub(bucket.lastRefill)
		bucket.mu.Unlock()

		if idle > 10*time.Minute {
			delete(r.buckets, key)
		}
	}
}

// Close stops the rate limiter.
func (r *RateLimiterImpl) Close() {
	close(r.stopCh)
}

// SlidingWindowRateLimiter implements a sliding window rate limiter
// using a sorted set of request timestamps (more accurate for burst handling).
type SlidingWindowRateLimiter struct {
	mu      sync.RWMutex
	windows map[string]*slidingWindow
	clock   func() time.Time
}

type slidingWindow struct {
	key      string
	limit    int
	window   time.Duration
	requests []time.Time
	mu       sync.Mutex
}

// NewSlidingWindowRateLimiter creates a new sliding window rate limiter.
func NewSlidingWindowRateLimiter() *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		windows: make(map[string]*slidingWindow),
		clock:   time.Now,
	}
}

// Check performs a sliding window rate limit check.
func (s *SlidingWindowRateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) RateLimitResult {
	s.mu.RLock()
	w, ok := s.windows[key]
	s.mu.RUnlock()

	if !ok {
		s.mu.Lock()
		w = &slidingWindow{key: key, limit: limit, window: window}
		s.windows[key] = w
		s.mu.Unlock()
	}

	return w.check(limit, window, s.clock)
}

// check performs the sliding window check.
func (w *slidingWindow) check(limit int, window time.Duration, clock func() time.Time) RateLimitResult {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := clock()
	cutoff := now.Add(-window)

	// Remove expired requests
	valid := make([]time.Time, 0, len(w.requests))
	for _, t := range w.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	w.requests = valid

	allowed := len(w.requests) < limit
	remaining := limit - len(w.requests)
	if remaining < 0 {
		remaining = 0
	}

	if allowed {
		w.requests = append(w.requests, now)
	}

	resetAt := now.Add(window)
	var retryAfter time.Duration
	if !allowed && len(w.requests) > 0 {
		retryAfter = w.requests[0].Add(window).Sub(now)
	}

	return RateLimitResult{
		Allowed:    allowed,
		Limit:      limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}
}

// cleanupLoop periodically cleans up old windows.
func (s *SlidingWindowRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := s.clock()
		cutoff := now.Add(-10 * time.Minute)
		for key, w := range s.windows {
			w.mu.Lock()
			if len(w.requests) == 0 || w.requests[len(w.requests)-1].Before(cutoff) {
				delete(s.windows, key)
			}
			w.mu.Unlock()
		}
		s.mu.Unlock()
	}
}
