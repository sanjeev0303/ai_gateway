// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// BudgetGuardMode specifies how BudgetGuard behaves when Redis is unavailable.
type BudgetGuardMode string

const (
	// BudgetModeDegradedLocal falls back to the in-memory tracker on Redis outage.
	// Sets result.Degraded = true and logs a warning.
	BudgetModeDegradedLocal BudgetGuardMode = "degraded_local"

	// BudgetModeFailClosed denies all budget-sensitive requests when Redis is unavailable.
	// Sets result.Degraded = true and result.Allowed = false.
	// GUARANTEE: Cluster-wide budget caps are strictly enforced; limits cannot be exceeded under outage.
	BudgetModeFailClosed BudgetGuardMode = "fail_closed"
)

// SpendLoader loads historical spend for a given budget key within the active window [windowStart, now].
// Used during cold start or gateway restart to reconcile in-memory tracker with persisted ledger.
type SpendLoader func(ctx context.Context, key string, windowStart time.Time) (float64, error)

type budgetReservation struct {
	reqID     string
	amount    float64
	expiresAt time.Time
}

// computeWindowStart calculates the deterministic start of the active budget window in UTC.
// Supported windows:
// - Daily (24h or 86400s): Midnight 00:00:00 UTC of current day.
// - Monthly (~30d or 2592000s): First day of month 00:00:00 UTC.
// - Weekly (7d or 604800s): Monday 00:00:00 UTC of current week.
// - Hourly / Arbitrary Duration: Truncated to the window duration.
// All resets are strictly calculated in UTC to guarantee deterministic boundaries across distributed nodes.
func computeWindowStart(now time.Time, window time.Duration) time.Time {
	now = now.UTC()
	if window <= 0 {
		return time.Time{}
	}
	if window == 24*time.Hour || window == 86400*time.Second {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	if window == 30*24*time.Hour || window == 2592000*time.Second {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if window == 7*24*time.Hour || window == 604800*time.Second {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday -> 7
		}
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return startOfDay.AddDate(0, 0, -(weekday - 1))
	}
	return now.Truncate(window)
}

type budgetTracker struct {
	key          string
	scope        string
	limit        float64
	window       time.Duration
	spent        float64
	reserved     float64
	reservations map[string]*budgetReservation
	finalized    map[string]bool
	windowStart  time.Time
	alertAt      float64
	guard        *BudgetGuardImpl
	mu           sync.Mutex
}

func (t *budgetTracker) now() time.Time {
	if t.guard != nil && t.guard.clock != nil {
		return t.guard.clock()
	}
	return time.Now()
}

// cleanExpired removes active reservations that have exceeded their safety TTL.
func (t *budgetTracker) cleanExpired(now time.Time) {
	for id, res := range t.reservations {
		if now.After(res.expiresAt) {
			t.reserved -= res.amount
			if t.reserved < 0 {
				t.reserved = 0
			}
			delete(t.reservations, id)
		}
	}
}

func (t *budgetTracker) checkRollover(now time.Time) {
	if t.window <= 0 {
		return
	}
	shouldReset := false
	if t.window == 24*time.Hour || t.window == 86400*time.Second || t.window == 30*24*time.Hour || t.window == 2592000*time.Second || t.window == 7*24*time.Hour || t.window == 604800*time.Second {
		winStart := computeWindowStart(now, t.window)
		if !t.windowStart.IsZero() && winStart.After(t.windowStart) {
			shouldReset = true
		}
	}
	if !t.windowStart.IsZero() && now.Sub(t.windowStart) > t.window {
		shouldReset = true
	}
	if shouldReset {
		t.spent = 0
		t.reserved = 0
		t.reservations = make(map[string]*budgetReservation)
		t.finalized = make(map[string]bool)
		t.windowStart = now
	}
}

// check verifies if the estimated cost fits within budget.
func (t *budgetTracker) check(estimatedCost float64) BudgetResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.checkRollover(now)
	t.cleanExpired(now)

	projected := t.spent + t.reserved + estimatedCost
	allowed := t.limit <= 0 || projected <= t.limit

	return BudgetResult{
		Allowed:   allowed,
		Current:   t.spent + t.reserved,
		Limit:     t.limit,
		Projected: projected,
		AlertAt:   t.alertAt,
		Scope:     t.scope,
		Key:       t.key,
	}
}

