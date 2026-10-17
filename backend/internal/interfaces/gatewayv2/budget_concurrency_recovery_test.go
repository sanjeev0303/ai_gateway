package gatewayv2

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// 1. Requests below, at, and above the limit
func TestBudget_BelowAtAndAboveLimit(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	// Below limit: $4.00 on $10.00 limit -> Allowed
	res1, release1 := bg.Reserve(ctx, budgetKey, limit, window, 4.0, "req-1")
	if !res1.Allowed {
		t.Fatalf("expected $4.00 request to be allowed, got denied")
	}
	release1(4.0, nil)

	spent, limitVal, _, ok := bg.GetBudget(budgetKey)
	if !ok || spent != 4.0 || limitVal != 10.0 {
		t.Fatalf("expected spent=4.0, got spent=%.2f, ok=%v", spent, ok)
	}

	// At limit: $6.00 on remaining $6.00 -> Allowed
	res2, release2 := bg.Reserve(ctx, budgetKey, limit, window, 6.0, "req-2")
	if !res2.Allowed {
		t.Fatalf("expected exactly at-limit request to be allowed")
	}
	release2(6.0, nil)

	spent, _, _, _ = bg.GetBudget(budgetKey)
	if spent != 10.0 {
		t.Fatalf("expected spent=10.0, got %.2f", spent)
	}

	// Above limit: $0.01 on exhausted budget -> Denied
	res3, _ := bg.Reserve(ctx, budgetKey, limit, window, 0.01, "req-3")
	if res3.Allowed {
		t.Fatalf("expected request above budget to be rejected")
	}
}

