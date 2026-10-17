package gatewayv2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func getLiveTestRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr := os.Getenv("AIGW_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("AIGW_REDIS_PASSWORD")
	if password == "" {
		password = "change-me"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping Redis budget regression tests: local Redis unavailable on %s (%v)", addr, err)
		return nil
	}
	return client
}

// 1. Requests below, at, and above the limit against live Redis
func TestBudgetRedis_BelowAtAndAboveLimit(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	limit := 10.0
	window := 1 * time.Hour

	// Below limit: $4.00 on $10.00 limit -> Allowed
	res1, release1 := bg.Reserve(ctx, budgetKey, limit, window, 4.0, "req-1")
	if !res1.Allowed {
		t.Fatalf("expected $4.00 request to be allowed, got denied")
	}
	if res1.Degraded {
		t.Fatalf("expected Degraded=false with healthy Redis")
	}
	release1(4.0, nil)

	spent, limitVal, _, ok := bg.GetBudget(budgetKey)
	if !ok || spent != 4.0 || limitVal != 10.0 {
		t.Fatalf("expected spent=4.0, got spent=%.2f, limit=%.2f, ok=%v", spent, limitVal, ok)
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

// 2. Concurrent requests competing for the final available budget in Redis
func TestBudgetRedis_ConcurrentCompetition(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

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
			reqID := fmt.Sprintf("req-worker-%s-%d", uuid.NewString(), workerID)
			res, release := bg.Reserve(ctx, budgetKey, limit, window, requestEstimate, reqID)
			if res.Allowed {
				atomic.AddInt32(&allowedCount, 1)
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

// 3. Window expiration and deterministic reset in UTC
func TestBudgetRedis_WindowExpirationAndReset(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	limit := 10.0
	window := 24 * time.Hour

	// Fill budget to limit in current window
	res1, release1 := bg.Reserve(ctx, budgetKey, limit, window, 10.0, "req-win-1")
	if !res1.Allowed {
		t.Fatalf("expected initial reservation to succeed")
	}
	release1(10.0, nil)

	// Verify budget is exhausted
	resExhausted, _ := bg.Reserve(ctx, budgetKey, limit, window, 1.0, "req-win-exhausted")
	if resExhausted.Allowed {
		t.Fatalf("expected budget to be exhausted in current window")
	}

	// Advance simulated clock by 25 hours (past window boundary into next UTC day)
	simulatedNow := time.Now().UTC().Add(25 * time.Hour)
	bg.clock = func() time.Time {
		return simulatedNow
	}

	// In the new window, expired counters must NOT continue blocking new requests
	resNewWindow, releaseNew := bg.Reserve(ctx, budgetKey, limit, window, 4.0, "req-win-new")
	if !resNewWindow.Allowed {
		t.Fatalf("expected window rollover to reset budget and allow request, got denied")
	}
	releaseNew(4.0, nil)

	spent, _, _, _ := bg.GetBudget(budgetKey)
	if spent != 4.0 {
		t.Fatalf("expected spent in new window to be 4.0, got %.2f", spent)
	}
}

// 4. Reservation expiry and crash recovery without permanent aggregate inflation
func TestBudgetRedis_ReservationExpiryAndNoInflation(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger).WithTTL(30 * time.Millisecond)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	limit := 5.0
	window := 1 * time.Hour

	// Abandoned request reserves all $5.00
	resAbandoned, releaseAbandoned := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-abandoned")
	if !resAbandoned.Allowed {
		t.Fatalf("expected initial reservation to succeed")
	}

	// Immediate second request must be rejected because $5 is active reserved
	resImmediate, _ := bg.Reserve(ctx, budgetKey, limit, window, 1.0, "req-immediate")
	if resImmediate.Allowed {
		t.Fatalf("expected immediate request to be rejected while budget is reserved")
	}

	// Wait for safety TTL to expire (simulating gateway crash or client abandonment)
	time.Sleep(50 * time.Millisecond)

	// Third request must succeed because Lua script sweeps expired reservations and decrements reservedKey
	resRecovered, releaseRecovered := bg.Reserve(ctx, budgetKey, limit, window, 2.0, "req-recovered")
	if !resRecovered.Allowed {
		t.Fatalf("expected abandoned reservation to be purged without inflating reserved spend, but was rejected")
	}
	releaseRecovered(2.0, nil)

	// Now simulate late zombie delivery of releaseAbandoned(5.0, nil).
	// It must NOT double-decrement reservedKey below 0 or corrupt state!
	releaseAbandoned(5.0, nil)

	spent, reserved, _, ok := bg.GetBudgetState(budgetKey)
	if !ok {
		t.Fatalf("expected ok=true for budget state")
	}
	if reserved != 0 {
		t.Fatalf("expected reserved to be 0, got %.2f", reserved)
	}
	// Total spent should reflect the actual completed usage
	if spent < 2.0 {
		t.Fatalf("expected spent to be at least 2.0, got %.2f", spent)
	}
}

// 5. Duplicate finalization idempotency (zero double-counting)
func TestBudgetRedis_DuplicateFinalizationIdempotency(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	limit := 10.0
	window := 1 * time.Hour

	res, release := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-dup-redis")
	if !res.Allowed {
		t.Fatal("expected reserve to succeed")
	}

	// First release commits $5.00
	release(5.0, nil)

	spent1, _, _, _ := bg.GetBudget(budgetKey)
	if spent1 != 5.0 {
		t.Fatalf("expected spent=5.0, got %.2f", spent1)
	}

	// Duplicate calls to release must be completely absorbed without double counting
	release(5.0, nil)
	release(5.0, nil)

	spent2, reserved2, _, _ := bg.GetBudgetState(budgetKey)
	if spent2 != 5.0 {
		t.Fatalf("duplicate release caused double counting! expected 5.0, got %.2f", spent2)
	}
	if reserved2 != 0 {
		t.Fatalf("expected reserved to be 0 after release, got %.2f", reserved2)
	}
}

// 6. Failed requests and interrupted streams
func TestBudgetRedis_FailedAndInterruptedStreams(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	limit := 10.0
	window := 1 * time.Hour

	// Case A: Upstream provider failure (e.g. 504 Timeout)
	_, releaseFail := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-fail-redis")
	releaseFail(0, errors.New("upstream provider error"))

	spent, reserved, _, _ := bg.GetBudgetState(budgetKey)
	if spent != 0 || reserved != 0 {
		t.Fatalf("failed request must not accrue spend: spent=%.2f, reserved=%.2f", spent, reserved)
	}

	// Case B: Interrupted stream (client disconnected after 1.75 tokens consumed)
	_, releaseDisconnect := bg.Reserve(ctx, budgetKey, limit, window, 5.0, "req-disc-redis")
	releaseDisconnect(1.75, ErrClientDisconnected)

	spent, reserved, _, _ = bg.GetBudgetState(budgetKey)
	if spent != 1.75 {
		t.Fatalf("interrupted stream should charge only actual consumption: expected 1.75, got %.2f", spent)
	}
	if reserved != 0 {
		t.Fatalf("expected reservation to be completely cleared: got %.2f", reserved)
	}
}

// 7. Redis unavailable before reservation: Fail-Closed vs Degraded Mode
func TestBudgetRedis_UnavailableBeforeReservation(t *testing.T) {
	deadRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54321", // non-existent Redis port
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadRedis.Close()

	logger := slog.Default()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()

	t.Run("FailClosedMode", func(t *testing.T) {
		bg := NewBudgetGuardWithRedis(deadRedis, logger).WithMode(BudgetModeFailClosed)
		res, _ := bg.Reserve(ctx, budgetKey, 10.0, time.Hour, 2.0, "req-dead-fc")
		if res.Allowed {
			t.Fatalf("expected fail-closed mode to reject requests when Redis is dead")
		}
		if !res.Degraded {
			t.Fatalf("expected Degraded=true when operating under outage")
		}
	})

	t.Run("DegradedLocalMode", func(t *testing.T) {
		bg := NewBudgetGuardWithRedis(deadRedis, logger).WithMode(BudgetModeDegradedLocal)
		res, release := bg.Reserve(ctx, budgetKey, 10.0, time.Hour, 2.0, "req-dead-deg")
		if !res.Allowed {
			t.Fatalf("expected degraded mode to allow request via process-local tracker")
		}
		if !res.Degraded {
			t.Fatalf("expected Degraded=true in degraded mode")
		}
		release(2.0, nil)

		spent, _, _, ok := bg.GetBudget(budgetKey)
		if !ok || spent != 2.0 {
			t.Fatalf("expected local tracker to record spent=2.0, got spent=%.2f", spent)
		}
	})
}

// 8. Redis unavailable after reservation but before finalization
func TestBudgetRedis_UnavailableAfterReservationBeforeFinalization(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger).WithTTL(50 * time.Millisecond)
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer bg.ResetBudget(budgetKey)

	// Step 1: Successful reservation against live Redis
	res, release := bg.Reserve(ctx, budgetKey, 10.0, time.Hour, 3.0, "req-mid-outage")
	if !res.Allowed {
		t.Fatalf("expected initial reservation to succeed")
	}

	// Step 2: Swap client to dead connection to simulate network severance mid-flight
	deadRedis := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:54321",
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	bg.redis = deadRedis

	// Step 3: Finalization must not panic, must release local tracker cleanly
	release(3.0, nil)
	deadRedis.Close()

	// Step 4: Restore live Redis connection
	bg.redis = client
	// After safety TTL, the expired reservation in Redis is purged automatically on next call
	time.Sleep(60 * time.Millisecond)

	resAfter, releaseAfter := bg.Reserve(ctx, budgetKey, 10.0, time.Hour, 2.0, "req-after-recovery")
	if !resAfter.Allowed {
		t.Fatalf("expected new request to succeed after Redis connectivity is restored")
	}
	releaseAfter(2.0, nil)
}

// 9. Restart behavior across separate gateway instances sharing Redis
func TestBudgetRedis_RestartBehavior(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	ctx := context.Background()
	budgetKey := "org:" + uuid.NewString()
	defer client.Del(ctx, "aigw:budget:"+budgetKey+":spent", "aigw:budget:"+budgetKey+":reserved")

	limit := 50.0
	window := 24 * time.Hour

	// Gateway Instance 1 spends $35.00
	gw1 := NewBudgetGuardWithRedis(client, logger)
	res1, release1 := gw1.Reserve(ctx, budgetKey, limit, window, 35.0, "req-restart-gw1")
	if !res1.Allowed {
		t.Fatalf("expected gw1 request to be allowed")
	}
	release1(35.0, nil)

	// Gateway Instance 2 starts fresh (simulating gateway restart / horizontal scale)
	gw2 := NewBudgetGuardWithRedis(client, logger)

	// Request for $20.00 exceeds remaining ($35 + $20 = $55 > $50) -> Must be Denied!
	resDeny, _ := gw2.Reserve(ctx, budgetKey, limit, window, 20.0, "req-restart-gw2-deny")
	if resDeny.Allowed {
		t.Fatalf("expected gw2 to reject $20 request over budget (35 spent on 50 limit)")
	}

	// Request for $10.00 fits in remaining ($35 + $10 = $45 <= $50) -> Must be Allowed!
	resAllow, releaseAllow := gw2.Reserve(ctx, budgetKey, limit, window, 10.0, "req-restart-gw2-allow")
	if !resAllow.Allowed {
		t.Fatalf("expected gw2 to allow $10 request within remaining budget")
	}
	releaseAllow(10.0, nil)

	spent, _, _, _ := gw2.GetBudget(budgetKey)
	if spent != 45.0 {
		t.Fatalf("expected total spend in Redis across instances to be 45.0, got %.2f", spent)
	}
}

// 10. Cross-organization isolation in Redis
func TestBudgetRedis_CrossOrganizationIsolation(t *testing.T) {
	client := getLiveTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	bg := NewBudgetGuardWithRedis(client, logger)
	ctx := context.Background()

	org1Key := "org:" + uuid.NewString()
	org2Key := "org:" + uuid.NewString()
	defer bg.ResetBudget(org1Key)
	defer bg.ResetBudget(org2Key)

	limit := 10.0
	window := 1 * time.Hour

	// Org 1 exhausts its budget in Redis
	_, release1 := bg.Reserve(ctx, org1Key, limit, window, 10.0, "req-org1")
	release1(10.0, nil)

	res1, _ := bg.Reserve(ctx, org1Key, limit, window, 1.0, "req-org1-deny")
	if res1.Allowed {
		t.Fatalf("expected Org 1 to be denied when budget is exhausted")
	}

	// Org 2 must be completely isolated and allowed
	res2, release2 := bg.Reserve(ctx, org2Key, limit, window, 5.0, "req-org2-allow")
	if !res2.Allowed {
		t.Fatalf("expected Org 2 to be allowed regardless of Org 1 exhaustion")
	}
	release2(5.0, nil)

	spent1, _, _, _ := bg.GetBudget(org1Key)
	spent2, _, _, _ := bg.GetBudget(org2Key)

	if spent1 != 10.0 || spent2 != 5.0 {
		t.Fatalf("cross-org state collision in Redis! spent1=%.2f, spent2=%.2f", spent1, spent2)
	}
}
