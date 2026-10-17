package gatewayv2

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiterMode specifies how RedisRateLimiter behaves when Redis is unavailable.
type RateLimiterMode string

const (
	// ModeDegradedLocal falls back to the in-memory token bucket on Redis outage.
	// Sets result.Degraded = true and logs a warning.
	// GUARANTEE: High availability; limits abuse per node.
	// LIMITATION: In a multi-node deployment, total cluster throughput can equal
	// limit * number of running nodes until Redis connectivity is restored.
	ModeDegradedLocal RateLimiterMode = "degraded_local"

	// ModeFailClosed denies all rate-limited requests when Redis is unavailable.
	// Sets result.Degraded = true and result.Allowed = false.
	// GUARANTEE: Strict cluster-wide rate limit enforcement; limit is never exceeded.
	// LIMITATION: Gateway rejects requests if Redis has a cluster or network outage.
	ModeFailClosed RateLimiterMode = "fail_closed"
)

// redisSlidingWindowLua is an atomic Lua script executing sliding-window rate limiting.
// It removes timestamps older than now - window, counts remaining entries,
// and if count < limit, records the current request timestamp with a unique nonce.
// When over limit, it refreshes key expiration and returns the oldest timestamp to compute exact Retry-After.
const redisSlidingWindowLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local nonce = ARGV[4]

local clearBefore = now - window
redis.call('ZREMRANGEBYSCORE', key, 0, clearBefore)
local currentCount = redis.call('ZCARD', key)

if currentCount < limit then
    redis.call('ZADD', key, now, nonce)
    redis.call('PEXPIRE', key, window)
    return {1, limit - currentCount - 1, 0}
else
    -- Refresh TTL so the key is not prematurely evicted during sustained throttling
    redis.call('PEXPIRE', key, window)
    local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    local oldestScore = 0
    if oldest and #oldest >= 2 then
        oldestScore = tonumber(oldest[2])
    end
    return {0, 0, oldestScore}
end
`

// RedisRateLimiter implements atomic, distributed sliding-window rate limiting via Redis.
// Guarantees:
// - Single-source distributed sliding window across all gateway instances when Redis is active.
// - Mathematical Retry-After calculation based on the oldest entry sliding out of the window.
// - Explicit failure handling: either ModeDegradedLocal (documented node-local fallback) or ModeFailClosed.
type RedisRateLimiter struct {
	client   *redis.Client
	script   *redis.Script
	fallback *RateLimiterImpl
	logger   *slog.Logger
	clock    func() time.Time
	mode     RateLimiterMode
	seq      uint64
}

// NewRedisRateLimiter creates a Redis-backed atomic rate limiter.
func NewRedisRateLimiter(client *redis.Client, logger *slog.Logger) *RedisRateLimiter {
	if logger == nil {
		logger = slog.Default()
	}

	mode := ModeDegradedLocal
	if os.Getenv("AIGW_RATE_LIMIT_FAIL_CLOSED") == "true" {
		mode = ModeFailClosed
	}

	return &RedisRateLimiter{
		client:   client,
		script:   redis.NewScript(redisSlidingWindowLua),
		fallback: NewRateLimiter(),
		logger:   logger,
		clock:    time.Now,
		mode:     mode,
	}
}

// WithMode configures the Redis outage handling mode.
func (r *RedisRateLimiter) WithMode(mode RateLimiterMode) *RedisRateLimiter {
	r.mode = mode
	return r
}

// Check evaluates whether the request is allowed under the rate limit.
func (r *RedisRateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) RateLimitResult {
	if limit <= 0 {
		return RateLimitResult{
			Allowed:   true,
			Limit:     limit,
			Remaining: 0,
			ResetAt:   r.clock().Add(window),
			Key:       key,
		}
	}

	if r.client == nil {
		return r.handleRedisOutage(ctx, key, limit, window, fmt.Errorf("redis client unconfigured"))
	}

	now := r.clock()
	nowMs := now.UnixMilli()
	windowMs := window.Milliseconds()
	nonce := fmt.Sprintf("%d-%d", nowMs, atomic.AddUint64(&r.seq, 1))
	redisKey := "aigw:ratelimit:" + key

	res, err := r.script.Run(ctx, r.client, []string{redisKey}, nowMs, windowMs, limit, nonce).Slice()
	if err != nil {
		return r.handleRedisOutage(ctx, key, limit, window, err)
	}

	allowedInt, _ := res[0].(int64)
	remainingInt, _ := res[1].(int64)
	oldestMs, _ := res[2].(int64)
	allowed := allowedInt == 1

	var retryAfter time.Duration
	resetAt := now.Add(window)
	if !allowed {
		if oldestMs > 0 {
			waitMs := (oldestMs + windowMs) - nowMs
			if waitMs > 0 {
				retryAfter = time.Duration(waitMs) * time.Millisecond
			}
		}
		if retryAfter <= 0 {
			retryAfter = window / time.Duration(limit)
			if retryAfter <= 0 {
				retryAfter = time.Second
			}
		}
		resetAt = now.Add(retryAfter)
	}

	return RateLimitResult{
		Allowed:    allowed,
		Limit:      limit,
		Remaining:  int(remainingInt),
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
		Key:        key,
		Degraded:   false,
	}
}

func (r *RedisRateLimiter) handleRedisOutage(ctx context.Context, key string, limit int, window time.Duration, err error) RateLimitResult {
	if r.mode == ModeFailClosed {
		r.logger.Error("redis rate limiter unavailable: FAILING CLOSED (request rejected)",
			"key", key, "error", err)
		now := r.clock()
		return RateLimitResult{
			Allowed:    false,
			Limit:      limit,
			Remaining:  0,
			ResetAt:    now.Add(5 * time.Second),
			RetryAfter: 5 * time.Second,
			Key:        key,
			Degraded:   true,
		}
	}

	// ModeDegradedLocal: Documented fallback to process-local limiter
	r.logger.Warn("redis rate limiter unavailable: operating in DEGRADED node-local mode; limits are not cluster-wide",
		"key", key, "error", err)
	res := r.fallback.Check(ctx, key, limit, window)
	res.Degraded = true
	return res
}
