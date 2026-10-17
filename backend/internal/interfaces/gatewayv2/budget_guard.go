// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SpendLoader loads historical spend for a given budget key within the active window [windowStart, now].
// Used during cold start or gateway restart to reconcile in-memory tracker with persisted ledger.
type SpendLoader func(ctx context.Context, key string, windowStart time.Time) (float64, error)

type budgetReservation struct {
	reqID     string
	amount    float64
	expiresAt time.Time
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

// check verifies if the estimated cost fits within budget.
func (t *budgetTracker) check(estimatedCost float64) BudgetResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	if t.window > 0 && now.Sub(t.windowStart) > t.window {
		t.spent = 0
		t.reserved = 0
		t.reservations = make(map[string]*budgetReservation)
		t.finalized = make(map[string]bool)
		t.windowStart = now
	}

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
	if t.window > 0 && now.Sub(t.windowStart) > t.window {
		t.spent = 0
		t.reserved = 0
		t.reservations = make(map[string]*budgetReservation)
		t.finalized = make(map[string]bool)
		t.windowStart = now
	}

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
}

// NewBudgetGuard creates a new budget guard with in-memory tracking.
func NewBudgetGuard() *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		clock:   time.Now,
		resTTL:  2 * time.Minute,
	}
}

// NewBudgetGuardWithResolver creates a new budget guard that loads budgets from policy snapshots.
func NewBudgetGuardWithResolver(policyResolver PolicyResolver) *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets:        make(map[string]*budgetTracker),
		policyResolver: policyResolver,
		clock:          time.Now,
		resTTL:         2 * time.Minute,
	}
}

// NewBudgetGuardWithRedis creates a budget guard backed by Redis with in-memory fallback.
func NewBudgetGuardWithRedis(client *redis.Client, logger *slog.Logger) *BudgetGuardImpl {
	if logger == nil {
		logger = slog.Default()
	}
	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		redis:   client,
		logger:  logger,
		clock:   time.Now,
		resTTL:  2 * time.Minute,
	}
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
		if b.logger != nil {
			b.logger.Warn("redis budget check failed, using in-memory tracker", "error", err, "key", key)
		}
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
		if b.logger != nil {
			b.logger.Warn("redis budget reservation failed, falling back to in-memory tracker", "error", err, "key", key)
		}
	}

	// 2. In-memory tracker
	tracker := b.getOrCreateTracker(ctx, key, limit, window, 0.8*limit)
	return tracker.reserve(reqID, estimatedCost)
}

// Lua script for atomic check & reserve in Redis
var redisReserveScript = redis.NewScript(`
local spent = tonumber(redis.call('GET', KEYS[1]) or '0')
local reserved = tonumber(redis.call('GET', KEYS[2]) or '0')
local limit = tonumber(ARGV[1])
local est = tonumber(ARGV[2])
local ttl = tonumber(ARGV[3])

local projected = spent + reserved + est
if limit <= 0 or projected <= limit then
    redis.call('INCRBYFLOAT', KEYS[2], est)
    redis.call('SETEX', KEYS[3], ttl, tostring(est))
    return {1, tostring(spent + reserved), tostring(projected)}
else
    return {0, tostring(spent + reserved), tostring(projected)}
end
`)

// Lua script for releasing reservation and committing actual spend in Redis
var redisReleaseScript = redis.NewScript(`
local est = tonumber(redis.call('GET', KEYS[3]) or ARGV[1])
local actual = tonumber(ARGV[2])
local success = tonumber(ARGV[3])

-- Clean up reservation key
redis.call('DEL', KEYS[3])

if est > 0 then
    local reserved = tonumber(redis.call('GET', KEYS[2]) or '0')
    local new_reserved = reserved - est
    if new_reserved < 0 then new_reserved = 0 end
    redis.call('SET', KEYS[2], tostring(new_reserved))
end

if success == 1 and actual > 0 then
    redis.call('INCRBYFLOAT', KEYS[1], actual)
end
return 1
`)

