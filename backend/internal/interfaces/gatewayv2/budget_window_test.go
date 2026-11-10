package gatewayv2

import (
	"context"
	"testing"
	"time"
)

const windowTestKey = "org:o:organization:o"

func guardAt(now *time.Time) *BudgetGuardImpl {
	g := NewBudgetGuard()
	g.clock = func() time.Time { return *now }
	return g
}

func snapshotWith(window time.Duration) PolicySnapshot {
	return PolicySnapshot{Budgets: map[string]BudgetConfig{
		windowTestKey: {Key: windowTestKey, Limit: 10, Window: window, Scope: "organization", LimitType: "spend"},
	}}
}

// spend reserves and commits amount, the way a finished request does.
func spend(t *testing.T, g *BudgetGuardImpl, window time.Duration, amount float64, reqID string) {
	t.Helper()
	res, release := g.Reserve(context.Background(), windowTestKey, 10, window, amount, reqID)
	if !res.Allowed {
		t.Fatalf("reserving %.0f was refused: %+v", amount, res)
	}
	release(amount, nil)
}

// The gateway syncs the snapshot into the guard on every request. Syncing must
// not move the window forward by itself, or the tracker never sees the new
// window and spend carries over from yesterday.
func TestBudgetWindow_SnapshotSyncStillRollsOver(t *testing.T) {
	now := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	g := guardAt(&now)
	snap := snapshotWith(24 * time.Hour)

	g.SyncFromSnapshot(snap)
	spend(t, g, 24*time.Hour, 8, "r1")

	now = time.Date(2026, 3, 11, 9, 0, 0, 0, time.UTC)
	g.SyncFromSnapshot(snap)
	res, _ := g.Reserve(context.Background(), windowTestKey, 10, 24*time.Hour, 5, "r2")
	if !res.Allowed {
		t.Fatalf("a new day should start from zero, but 5 was refused with %.0f already counted", res.Current)
	}
}

// A monthly budget is a calendar month. On the 31st more than 30 days have
// passed since the 1st, which must not reset month-to-date spend.
func TestBudgetWindow_MonthDoesNotResetOnThe31st(t *testing.T) {
	month := 30 * 24 * time.Hour
	now := time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)
	g := guardAt(&now)
	g.SyncFromSnapshot(snapshotWith(month))
	spend(t, g, month, 8, "r1")

	now = time.Date(2026, 12, 31, 0, 0, 10, 0, time.UTC)
	g.SyncFromSnapshot(snapshotWith(month))
	res, _ := g.Reserve(context.Background(), windowTestKey, 10, month, 5, "r2")
	if res.Allowed {
		t.Fatal("December spend was reset on the 31st: 8 + 5 should exceed the limit of 10")
	}

	now = time.Date(2027, 1, 1, 0, 0, 1, 0, time.UTC)
	res, _ = g.Reserve(context.Background(), windowTestKey, 10, month, 5, "r3")
	if !res.Allowed {
		t.Fatal("January should start from zero")
	}
}