// reserve atomically checks and reserves estimated cost/tokens under mutex.
func (t *budgetTracker) reserve(reqID string, estimatedCost float64) (BudgetResult, BudgetReleaseFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.checkRollover(now)
	t.cleanExpired(now)

	projected := t.spent + t.reserved + estimatedCost
	allowed := t.limit <= 0 || projected <= t.limit

	res := BudgetResult{
		Allowed:   allowed,
		Current:   t.spent + t.reserved,
		Limit:     t.limit,
		Projected: projected,
		AlertAt:   t.alertAt,
		Scope:     t.scope,
		Key:       t.key,
	}

	if !allowed {
		return res, func(_ float64, _ error) {}
	}

	if reqID == "" {
		reqID = uuid.NewString()
	}

	ttl := 2 * time.Minute
	if t.guard != nil && t.guard.resTTL > 0 {
		ttl = t.guard.resTTL
	}

	if t.reservations == nil {
		t.reservations = make(map[string]*budgetReservation)
	}
	t.reservations[reqID] = &budgetReservation{
		reqID:     reqID,
		amount:    estimatedCost,
		expiresAt: now.Add(ttl),
	}
	t.reserved += estimatedCost

	var once sync.Once
	release := func(actualCost float64, err error) {
		once.Do(func() {
			t.mu.Lock()
			defer t.mu.Unlock()

			if t.finalized != nil && t.finalized[reqID] {
				return
			}
			if t.finalized == nil {
				t.finalized = make(map[string]bool)
			}
			t.finalized[reqID] = true

			if r, ok := t.reservations[reqID]; ok {
				t.reserved -= r.amount
				if t.reserved < 0 {
					t.reserved = 0
				}
				delete(t.reservations, reqID)
			}

			// Add actual cost whenever greater than zero (both full completion and partial/interrupted streams)
			if actualCost > 0 {
				t.spent += actualCost
			}
		})
	}

	return res, release
}

// BudgetGuardImpl implements budget tracking and enforcement.
type BudgetGuardImpl struct {
	mu             sync.RWMutex
	budgets        map[string]*budgetTracker
	policyResolver PolicyResolver
	redis          *redis.Client
	spendLoader    SpendLoader
	logger         *slog.Logger
	clock          func() time.Time
	resTTL         time.Duration
	mode           BudgetGuardMode
}

// NewBudgetGuard creates a new budget guard with in-memory tracking.
func NewBudgetGuard() *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		clock:   time.Now,
		resTTL:  2 * time.Minute,
		mode:    BudgetModeDegradedLocal,
	}
}

// NewBudgetGuardWithResolver creates a new budget guard that loads budgets from policy snapshots.
func NewBudgetGuardWithResolver(policyResolver PolicyResolver) *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets:        make(map[string]*budgetTracker),
		policyResolver: policyResolver,
		clock:          time.Now,
		resTTL:         2 * time.Minute,
		mode:           BudgetModeDegradedLocal,
	}
}

// NewBudgetGuardWithRedis creates a budget guard backed by Redis with in-memory fallback.
func NewBudgetGuardWithRedis(client *redis.Client, logger *slog.Logger) *BudgetGuardImpl {
	if logger == nil {
		logger = slog.Default()
	}

	mode := BudgetModeDegradedLocal
	if os.Getenv("AIGW_BUDGET_FAIL_CLOSED") == "true" {
		mode = BudgetModeFailClosed
	}

	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		redis:   client,
		logger:  logger,
		clock:   time.Now,
		resTTL:  2 * time.Minute,
		mode:    mode,
	}
}

// WithMode configures the outage handling policy (fail-closed or degraded local).
func (b *BudgetGuardImpl) WithMode(mode BudgetGuardMode) *BudgetGuardImpl {
	b.mode = mode
	return b
}

// Mode returns the active failure policy mode.
func (b *BudgetGuardImpl) Mode() BudgetGuardMode {
	return b.mode
}

