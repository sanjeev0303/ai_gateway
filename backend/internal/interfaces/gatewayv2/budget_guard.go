// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"sync"
	"time"
)

// BudgetGuardImpl implements budget tracking and enforcement.
type BudgetGuardImpl struct {
	mu      sync.RWMutex
	budgets map[string]*budgetTracker
	clock   func() time.Time
}

type budgetTracker struct {
	key       string
	limit     float64
	window    time.Duration
	spent     float64
	windowStart time.Time
	alertAt   float64
	mu        sync.Mutex
}

// NewBudgetGuard creates a new budget guard.
func NewBudgetGuard() *BudgetGuardImpl {
	return &BudgetGuardImpl{
		budgets: make(map[string]*budgetTracker),
		clock:   time.Now,
	}
}

// Check verifies if the estimated cost fits within the budget.
func (b *BudgetGuardImpl) Check(ctx context.Context, key string, limit float64, window time.Duration, estimatedCost float64) BudgetResult {
	tracker := b.getOrCreateTracker(key, limit, window, 0.8*limit) // default alert at 80%
	return tracker.check(estimatedCost)
}

// getOrCreateTracker retrieves or creates a budget tracker.
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

	tracker = &budgetTracker{
		key:         key,
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
		Scope:     "organization", // default
		Key:       t.key,
	}
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