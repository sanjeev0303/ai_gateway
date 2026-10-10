// Contract tests for the budget guard: allowance checks, window reset,
// usage recording and budget queries.
package gatewayv2

import (
	"context"
	"testing"
	"time"
)

const budgetKey = "org:budget"

func TestBudgetGuard_AllowsUnderLimit(t *testing.T) {
	bg := NewBudgetGuard()

	res := bg.Check(context.Background(), budgetKey, 100, time.Minute, 40)

	if !res.Allowed {
		t.Fatal("expected cost within limit to be allowed")
	}
	if res.Current != 0 {
		t.Fatalf("expected zero spent on fresh tracker, got %v", res.Current)
	}
	if res.Projected != 40 {
		t.Fatalf("expected projected cost 40, got %v", res.Projected)
	}
	if res.Limit != 100 {
		t.Fatalf("expected limit 100, got %v", res.Limit)
	}
	if res.AlertAt != 80 {
		t.Fatalf("expected default alert threshold 80 (80%% of limit), got %v", res.AlertAt)
	}
	if res.Scope != "org" {
		t.Fatalf("expected default scope org, got %q", res.Scope)
	}
	if res.Key != budgetKey {
		t.Fatalf("expected key %q, got %q", budgetKey, res.Key)
	}
}

func TestBudgetGuard_DeniesAboveLimit(t *testing.T) {
	bg := NewBudgetGuard()

	res := bg.Check(context.Background(), budgetKey, 100, time.Minute, 100.01)

	if res.Allowed {
		t.Fatal("expected cost above limit to be denied")
	}
	if res.Projected != 100.01 {
		t.Fatalf("expected projected cost 100.01, got %v", res.Projected)
	}
}

func TestBudgetGuard_DeniesOverLimit(t *testing.T) {
	bg := NewBudgetGuard()

	res := bg.Check(context.Background(), budgetKey, 50, time.Minute, 60)

	if res.Allowed {
		t.Fatal("expected cost above limit to be denied")
	}
	if res.Projected != 60 {
		t.Fatalf("expected projected cost 60, got %v", res.Projected)
	}
}

func TestBudgetGuard_TracksAccumulatedUsage(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()

	bg.Check(ctx, budgetKey, 100, time.Minute, 20)
	if err := bg.RecordUsage(ctx, budgetKey, 30); err != nil {
		t.Fatalf("unexpected error recording usage: %v", err)
	}

	res := bg.Check(ctx, budgetKey, 100, time.Minute, 50)

	if !res.Allowed {
		t.Fatal("expected 30 spent + 50 projected to be allowed under limit 100")
	}
	if res.Current != 30 {
		t.Fatalf("expected accumulated spend 30, got %v", res.Current)
	}
	if res.Projected != 80 {
		t.Fatalf("expected projected 80, got %v", res.Projected)
	}
}

func TestBudgetGuard_IgnoresNegativeCost(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()

	bg.Check(ctx, budgetKey, 100, time.Minute, 0)

	if err := bg.RecordUsage(ctx, budgetKey, -50); err != nil {
		t.Fatalf("expected nil error for negative cost, got %v", err)
	}
	if err := bg.RecordUsage(ctx, budgetKey, 10); err != nil {
		t.Fatalf("unexpected error recording positive usage: %v", err)
	}

	res := bg.Check(ctx, budgetKey, 100, time.Minute, 0)
	if res.Current != 10 {
		t.Fatalf("expected negative cost ignored (spent 10), got %v", res.Current)
	}
}

func TestBudgetGuard_WindowReset(t *testing.T) {
	bg := NewBudgetGuard()
	now := time.Now()

	bg.mu.Lock()
	bg.budgets[budgetKey] = &budgetTracker{
		key:         budgetKey,
		limit:       100,
		window:      time.Minute,
		spent:       90,
		windowStart: now.Add(-2 * time.Minute),
		alertAt:     80,
	}
	bg.mu.Unlock()

	res := bg.Check(context.Background(), budgetKey, 100, time.Minute, 10)

	if !res.Allowed {
		t.Fatal("expected window rollover to reset spend and allow request")
	}
	if res.Current != 0 {
		t.Fatalf("expected spend reset to 0 after window expiry, got %v", res.Current)
	}
	if res.Projected != 10 {
		t.Fatalf("expected projected 10 after reset, got %v", res.Projected)
	}
}

func TestBudgetGuard_NoResetWithinWindow(t *testing.T) {
	bg := NewBudgetGuard()
	now := time.Now()

	bg.mu.Lock()
	bg.budgets[budgetKey] = &budgetTracker{
		key:         budgetKey,
		limit:       100,
		window:      time.Minute,
		spent:       40,
		windowStart: now.Add(-30 * time.Second),
		alertAt:     80,
	}
	bg.mu.Unlock()

	res := bg.Check(context.Background(), budgetKey, 100, time.Minute, 40)

	if !res.Allowed {
		t.Fatal("expected spend to persist within the active window")
	}
	if res.Current != 40 {
		t.Fatalf("expected spend kept at 40 within window, got %v", res.Current)
	}
	if res.Projected != 80 {
		t.Fatalf("expected projected 80, got %v", res.Projected)
	}
}

func TestBudgetGuard_GetBudget(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()

	if _, _, _, ok := bg.GetBudget("unknown"); ok {
		t.Fatal("expected ok=false for unknown key")
	}

	bg.Check(ctx, budgetKey, 100, time.Minute, 10)
	if err := bg.RecordUsage(ctx, budgetKey, 25); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spent, limit, windowStart, ok := bg.GetBudget(budgetKey)
	if !ok {
		t.Fatal("expected ok=true for tracked key")
	}
	if spent != 25 {
		t.Fatalf("expected spent 25, got %v", spent)
	}
	if limit != 100 {
		t.Fatalf("expected limit from tracking check, got %v", limit)
	}
	if windowStart.IsZero() {
		t.Fatal("expected non-zero window start")
	}
}

func TestBudgetGuard_ResetBudget(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()

	bg.Check(ctx, budgetKey, 100, time.Minute, 10)
	if err := bg.RecordUsage(ctx, budgetKey, 25); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, _, ok := bg.GetBudget(budgetKey); !ok {
		t.Fatal("expected key tracked before reset")
	}

	bg.ResetBudget(budgetKey)

	if _, _, _, ok := bg.GetBudget(budgetKey); ok {
		t.Fatal("expected key untracked after reset")
	}
}
