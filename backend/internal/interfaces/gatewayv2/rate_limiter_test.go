// Contract tests for the rate limiters: token bucket burst/refill behavior
// and sliding window pruning. take() reads time.Now() directly, so refill
// tests seed the bucket state in-package instead of sleeping.
package gatewayv2

import (
	"context"
	"testing"
	"time"
)

func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	rl := NewRateLimiter()
	t.Cleanup(rl.Close)
	ctx := context.Background()
	key := "cred:anthropic:acct-1"

	for i := 0; i < 3; i++ {
		res := rl.Check(ctx, key, 3, time.Hour)
		if !res.Allowed {
			t.Fatalf("call %d: expected token available under limit 3", i+1)
		}
		if res.Remaining != 2-i {
			t.Fatalf("call %d: expected remaining %d, got %d", i+1, 2-i, res.Remaining)
		}
	}
}

func TestRateLimiter_DeniesWhenExhausted(t *testing.T) {
	rl := NewRateLimiter()
	t.Cleanup(rl.Close)
	ctx := context.Background()
	key := "cred:anthropic:acct-1"

	for i := 0; i < 3; i++ {
		rl.Check(ctx, key, 3, time.Hour)
	}

	res := rl.Check(ctx, key, 3, time.Hour)

	if res.Allowed {
		t.Fatal("expected token bucket to be exhausted after 3 of 3 tokens")
	}
	if res.Remaining != 0 {
		t.Fatalf("expected remaining 0 on denial, got %d", res.Remaining)
	}
	if res.Limit != 3 {
		t.Fatalf("expected limit 3, got %d", res.Limit)
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("expected positive retry-after on denial, got %v", res.RetryAfter)
	}
	if !res.ResetAt.After(time.Now()) {
		t.Fatal("expected reset-at in the future on denial")
	}
	if res.Scope != "credential" {
		t.Fatalf("expected default scope credential, got %q", res.Scope)
	}
	if res.Key != key {
		t.Fatalf("expected key %q, got %q", key, res.Key)
	}
}

func TestRateLimiter_IndependentBucketsPerKey(t *testing.T) {
	rl := NewRateLimiter()
	t.Cleanup(rl.Close)
	ctx := context.Background()

	first := rl.Check(ctx, "cred:a", 1, time.Hour)
	if !first.Allowed {
		t.Fatal("expected first key to have its own token")
	}
	second := rl.Check(ctx, "cred:b", 1, time.Hour)
	if !second.Allowed {
		t.Fatal("expected second key to have its own token")
	}

	if denied := rl.Check(ctx, "cred:a", 1, time.Hour); denied.Allowed {
		t.Fatal("expected first key exhausted while second key untouched")
	}
	if allowed := rl.Check(ctx, "cred:b", 1, time.Hour); allowed.Allowed {
		t.Fatal("expected second key to also be exhausted")
	}
}

func TestBucket_RefillsUpToLimitAfterWindow(t *testing.T) {
	b := &tokenBucket{
		key:        "cred:a",
		limit:      10,
		window:     time.Hour,
		tokens:     0.5,
		lastRefill: time.Now().Add(-time.Hour),
		refillRate: float64(10) / time.Hour.Seconds(),
	}

	res := b.take()

	if !res.Allowed {
		t.Fatal("expected bucket fully refilled after full window elapsed")
	}
	if res.Remaining != 9 {
		t.Fatalf("expected remaining 9 after consumption, got %d", res.Remaining)
	}
}

func TestBucket_MinimalRefillDoesNotAllow(t *testing.T) {
	b := &tokenBucket{
		key:        "cred:a",
		limit:      3,
		window:     time.Hour,
		tokens:     0.9,
		lastRefill: time.Now(),
		refillRate: float64(3) / time.Hour.Seconds(),
	}

	res := b.take()

	if res.Allowed {
		t.Fatal("expected sub-token bucket to deny")
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("expected positive retry-after, got %v", res.RetryAfter)
	}
}

func TestSlidingWindow_AllowsUpToLimit(t *testing.T) {
	sw := NewSlidingWindowRateLimiter()
	ctx := context.Background()
	key := "cred:anthropic:acct-1"
	now := time.Now()
	sw.clock = func() time.Time { return now }

	first := sw.Check(ctx, key, 2, time.Hour)
	if !first.Allowed {
		t.Fatal("expected first request allowed")
	}
	if first.Remaining != 2 {
		t.Fatalf("expected remaining 2 before any request recorded, got %d", first.Remaining)
	}

	second := sw.Check(ctx, key, 2, time.Hour)
	if !second.Allowed {
		t.Fatal("expected second request allowed")
	}
	if second.Remaining != 1 {
		t.Fatalf("expected remaining 1, got %d", second.Remaining)
	}

	third := sw.Check(ctx, key, 2, time.Hour)
	if third.Allowed {
		t.Fatal("expected third request denied at limit 2")
	}
	if third.Remaining != 0 {
		t.Fatalf("expected remaining 0 on denial, got %d", third.Remaining)
	}
	if third.RetryAfter != time.Hour {
		t.Fatalf("expected retry-after of full window (1h), got %v", third.RetryAfter)
	}
	if !third.ResetAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expected reset-at now+1h, got %v", third.ResetAt)
	}
}

func TestSlidingWindow_ExpiredRequestsPruned(t *testing.T) {
	sw := NewSlidingWindowRateLimiter()
	ctx := context.Background()
	key := "cred:anthropic:acct-1"
	now := time.Now()
	sw.clock = func() time.Time { return now }

	if res := sw.Check(ctx, key, 1, time.Hour); !res.Allowed {
		t.Fatal("expected first request allowed")
	}
	if res := sw.Check(ctx, key, 1, time.Hour); res.Allowed {
		t.Fatal("expected second request denied at limit 1")
	}

	now = now.Add(61 * time.Minute)

	res := sw.Check(ctx, key, 1, time.Hour)
	if !res.Allowed {
		t.Fatal("expected request allowed after the original request aged past the window")
	}
}

func TestSlidingWindow_RetryAfterShrinksAsRequestsAge(t *testing.T) {
	sw := NewSlidingWindowRateLimiter()
	ctx := context.Background()
	key := "cred:anthropic:acct-1"
	now := time.Now()
	sw.clock = func() time.Time { return now }

	sw.Check(ctx, key, 1, time.Hour)
	sw.Check(ctx, key, 1, time.Hour)
	denied := sw.Check(ctx, key, 1, time.Hour)
	if !denied.Allowed && denied.RetryAfter != time.Hour {
		t.Fatalf("expected retry-after 1h, got %v", denied.RetryAfter)
	}

	now = now.Add(30 * time.Minute)

	denied = sw.Check(ctx, key, 1, time.Hour)
	if denied.Allowed {
		t.Fatal("expected still denied after 30 minutes (request within window)")
	}
	if denied.RetryAfter != 30*time.Minute {
		t.Fatalf("expected retry-after 30m, got %v", denied.RetryAfter)
	}

	now = now.Add(31 * time.Minute)

	if res := sw.Check(ctx, key, 1, time.Hour); !res.Allowed {
		t.Fatal("expected allowed after the recorded request aged past the window")
	}
}