// WithTTL overrides reservation safety timeout (primarily for tests).
func (b *BudgetGuardImpl) WithTTL(ttl time.Duration) *BudgetGuardImpl {
	b.resTTL = ttl
	return b
}

// WithSpendLoader registers a loader function to populate initial spend on gateway restart.
func (b *BudgetGuardImpl) WithSpendLoader(loader SpendLoader) *BudgetGuardImpl {
	b.spendLoader = loader
	return b
}

// Check verifies if the estimated cost fits within the budget.
func (b *BudgetGuardImpl) Check(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) BudgetResult {
	if b.policyResolver != nil && limit == 0 && window == 0 {
		if budgetConfig, ok := b.getBudgetConfigFromSnapshot(ctx, key); ok {
			limit = budgetConfig.Limit
			window = budgetConfig.Window
		}
	}

	if b.redis != nil {
		result, err := b.checkRedis(ctx, key, limit, window, estimatedCost)
		if err == nil {
			return result
		}
		return b.handleRedisCheckOutage(ctx, key, limit, window, estimatedCost, err)
	}

	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	return tracker.check(estimatedCost)
}

// Reserve atomically checks and reserves estimated cost/tokens against the budget if within limit,
// returning a release callback to apply actual cost when the request finishes.
func (b *BudgetGuardImpl) Reserve(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, reqID string) (BudgetResult, BudgetReleaseFunc) {
	if b.policyResolver != nil && limit == 0 && window == 0 {
		if budgetConfig, ok := b.getBudgetConfigFromSnapshot(ctx, key); ok {
			limit = budgetConfig.Limit
			window = budgetConfig.Window
		}
	}

	if reqID == "" {
		reqID = uuid.NewString()
	}

	// 1. If Redis is available, attempt atomic Redis reservation
	if b.redis != nil {
		result, release, err := b.reserveRedis(ctx, key, limit, window, estimatedCost, reqID)
		if err == nil {
			return result, release
		}
		return b.handleRedisOutage(ctx, key, limit, window, estimatedCost, reqID, err)
	}

	// 2. In-memory tracker
	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	return tracker.reserve(reqID, estimatedCost)
}

// Lua script for atomic check in Redis with window rollover and expired reservation recovery
var redisCheckScript = redis.NewScript(`
local spentKey = KEYS[1]
local reservedKey = KEYS[2]
local winStartKey = KEYS[3]
local resZSetKey = KEYS[4]
local resHashKey = KEYS[5]

local limit = tonumber(ARGV[1])
local est = tonumber(ARGV[2])
local nowMs = tonumber(ARGV[3])
local windowMs = tonumber(ARGV[4])
local windowStartMs = tonumber(ARGV[5])

-- 1. Window rollover check
if windowMs > 0 and windowStartMs > 0 then
    local storedWinStart = tonumber(redis.call('GET', winStartKey) or '0')
    if storedWinStart == 0 then
        redis.call('SET', winStartKey, tostring(windowStartMs))
    elseif windowStartMs > storedWinStart or (nowMs - storedWinStart) > windowMs then
        redis.call('SET', spentKey, '0')
        redis.call('SET', winStartKey, tostring(windowStartMs))
    end
end

-- 2. Sweep expired reservations (prevent aggregate inflation)
local expired = redis.call('ZRANGEBYSCORE', resZSetKey, 0, nowMs)
if expired and #expired > 0 then
    local totalExpiredAmt = 0
    for _, expID in ipairs(expired) do
        local amt = tonumber(redis.call('HGET', resHashKey, expID) or '0')
        totalExpiredAmt = totalExpiredAmt + amt
        redis.call('HDEL', resHashKey, expID)
    end
    redis.call('ZREMRANGEBYSCORE', resZSetKey, 0, nowMs)
    if totalExpiredAmt > 0 then
        local curRes = tonumber(redis.call('GET', reservedKey) or '0')
        local newRes = curRes - totalExpiredAmt
        if newRes < 0 then newRes = 0 end
        redis.call('SET', reservedKey, tostring(newRes))
    end
end

-- 3. Check allowance
local spent = tonumber(redis.call('GET', spentKey) or '0')
local reserved = tonumber(redis.call('GET', reservedKey) or '0')
local projected = spent + reserved + est
local allowed = 0
if limit <= 0 or projected <= limit then
    allowed = 1
end

return {allowed, tostring(spent + reserved), tostring(projected)}
`)