// 2. Concurrent requests competing for the final available budget
func TestBudget_ConcurrentRequestsCompetition(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	const concurrentWorkers = 20
	const requestEstimate = 2.0 // Exactly 5 requests can fit in $10.00

	var allowedCount int32
	var rejectedCount int32

	var wg sync.WaitGroup
	wg.Add(concurrentWorkers)

	for i := 0; i < concurrentWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			reqID := "req-worker-" + uuid.NewString()
			res, release := bg.Reserve(ctx, budgetKey, limit, window, requestEstimate, reqID)
			if res.Allowed {
				atomic.AddInt32(&allowedCount, 1)
				// Hold the reservation briefly to simulate streaming
				time.Sleep(5 * time.Millisecond)
				release(requestEstimate, nil)
			} else {
				atomic.AddInt32(&rejectedCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if allowedCount != 5 {
		t.Fatalf("expected exactly 5 requests allowed for $10 limit at $2/req, got allowed=%d, rejected=%d",
			allowedCount, rejectedCount)
	}
	if rejectedCount != 15 {
		t.Fatalf("expected exactly 15 requests rejected, got rejected=%d", rejectedCount)
	}

	spent, _, _, _ := bg.GetBudget(budgetKey)
	if spent != 10.0 {
		t.Fatalf("expected final committed spent=10.0, got %.2f", spent)
	}
}

// 3. Restarting the gateway (reconciles with SpendLoader or Redis)
func TestBudget_RestartGatewayWithPersistence(t *testing.T) {
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 50.0
	window := 24 * time.Hour

	// Simulate persistent ledger returning $35.00 spent from past requests
	spendLedger := map[string]float64{
		budgetKey: 35.0,
	}
	mockSpendLoader := func(_ context.Context, key string, _ time.Time) (float64, error) {
		return spendLedger[key], nil
	}

	// Gateway instance 1 starts up with spend loader
	gw1 := NewBudgetGuard().WithSpendLoader(mockSpendLoader)

	// Verify gw1 initializes from ledger
	res, release := gw1.Reserve(ctx, budgetKey, limit, window, 10.0, "req-restart-1")
	if !res.Allowed {
		t.Fatalf("expected request within remaining allowance (35 + 10 <= 50) to be allowed")
	}
	release(10.0, nil)

	// Update ledger with new total spend
	spendLedger[budgetKey] = 45.0

	// Gateway instance 2 "restarts" (cold boot with fresh memory)
	gw2 := NewBudgetGuard().WithSpendLoader(mockSpendLoader)

	// Now remaining is 5.0 (50 - 45). A request for 10.0 must be denied!
	resDeny, _ := gw2.Reserve(ctx, budgetKey, limit, window, 10.0, "req-restart-2")
	if resDeny.Allowed {
		t.Fatalf("expected 10.0 request to be denied after gateway restart (45 spent on 50 limit)")
	}

	// A request for 4.0 fits in the remaining 5.0
	resAllow, releaseAllow := gw2.Reserve(ctx, budgetKey, limit, window, 4.0, "req-restart-3")
	if !resAllow.Allowed {
		t.Fatalf("expected 4.0 request to fit in remaining budget after restart")
	}
	releaseAllow(4.0, nil)
}

// 4. Duplicate usage events (idempotency and zero double-counting)
func TestBudget_DuplicateUsageEventsIdempotency(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	res, release := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-dup-1")
	if !res.Allowed {
		t.Fatal("expected reserve to succeed")
	}

	// First release commits $5.00
	release(5.0, nil)

	spent, _, _, _ := bg.GetBudget(budgetKey)
	if spent != 5.0 {
		t.Fatalf("expected spent=5.0, got %.2f", spent)
	}

	// Duplicate delivery of the exact same event / release call
	release(5.0, nil)
	release(5.0, nil)

	spentAfterDupes, _, _, _ := bg.GetBudget(budgetKey)
	if spentAfterDupes != 5.0 {
		t.Fatalf("duplicate release caused double-counting! expected 5.0, got %.2f", spentAfterDupes)
	}
}

// 5. Failed and interrupted provider requests
func TestBudget_FailedAndInterruptedProviderRequests(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	// Case A: Upstream provider failure (e.g. 504 Gateway Timeout)
	_, releaseFail := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-fail-1")
	// Provider failed, release with error and 0 actual cost
	releaseFail(0, errors.New("upstream provider timeout"))

	spent, reserved, limitVal, _ := bg.GetBudgetState(budgetKey)
	if spent != 0 || reserved != 0 {
		t.Fatalf("failed request should not accrue spend: got spent=%.2f, reserved=%.2f", spent, reserved)
	}
	if limitVal != 10.0 {
		t.Fatalf("limit should remain 10.0, got %.2f", limitVal)
	}

	// Case B: Interrupted stream (client disconnected mid-stream)
	_, releaseDisconnect := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-disconnect-1")
	// Client disconnected after consuming 1.25 worth of tokens
	releaseDisconnect(1.25, ErrClientDisconnected)

	spent, reserved, _, _ = bg.GetBudgetState(budgetKey)
	if spent != 1.25 {
		t.Fatalf("interrupted stream should charge only actual tokens consumed: expected 1.25, got %.2f", spent)
	}
	if reserved != 0 {
		t.Fatalf("reservation should be completely cleared: got reserved=%.2f", reserved)
	}
}

// 6. Reservation expiration and recovery for abandoned requests
func TestBudget_ReservationExpirationAndRecovery(t *testing.T) {
	bg := NewBudgetGuard()
	// Set short safety TTL for test
	bg.resTTL = 20 * time.Millisecond
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 5.0
	window := 1 * time.Hour

	// Abandoned request reserves all $5.00
	res1, _ := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-abandoned")
	if !res1.Allowed {
		t.Fatal("expected initial reservation to succeed")
	}

	// Immediate second request must be rejected because $5 is reserved
	res2, _ := bg.Reserve(ctx, budgetKey, limit, window, 1.0, "req-immediate")
	if res2.Allowed {
		t.Fatal("expected second request to be rejected while first is in-flight")
	}

	// Wait for reservation to expire (safety timeout)
	time.Sleep(30 * time.Millisecond)

	// Now third request should succeed as expired reservation is purged
	res3, release3 := bg.Reserve(ctx, budgetKey, limit, window, 2.0, "req-recovered")
	if !res3.Allowed {
		t.Fatalf("expected abandoned reservation to expire and allow new request, got rejected")
	}
	release3(2.0, nil)

	spent, reserved, _, _ := bg.GetBudgetState(budgetKey)
	if spent != 2.0 || reserved != 0 {
		t.Fatalf("expected spent=2.0, reserved=0, got spent=%.2f, reserved=%.2f", spent, reserved)
	}
}

// 7. Cross-organization isolation
func TestBudget_CrossOrganizationIsolation(t *testing.T) {
	bg := NewBudgetGuard()
	ctx := context.Background()
	org1Key := "org:" + uuid.NewString()
	org2Key := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	// Org 1 exhausts its budget
	_, release1 := bg.Reserve(ctx, org1Key, limit, window, 10.0, "req-org1")
	release1(10.0, nil)

	// Org 1 is denied
	res1, _ := bg.Reserve(ctx, org1Key, limit, window, 1.0, "req-org1-deny")
	if res1.Allowed {
		t.Fatal("expected org1 to be rejected when budget exhausted")
	}

	// Org 2 must be completely unaffected and allowed
	res2, release2 := bg.Reserve(ctx, org2Key, limit, window, 5.0, "req-org2-allow")
	if !res2.Allowed {
		t.Fatal("expected org2 to be allowed regardless of org1 budget exhaustion")
	}
	release2(5.0, nil)

	spent1, _, _, _ := bg.GetBudget(org1Key)
	spent2, _, _, _ := bg.GetBudget(org2Key)

	if spent1 != 10.0 || spent2 != 5.0 {
		t.Fatalf("cross-org leakage detected! spent1=%.2f, spent2=%.2f", spent1, spent2)
	}
}

// 8. Persistence outage graceful fallback
func TestBudget_PersistenceOutageGracefulFallback(t *testing.T) {
	// Connect to an unreachable Redis host
	deadRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54321", // non-existent Redis port
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadRedis.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(deadRedis, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	limit := 10.0
	window := 1 * time.Hour

	// Must not panic, must fall back to in-memory tracker
	res, release := bg.Reserve(ctx, budgetKey, limit, window, 3.0, "req-outage-1")
	if !res.Allowed {
		t.Fatalf("expected fallback to in-memory tracker to allow request under limit")
	}
	release(3.0, nil)

	spent, _, _, ok := bg.GetBudget(budgetKey)
	if !ok || spent != 3.0 {
		t.Fatalf("expected in-memory fallback to record spent=3.0, got spent=%.2f, ok=%v", spent, ok)
	}
}