func (b *BudgetGuardImpl) reserveRedis(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64, reqID string) (BudgetResult, BudgetReleaseFunc, error) {
	spentKey := "aigw:budget:" + key + ":spent"
	reservedKey := "aigw:budget:" + key + ":reserved"
	resItemKey := "aigw:budget:" + key + ":res:" + reqID
	ttlSec := int(b.resTTL.Seconds())
	if ttlSec <= 0 {
		ttlSec = 120
	}

	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	res, err := redisReserveScript.Run(ctx, b.redis, []string{spentKey, reservedKey, resItemKey}, limit, estimatedCost, ttlSec).Result()
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

			// Release in Redis
			_ = redisReleaseScript.Run(context.Background(), b.redis, []string{spentKey, reservedKey, resItemKey}, estimatedCost, actualCost, successFlag).Err()

			// Also release in local tracker
			localRelease(actualCost, err)
		})
	}

	return bResult, release, nil
}

func (b *BudgetGuardImpl) checkRedis(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) (BudgetResult, error) {
	spentKey := "aigw:budget:" + key + ":spent"
	reservedKey := "aigw:budget:" + key + ":reserved"

	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	pipe := b.redis.Pipeline()
	spentCmd := pipe.Get(ctx, spentKey)
	resCmd := pipe.Get(ctx, reservedKey)
	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return BudgetResult{}, err
	}

	var spent, reserved float64
	if sVal, sErr := spentCmd.Result(); sErr == nil {
		spent, _ = strconv.ParseFloat(sVal, 64)
	}
	if rVal, rErr := resCmd.Result(); rErr == nil {
		reserved, _ = strconv.ParseFloat(rVal, 64)
	}

	projected := spent + reserved + estimatedCost
	allowed := limit <= 0 || projected <= limit

	return BudgetResult{
		Allowed:   allowed,
		Current:   spent + reserved,
		Limit:     limit,
		Projected: projected,
		AlertAt:   0.8 * limit,
		Scope:     scope,
		Key:       key,
	}, nil
}

// RecordUsage records actual usage directly against the budget.
func (b *BudgetGuardImpl) RecordUsage(ctx context.Context, key string, cost float64) error {
	if cost <= 0 {
		return nil
	}

	if b.redis != nil {
		spentKey := "aigw:budget:" + key + ":spent"
		_ = b.redis.IncrByFloat(ctx, spentKey, cost).Err()
	}

	tracker := b.getOrCreateTracker(ctx, key, 0, 0, 0)
	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := tracker.now()
	if tracker.window > 0 && now.Sub(tracker.windowStart) > tracker.window {
		tracker.spent = 0
		tracker.windowStart = now
	}
	tracker.spent += cost
	return nil
}

// GetBudget returns current budget status for a key.
func (b *BudgetGuardImpl) GetBudget(key string) (spent, limit float64, windowStart time.Time, ok bool) {
	b.mu.RLock()
	tracker, ok := b.budgets[key]
	b.mu.RUnlock()

	if !ok {
		return 0, 0, time.Time{}, false
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.spent, tracker.limit, tracker.windowStart, true
}

// GetBudgetState returns current spent, active reserved, and limit for a key.
func (b *BudgetGuardImpl) GetBudgetState(key string) (spent, reserved, limit float64, ok bool) {
	b.mu.RLock()
	tracker, ok := b.budgets[key]
	b.mu.RUnlock()

	if !ok {
		return 0, 0, 0, false
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	return tracker.spent, tracker.reserved, tracker.limit, true
}

// ResetBudget resets a budget tracker.
func (b *BudgetGuardImpl) ResetBudget(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.budgets, key)
	if b.redis != nil {
		ctx := context.Background()
		_ = b.redis.Del(ctx, "aigw:budget:"+key+":spent", "aigw:budget:"+key+":reserved").Err()
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

	now := time.Now()
	if b.clock != nil {
		now = b.clock()
	}

	for key, config := range snapshot.Budgets {
		if tracker, ok := b.budgets[key]; ok {
			tracker.mu.Lock()
			tracker.limit = config.Limit
			tracker.window = config.Window
			tracker.alertAt = config.AlertAt
			tracker.scope = config.Scope
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
				windowStart:  now,
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

	now := time.Now()
	if b.clock != nil {
		now = b.clock()
	}

	tracker = &budgetTracker{
		key:          key,
		scope:        scope,
		limit:        limit,
		window:       window,
		spent:        0,
		reserved:     0,
		reservations: make(map[string]*budgetReservation),
		finalized:    make(map[string]bool),
		windowStart:  now,
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