// Lua script for atomic check & reserve in Redis
var redisReserveScript = redis.NewScript(`
local spentKey = KEYS[1]
local reservedKey = KEYS[2]
local winStartKey = KEYS[3]
local resZSetKey = KEYS[4]
local resHashKey = KEYS[5]

local limit = tonumber(ARGV[1])
local est = tonumber(ARGV[2])
local ttlMs = tonumber(ARGV[3])
local nowMs = tonumber(ARGV[4])
local windowMs = tonumber(ARGV[5])
local windowStartMs = tonumber(ARGV[6])
local reqID = ARGV[7]

-- 1. Window rollover check
if windowMs > 0 and windowStartMs > 0 then
    local storedWinStart = tonumber(redis.call('GET', winStartKey) or '0')
    if storedWinStart == 0 then
        redis.call('SET', winStartKey, tostring(windowStartMs))
    elseif windowStartMs > storedWinStart or (nowMs - storedWinStart) > windowMs then
        redis.call('SET', spentKey, '0')
        redis.call('SET', winStartKey, tostring(windowStartMs))
    end
end

-- 2. Sweep expired reservations to prevent aggregate inflation
local expired = redis.call('ZRANGEBYSCORE', resZSetKey, 0, nowMs)
if expired and #expired > 0 then
    local totalExpiredAmt = 0
    for _, expID in ipairs(expired) do
        local amt = tonumber(redis.call('HGET', resHashKey, expID) or '0')
        totalExpiredAmt = totalExpiredAmt + amt
        redis.call('HDEL', resHashKey, expID)
    end
    redis.call('ZREMRANGEBYSCORE', resZSetKey, 0, nowMs)
    if totalExpiredAmt > 0 then
        local curRes = tonumber(redis.call('GET', reservedKey) or '0')
        local newRes = curRes - totalExpiredAmt
        if newRes < 0 then newRes = 0 end
        redis.call('SET', reservedKey, tostring(newRes))
    end
end

-- 3. Idempotency: if reqID is already active, return success without double reserving
local existingAmt = redis.call('HGET', resHashKey, reqID)
local spent = tonumber(redis.call('GET', spentKey) or '0')
local reserved = tonumber(redis.call('GET', reservedKey) or '0')

if existingAmt then
    return {1, tostring(spent + reserved), tostring(spent + reserved)}
end

-- 4. Check projected cost against limit
local projected = spent + reserved + est
if limit <= 0 or projected <= limit then
    redis.call('INCRBYFLOAT', reservedKey, est)
    redis.call('ZADD', resZSetKey, nowMs + ttlMs, reqID)
    redis.call('HSET', resHashKey, reqID, tostring(est))
    return {1, tostring(spent + reserved), tostring(projected)}
else
    return {0, tostring(spent + reserved), tostring(projected)}
end
`)

// Lua script for releasing reservation and committing actual spend in Redis
var redisReleaseScript = redis.NewScript(`
local spentKey = KEYS[1]
local reservedKey = KEYS[2]
local winStartKey = KEYS[3]
local resZSetKey = KEYS[4]
local resHashKey = KEYS[5]

local fallbackEst = tonumber(ARGV[1])
local actual = tonumber(ARGV[2])
local success = tonumber(ARGV[3])
local nowMs = tonumber(ARGV[4])
local reqID = ARGV[5]

-- 1. Sweep expired reservations (excluding current reqID)
local expired = redis.call('ZRANGEBYSCORE', resZSetKey, 0, nowMs)
if expired and #expired > 0 then
    local totalExpiredAmt = 0
    for _, expID in ipairs(expired) do
        if expID ~= reqID then
            local amt = tonumber(redis.call('HGET', resHashKey, expID) or '0')
            totalExpiredAmt = totalExpiredAmt + amt
            redis.call('HDEL', resHashKey, expID)
        end
    end
    redis.call('ZREMRANGEBYSCORE', resZSetKey, 0, nowMs)
    if totalExpiredAmt > 0 then
        local curRes = tonumber(redis.call('GET', reservedKey) or '0')
        local newRes = curRes - totalExpiredAmt
        if newRes < 0 then newRes = 0 end
        redis.call('SET', reservedKey, tostring(newRes))
    end
end

-- 2. Release reservation for target reqID
local est = tonumber(redis.call('HGET', resHashKey, reqID) or fallbackEst)
local wasReserved = redis.call('HEXISTS', resHashKey, reqID)

redis.call('HDEL', resHashKey, reqID)
redis.call('ZREM', resZSetKey, reqID)

if wasReserved == 1 and est > 0 then
    local curRes = tonumber(redis.call('GET', reservedKey) or '0')
    local newRes = curRes - est
    if newRes < 0 then newRes = 0 end
    redis.call('SET', reservedKey, tostring(newRes))
end

-- 3. If request succeeded or client disconnected with partial usage, commit actual spend
if success == 1 and actual > 0 then
    redis.call('INCRBYFLOAT', spentKey, actual)
end

return 1
`)

