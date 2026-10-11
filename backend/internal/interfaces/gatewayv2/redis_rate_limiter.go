package gatewayv2

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisSlidingWindowLua is an atomic Lua script executing sliding-window rate limiting.
// It removes timestamps older than now - window, counts remaining entries,
// and if count < limit, records the current request timestamp with a unique nonce.
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
    return {1, limit - currentCount - 1}
else
    return {0, 0}
end
`

// RedisRateLimiter implements atomic, distributed sliding-window rate limiting via Redis.
// When Redis is unavailable, it safely falls back to the in-memory token bucket.
type RedisRateLimiter struct {
	client   *redis.Client
	script   *redis.Script
	fallback *RateLimiterImpl
	logger   *slog.Logger
	clock    func() time.Time
	seq      uint64
}

// NewRedisRateLimiter creates a Redis-backed atomic rate limiter.
func NewRedisRateLimiter(client *redis.Client, logger *slog.Logger) *RedisRateLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisRateLimiter{
		client:   client,
		script:   redis.NewScript(redisSlidingWindowLua),
		fallback: NewRateLimiter(),
		logger:   logger,
		clock:    time.Now,
	}
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
		return r.fallback.Check(ctx, key, limit, window)
	}

	now := r.clock()
	nowMs := now.UnixMilli()
	windowMs := window.Milliseconds()
	nonce := fmt.Sprintf("%d-%d", nowMs, atomic.AddUint64(&r.seq, 1))
	redisKey := "aigw:ratelimit:" + key

	res, err := r.script.Run(ctx, r.client, []string{redisKey}, nowMs, windowMs, limit, nonce).Slice()
	if err != nil {
		r.logger.Warn("redis rate limiter script failed, falling back to in-memory", "key", key, "error", err)
		return r.fallback.Check(ctx, key, limit, window)
	}

	allowedInt, _ := res[0].(int64)
	remainingInt, _ := res[1].(int64)
	allowed := allowedInt == 1

	resetAt := now.Add(window)
	var retryAfter time.Duration
	if !allowed {
		retryAfter = window / time.Duration(limit)
		if retryAfter <= 0 {
			retryAfter = time.Second
		}
	}

	return RateLimitResult{
		Allowed:    allowed,
		Limit:      limit,
		Remaining:  int(remainingInt),
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
		Key:        key,
	}
}
