// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"strings"
	"sync"
	"time"
)

// BudgetGuardImpl implements budget tracking and enforcement.
type BudgetGuardImpl struct {
	mu          sync.RWMutex
	budgets     map[string]*budgetTracker
	policyResolver PolicyResolver
	clock       func() time.Time
}

type budgetTracker struct {
	key         string
	scope       string
	limit       float64
	window      time.Duration
	spent       float64
	windowStart time.Time
	alertAt     float64
	mu          sync.Mutex
}

// NewBudgetGuard creates a new budget guard.
func NewBudgetGuard() *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		clock:   time.Now,
	}
}

// NewBudgetGuardWithResolver creates a new budget guard that loads budgets from policy snapshots.
func NewBudgetGuardWithResolver(policyResolver PolicyResolver) *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets:        make(map[string]*budgetTracker),
		policyResolver: policyResolver,
		clock:          time.Now,
	}
}

// Check verifies if the estimated cost fits within the budget.
func (b *BudgetGuardImpl) Check(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) BudgetResult {
	// If we have a policy resolver, try to load budget config from snapshot
	if b.policyResolver != nil && limit == 0 && window == 0 {
		// Try to get budget config from policy snapshot
		// The key should be the budget key from the policy rule
		if budgetConfig, ok := b.getBudgetConfigFromSnapshot(ctx, key); ok {
			limit = budgetConfig.Limit
			window = budgetConfig.Window
		}
	}
	
	tracker := b.getOrCreateTracker(key, limit, window, 0.8*limit) // default alert at 80%
	return tracker.check(estimatedCost)
}

// Reserve atomically reserves estimated cost/tokens against the budget if within limit,
// returning a release callback to apply actual cost when the request finishes.
func (b *BudgetGuardImpl) Reserve(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) (BudgetResult, func(actualCost float64)) {
	if b.policyResolver != nil && limit == 0 && window == 0 {
		if budgetConfig, ok := b.getBudgetConfigFromSnapshot(ctx, key); ok {
			limit = budgetConfig.Limit
			window = budgetConfig.Window
		}
	}
	tracker := b.getOrCreateTracker(key, limit, window, 0.8*limit)
	return tracker.reserve(estimatedCost)
}

// getBudgetConfigFromSnapshot retrieves budget config from the already-synced
// snapshot by matching the budget key directly.
func (b *BudgetGuardImpl) getBudgetConfigFromSnapshot(ctx context.Context, key string) (BudgetConfig, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Direct lookup by key in the internal budgets map (populated by SyncFromSnapshot)
	if tracker, ok := b.budgets[key]; ok {
		tracker.mu.Lock()
		config := BudgetConfig{
			Key:     tracker.key,
			Limit:   tracker.limit,
			Window:  tracker.window,
			Scope:   tracker.scope,
			AlertAt: tracker.alertAt,
		}
		tracker.mu.Unlock()
		return config, true
	}

	return BudgetConfig{}, false
}

// normalizeScope maps short scope names to their full names
func normalizeScope(scope string) string {
	switch scope {
	case "org":
		return "organization"
	default:
		return scope
	}
}

// getOrCreateTracker retrieves or creates a budget tracker.
// If key is a scope-based key (e.g., "org:xxx"), parse the scope from it.
func (b *BudgetGuardImpl) getOrCreateTracker(key string, limit float64, window time.Duration, alertAt float64) *budgetTracker {
	b.mu.RLock()
	tracker, ok := b.budgets[key]
	b.mu.RUnlock()

	if ok {
		return tracker
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if tracker, ok := b.budgets[key]; ok {
		return tracker
	}

	// Parse scope from key if it's a scope-based key
	scope := ""
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		scope = parts[0]
	}

	tracker = &budgetTracker{
		key:         key,
		scope:       scope,
		limit:       limit,
		window:      window,
		spent:       0,
		windowStart: b.clock(),
		alertAt:     alertAt,
	}
	b.budgets[key] = tracker
	return tracker
}

// check verifies if the estimated cost fits within budget.
func (t *budgetTracker) check(estimatedCost float64) BudgetResult {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	// Reset window if expired
	if now.Sub(t.windowStart) > t.window {
		t.spent = 0
		t.windowStart = now
	}

	projected := t.spent + estimatedCost
	allowed := projected <= t.limit

	return BudgetResult{
		Allowed:   allowed,
		Current:   t.spent,
		Limit:     t.limit,
		Projected: projected,
		AlertAt:   t.alertAt,
		Scope:     t.scope,
		Key:       t.key,
	}
}

// reserve atomically checks and reserves estimated cost/tokens under mutex.
func (t *budgetTracker) reserve(estimatedCost float64) (BudgetResult, func(actualCost float64)) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	// Reset window if expired
	if now.Sub(t.windowStart) > t.window {
		t.spent = 0
		t.windowStart = now
	}

	projected := t.spent + estimatedCost
	allowed := projected <= t.limit

	res := BudgetResult{
		Allowed:   allowed,
		Current:   t.spent,
		Limit:     t.limit,
		Projected: projected,
		AlertAt:   t.alertAt,
		Scope:     t.scope,
		Key:       t.key,
	}

	if !allowed {
		return res, func(_ float64) {}
	}

	// Atomically reserve
	t.spent = projected

	committed := false
	release := func(actualCost float64) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if committed {
			return
		}
		committed = true
		if actualCost < 0 {
			actualCost = 0
		}
		t.spent = t.spent - estimatedCost + actualCost
		if t.spent < 0 {
			t.spent = 0
		}
	}

	return res, release
}

// RecordUsage records actual usage against the budget.
func (b *BudgetGuardImpl) RecordUsage(ctx context.Context, key string, cost float64) error {
	tracker := b.getOrCreateTracker(key, 0, 0, 0)
	if cost < 0 {
		return nil
	}

	tracker.mu.Lock()
	defer tracker.mu.Unlock()

	now := time.Now()
	if now.Sub(tracker.windowStart) > tracker.window {
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

// ResetBudget resets a budget tracker (e.g., on window rollover).
func (b *BudgetGuardImpl) ResetBudget(key string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.budgets, key)
}

// Close stops the budget guard (no-op for now, but useful for cleanup).
func (b *BudgetGuardImpl) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.budgets = make(map[string]*budgetTracker)
}

// SyncFromSnapshot loads budget configurations from a policy snapshot.
func (b *BudgetGuardImpl) SyncFromSnapshot(snapshot PolicySnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for key, config := range snapshot.Budgets {
		if tracker, ok := b.budgets[key]; ok {
			// Update existing tracker with new config
			tracker.mu.Lock()
			tracker.limit = config.Limit
			tracker.window = config.Window
			tracker.alertAt = config.AlertAt
			tracker.scope = config.Scope
			tracker.mu.Unlock()
		} else {
			// Create new tracker with config from snapshot
			b.budgets[key] = &budgetTracker{
				key:         key,
				scope:       config.Scope,
				limit:       config.Limit,
				window:      config.Window,
				spent:       0,
				windowStart: b.clock(),
				alertAt:     config.AlertAt,
			}
		}
	}
}