// Lua script for recording usage directly with window rollover support
var redisRecordUsageScript = redis.NewScript(`
local spentKey = KEYS[1]
local winStartKey = KEYS[2]

local cost = tonumber(ARGV[1])
local windowMs = tonumber(ARGV[2])
local windowStartMs = tonumber(ARGV[3])

if windowMs > 0 and windowStartMs > 0 then
    local storedWinStart = tonumber(redis.call('GET', winStartKey) or '0')
    if storedWinStart == 0 then
        redis.call('SET', winStartKey, tostring(windowStartMs))
    elseif windowStartMs > storedWinStart then
        redis.call('SET', spentKey, '0')
        redis.call('SET', winStartKey, tostring(windowStartMs))
    end
end

if cost > 0 then
    redis.call('INCRBYFLOAT', spentKey, cost)
end

return 1
`)

func (b *BudgetGuardImpl) reserveRedis(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, reqID string) (BudgetResult, BudgetReleaseFunc, error) {
	spentKey := "aigw:budget:" + key + ":spent"
	reservedKey := "aigw:budget:" + key + ":reserved"
	winStartKey := "aigw:budget:" + key + ":window_start"
	resZSetKey := "aigw:budget:" + key + ":res_zset"
	resHashKey := "aigw:budget:" + key + ":res_hash"

	ttl := b.resTTL
	if ttl <= 0 {
		ttl = 2 * time.Minute
	}
	ttlMs := ttl.Milliseconds()
	now := b.clock().UTC()
	nowMs := now.UnixMilli()
	windowMs := window.Milliseconds()
	windowStartMs := computeWindowStart(now, window).UnixMilli()

	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	keys := []string{spentKey, reservedKey, winStartKey, resZSetKey, resHashKey}
	res, err := redisReserveScript.Run(ctx, b.redis, keys, limit, estimatedCost, ttlMs, nowMs, windowMs, windowStartMs, reqID).Result()
	if err != nil {
		return BudgetResult{}, nil, err
	}

	vals, ok := res.([]any)
	if !ok || len(vals) < 3 {
		return BudgetResult{}, nil, errors.New("invalid redis script response")
	}

	allowed := vals[0].(int64) == 1
	current, _ := strconv.ParseFloat(vals[1].(string), 64)
	projected, _ := strconv.ParseFloat(vals[2].(string), 64)

	bResult := BudgetResult{
		Allowed:   allowed,
		Current:   current,
		Limit:     limit,
		Projected: projected,
		AlertAt:   0.8 * limit,
		Scope:     scope,
		Key:       key,
		Degraded:  false,
	}

	if !allowed {
		return bResult, func(_ float64, _ error) {}, nil
	}

	// Also update local tracker to keep in-memory fallback in sync
	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	_, localRelease := tracker.reserve(reqID, estimatedCost)

	var once sync.Once
	release := func(actualCost float64, err error) {
		once.Do(func() {
			successFlag := 1
			if actualCost <= 0 {
				successFlag = 0
				actualCost = 0
			}

			relNowMs := b.clock().UTC().UnixMilli()
			// Release in Redis
			rErr := redisReleaseScript.Run(context.Background(), b.redis, keys, estimatedCost, actualCost, successFlag, relNowMs, reqID).Err()
			if rErr != nil && b.logger != nil {
				b.logger.Error("redis budget release script failed during finalization", "error", rErr, "reqID", reqID, "key", key)
			}

			// Also release in local tracker
			localRelease(actualCost, err)
		})
	}

	return bResult, release, nil
}

func (b *BudgetGuardImpl) checkRedis(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) (BudgetResult, error) {
	spentKey := "aigw:budget:" + key + ":spent"
	reservedKey := "aigw:budget:" + key + ":reserved"
	winStartKey := "aigw:budget:" + key + ":window_start"
	resZSetKey := "aigw:budget:" + key + ":res_zset"
	resHashKey := "aigw:budget:" + key + ":res_hash"

	now := b.clock().UTC()
	nowMs := now.UnixMilli()
	windowMs := window.Milliseconds()
	windowStartMs := computeWindowStart(now, window).UnixMilli()

	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	keys := []string{spentKey, reservedKey, winStartKey, resZSetKey, resHashKey}
	res, err := redisCheckScript.Run(ctx, b.redis, keys, limit, estimatedCost, nowMs, windowMs, windowStartMs).Result()
	if err != nil {
		return BudgetResult{}, err
	}

	vals, ok := res.([]any)
	if !ok || len(vals) < 3 {
		return BudgetResult{}, errors.New("invalid redis check script response")
	}

	allowed := vals[0].(int64) == 1
	current, _ := strconv.ParseFloat(vals[1].(string), 64)
	projected, _ := strconv.ParseFloat(vals[2].(string), 64)

	return BudgetResult{
		Allowed:   allowed,
		Current:   current,
		Limit:     limit,
		Projected: projected,
		AlertAt:   0.8 * limit,
		Scope:     scope,
		Key:       key,
		Degraded:  false,
	}, nil
}

func (b *BudgetGuardImpl) handleRedisOutage(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, reqID string, err error) (BudgetResult, BudgetReleaseFunc) {
	if b.mode == BudgetModeFailClosed {
		if b.logger != nil {
			b.logger.Error("redis budget guard unavailable: FAILING CLOSED (request rejected)", "key", key, "error", err)
		}
		scope := ""
		if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
			scope = parts[0]
		}
		return BudgetResult{
			Allowed:   false,
			Current:   0,
			Limit:     limit,
			Projected: estimatedCost,
			AlertAt:   0.8 * limit,
			Scope:     scope,
			Key:       key,
			Degraded:  true,
		}, func(_ float64, _ error) {}
	}

	if b.logger != nil {
		b.logger.Warn("redis budget guard unavailable: operating in DEGRADED node-local mode; limits are not cluster-wide", "key", key, "error", err)
	}
	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	res, release := tracker.reserve(reqID, estimatedCost)
	res.Degraded = true
	return res, release
}

func (b *BudgetGuardImpl) handleRedisCheckOutage(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, err error) BudgetResult {
	if b.mode == BudgetModeFailClosed {
		if b.logger != nil {
			b.logger.Error("redis budget check unavailable: FAILING CLOSED (check rejected)", "key", key, "error", err)
		}
		scope := ""
		if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
			scope = parts[0]
		}
		return BudgetResult{
			Allowed:   false,
			Current:   0,
			Limit:     limit,
			Projected: estimatedCost,
			AlertAt:   0.8 * limit,
			Scope:     scope,
			Key:       key,
			Degraded:  true,
		}
	}

	if b.logger != nil {
		b.logger.Warn("redis budget check unavailable: operating in DEGRADED node-local mode", "key", key, "error", err)
	}
	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	res := tracker.check(estimatedCost)
	res.Degraded = true
	return res
}

// RecordUsage records actual usage directly against the budget.
func (b *BudgetGuardImpl) RecordUsage(ctx context.Context, key string, cost float64) error {
	if cost <= 0 {
		return nil
	}

	now := b.clock().UTC()
	var winMs int64
	var winStartMs int64
	b.mu.RLock()
	if tracker, ok := b.budgets[key]; ok {
		if tracker.window > 0 {
			winMs = tracker.window.Milliseconds()
			winStartMs = computeWindowStart(now, tracker.window).UnixMilli()
		}
	}
	b.mu.RUnlock()

	if b.redis != nil {
		spentKey := "aigw:budget:" + key + ":spent"
		winStartKey := "aigw:budget:" + key + ":window_start"
		err := redisRecordUsageScript.Run(ctx, b.redis, []string{spentKey, winStartKey}, cost, winMs, winStartMs).Err()
		if err != nil {
			if b.mode == BudgetModeFailClosed {
				if b.logger != nil {
					b.logger.Error("redis record usage failed: FAILING CLOSED", "error", err, "key", key)
				}
				return fmt.Errorf("redis record usage failed: %w", err)
			}
			if b.logger != nil {
				b.logger.Warn("redis record usage failed, continuing with local tracker in degraded mode", "error", err, "key", key)
			}
		}
	}

	tracker := b.getOrCreateTracker(ctx, key, 0, 0, 0)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	tracker.checkRollover(now)
	tracker.spent += cost
	return nil
}

// GetBudget returns current budget status for a key.
func (b *BudgetGuardImpl) GetBudget(key string) (spent, limit float64, windowStart time.Time, ok bool) {
	b.mu.RLock()
	tracker, trackerOk := b.budgets[key]
	b.mu.RUnlock()

	limitVal := 0.0
	winStart := time.Time{}
	if trackerOk {
		tracker.mu.Lock()
		limitVal = tracker.limit
		winStart = tracker.windowStart
		tracker.mu.Unlock()
	}

	if b.redis != nil {
		spentKey := "aigw:budget:" + key + ":spent"
		winKey := "aigw:budget:" + key + ":window_start"
		pipe := b.redis.Pipeline()
		spentCmd := pipe.Get(context.Background(), spentKey)
		winCmd := pipe.Get(context.Background(), winKey)
		_, err := pipe.Exec(context.Background())
		if err == nil || errors.Is(err, redis.Nil) {
			var rSpent float64
			if s, sErr := spentCmd.Result(); sErr == nil {
				rSpent, _ = strconv.ParseFloat(s, 64)
			}
			if w, wErr := winCmd.Result(); wErr == nil {
				if wMs, parseErr := strconv.ParseInt(w, 10, 64); parseErr == nil && wMs > 0 {
					winStart = time.UnixMilli(wMs).UTC()
				}
			}
			return rSpent, limitVal, winStart, true
		}
	}

	if !trackerOk {
		return 0, 0, time.Time{}, false
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.spent, tracker.limit, tracker.windowStart, true
}

// GetBudgetState returns current spent, active reserved, and limit for a key.
func (b *BudgetGuardImpl) GetBudgetState(key string) (spent, reserved, limit float64, ok bool) {
	b.mu.RLock()
	tracker, trackerOk := b.budgets[key]
	b.mu.RUnlock()

	limitVal := 0.0
	winVal := time.Duration(0)
	if trackerOk {
		tracker.mu.Lock()
		limitVal = tracker.limit
		winVal = tracker.window
		tracker.mu.Unlock()
	}

	if b.redis != nil {
		spentKey := "aigw:budget:" + key + ":spent"
		resKey := "aigw:budget:" + key + ":reserved"
		winKey := "aigw:budget:" + key + ":window_start"
		resZSetKey := "aigw:budget:" + key + ":res_zset"
		resHashKey := "aigw:budget:" + key + ":res_hash"
		nowMs := b.clock().UTC().UnixMilli()

		// Run check script with est=0 to sweep expired reservations before returning state
		_, err := redisCheckScript.Run(context.Background(), b.redis, []string{spentKey, resKey, winKey, resZSetKey, resHashKey}, limitVal, 0, nowMs, winVal.Milliseconds(), computeWindowStart(b.clock().UTC(), winVal).UnixMilli()).Result()
		if err == nil {
			pipe := b.redis.Pipeline()
			sCmd := pipe.Get(context.Background(), spentKey)
			rCmd := pipe.Get(context.Background(), resKey)
			_, pErr := pipe.Exec(context.Background())
			if pErr == nil || errors.Is(pErr, redis.Nil) {
				sVal, _ := strconv.ParseFloat(sCmd.Val(), 64)
				rVal, _ := strconv.ParseFloat(rCmd.Val(), 64)
				return sVal, rVal, limitVal, true
			}
		}
	}

	if !trackerOk {
		return 0, 0, 0, false
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.spent, tracker.reserved, tracker.limit, true
}

// ResetBudget resets a budget tracker.
func (b *BudgetGuardImpl) ResetBudget(key string) {
	b.mu.Lock()
	delete(b.budgets, key)
	b.mu.Unlock()

	if b.redis != nil {
		ctx := context.Background()
		_ = b.redis.Del(ctx,
			"aigw:budget:"+key+":spent",
			"aigw:budget:"+key+":reserved",
			"aigw:budget:"+key+":window_start",
			"aigw:budget:"+key+":res_zset",
			"aigw:budget:"+key+":res_hash",
		).Err()
	}
}

// Close stops the budget guard and flushes in-memory trackers.
func (b *BudgetGuardImpl) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.budgets = make(map[string]*budgetTracker)
}

// SyncFromSnapshot loads budget configurations from a policy snapshot.
func (b *BudgetGuardImpl) SyncFromSnapshot(snapshot PolicySnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().UTC()
	if b.clock != nil {
		now = b.clock().UTC()
	}

	for key, config := range snapshot.Budgets {
		winStart := computeWindowStart(now, config.Window)
		if tracker, ok := b.budgets[key]; ok {
			tracker.mu.Lock()
			tracker.limit = config.Limit
			tracker.window = config.Window
			tracker.alertAt = config.AlertAt
			tracker.scope = config.Scope
			tracker.windowStart = winStart
			tracker.mu.Unlock()
		} else {
			tracker := &budgetTracker{
				key:          key,
				scope:        config.Scope,
				limit:        config.Limit,
				window:       config.Window,
				spent:        0,
				reserved:     0,
				reservations: make(map[string]*budgetReservation),
				finalized:    make(map[string]bool),
				windowStart:  winStart,
				alertAt:      config.AlertAt,
				guard:        b,
			}
			if b.spendLoader != nil {
				if s, err := b.spendLoader(context.Background(), key, tracker.windowStart); err == nil && s > 0 {
					tracker.spent = s
				}
			}
			b.budgets[key] = tracker
		}
	}
}

func (b *BudgetGuardImpl) getBudgetConfigFromSnapshot(ctx context.Context, key string) (BudgetConfig, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if tracker, ok := b.budgets[key]; ok {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()
		return BudgetConfig{
			Key:     tracker.key,
			Limit:   tracker.limit,
			Window:  tracker.window,
			Scope:   tracker.scope,
			AlertAt: tracker.alertAt,
		}, true
	}
	return BudgetConfig{}, false
}

func (b *BudgetGuardImpl) getOrCreateTracker(ctx context.Context, key string, limit float64, window time.Duration, alertAt float64) *budgetTracker {
	b.mu.RLock()
	tracker, ok := b.budgets[key]
	b.mu.RUnlock()

	if ok {
		if limit > 0 && tracker.limit != limit {
			tracker.mu.Lock()
			tracker.limit = limit
			tracker.window = window
			tracker.alertAt = alertAt
			tracker.mu.Unlock()
		}
		return tracker
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if tracker, ok := b.budgets[key]; ok {
		return tracker
	}

	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	now := time.Now().UTC()
	if b.clock != nil {
		now = b.clock().UTC()
	}

	winStart := computeWindowStart(now, window)

	tracker = &budgetTracker{
		key:          key,
		scope:        scope,
		limit:        limit,
		window:       window,
		spent:        0,
		reserved:     0,
		reservations: make(map[string]*budgetReservation),
		finalized:    make(map[string]bool),
		windowStart:  winStart,
		alertAt:      alertAt,
		guard:        b,
	}

	if b.spendLoader != nil {
		if s, err := b.spendLoader(ctx, key, tracker.windowStart); err == nil && s > 0 {
			tracker.spent = s
		}
	}

	b.budgets[key] = tracker
	return tracker
}
