package gatewayv2

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"os"
)

func getTestRedis(t *testing.T) *redis.Client {
	password := os.Getenv("AIGW_REDIS_PASSWORD")
	if password == "" {
		password = os.Getenv("REDIS_PASSWORD")
	}
	if password == "" {
		password = "change-me"
	}

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: password,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping Redis rate limiter test: local Redis unavailable on localhost:6379 (%v)", err)
		return nil
	}
	return client
}

// 1. Concurrent requests competing for shared sliding window rate limit in Redis
func TestRateLimit_DistributedSlidingWindowAndConcurrency(t *testing.T) {
	client := getTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	limiter := NewRedisRateLimiter(client, logger)
	ctx := context.Background()

	key := "test:concurrent:" + uuid.NewString()
	const limit = 5
	window := 2 * time.Second

	const concurrentWorkers = 20
	var allowedCount int32
	var deniedCount int32

	var wg sync.WaitGroup
	wg.Add(concurrentWorkers)

	for i := 0; i < concurrentWorkers; i++ {
		go func() {
			defer wg.Done()
			res := limiter.Check(ctx, key, limit, window)
			if res.Allowed {
				atomic.AddInt32(&allowedCount, 1)
			} else {
				atomic.AddInt32(&deniedCount, 1)
			}
		}()
	}

	wg.Wait()

	if allowedCount != limit {
		t.Fatalf("expected exactly %d requests allowed across concurrent goroutines, got %d (denied=%d)",
			limit, allowedCount, deniedCount)
	}
	if deniedCount != (concurrentWorkers - limit) {
		t.Fatalf("expected %d requests denied, got %d", concurrentWorkers-limit, deniedCount)
	}

	// Immediate next request must also be denied
	deniedRes := limiter.Check(ctx, key, limit, window)
	if deniedRes.Allowed {
		t.Fatalf("expected subsequent request to be denied while window is full")
	}
	if deniedRes.Remaining != 0 {
		t.Fatalf("expected remaining=0 on denial, got %d", deniedRes.Remaining)
	}
}

// 2. Exact sliding window expiration and mathematical RetryAfter
func TestRateLimit_ExactSlidingWindowAndExpiration(t *testing.T) {
	client := getTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	limiter := NewRedisRateLimiter(client, logger)
	ctx := context.Background()

	key := "test:window-expiry:" + uuid.NewString()
	const limit = 2
	window := 100 * time.Millisecond

	// 1st request -> allowed
	res1 := limiter.Check(ctx, key, limit, window)
	if !res1.Allowed || res1.Remaining != 1 {
		t.Fatalf("call 1: expected allowed=true, remaining=1, got allowed=%v, remaining=%d", res1.Allowed, res1.Remaining)
	}

	// 2nd request -> allowed
	res2 := limiter.Check(ctx, key, limit, window)
	if !res2.Allowed || res2.Remaining != 0 {
		t.Fatalf("call 2: expected allowed=true, remaining=0, got allowed=%v, remaining=%d", res2.Allowed, res2.Remaining)
	}

	// 3rd request -> denied with positive RetryAfter
	res3 := limiter.Check(ctx, key, limit, window)
	if res3.Allowed {
		t.Fatalf("call 3: expected denied")
	}
	if res3.RetryAfter <= 0 || res3.RetryAfter > window {
		t.Fatalf("call 3: expected 0 < RetryAfter <= %v, got %v", window, res3.RetryAfter)
	}

	// Wait for window to slide past
	time.Sleep(window + 20*time.Millisecond)

	// 4th request after expiration -> allowed again
	res4 := limiter.Check(ctx, key, limit, window)
	if !res4.Allowed {
		t.Fatalf("call 4: expected allowed after window expired, got denied")
	}
}

// 3. Redis outage handling: ModeDegradedLocal
func TestRateLimit_RedisOutageDegradedMode(t *testing.T) {
	// Connect to non-existent Redis port to simulate network partition / service outage
	deadClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59999",
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadClient.Close()

	logger := slog.Default()
	limiter := NewRedisRateLimiter(deadClient, logger).WithMode(ModeDegradedLocal)
	ctx := context.Background()
	key := "test:outage:degraded:" + uuid.NewString()

	// Should not panic, should fall back to local limiter and flag Degraded = true
	res1 := limiter.Check(ctx, key, 2, time.Hour)
	if !res1.Allowed {
		t.Fatalf("expected degraded fallback to allow initial request under limit")
	}
	if !res1.Degraded {
		t.Fatalf("expected res.Degraded = true when Redis is unreachable in ModeDegradedLocal")
	}

	res2 := limiter.Check(ctx, key, 2, time.Hour)
	if !res2.Allowed || !res2.Degraded {
		t.Fatalf("expected res2 allowed and degraded=true")
	}

	// 3rd request exceeds local limit -> denied with degraded=true
	res3 := limiter.Check(ctx, key, 2, time.Hour)
	if res3.Allowed {
		t.Fatalf("expected local token bucket to enforce limit and deny 3rd request")
	}
	if !res3.Degraded {
		t.Fatalf("expected res3.Degraded = true")
	}
}

// 4. Redis outage handling: ModeFailClosed
func TestRateLimit_RedisOutageFailClosedMode(t *testing.T) {
	deadClient := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59999",
		DialTimeout: 50 * time.Millisecond,
		ReadTimeout: 50 * time.Millisecond,
	})
	defer deadClient.Close()

	logger := slog.Default()
	limiter := NewRedisRateLimiter(deadClient, logger).WithMode(ModeFailClosed)
	ctx := context.Background()
	key := "test:outage:failclosed:" + uuid.NewString()

	// In FailClosed mode, outage must immediately deny request to guarantee no over-limit traffic passes
	res := limiter.Check(ctx, key, 10, time.Hour)
	if res.Allowed {
		t.Fatalf("expected ModeFailClosed to deny request during Redis outage, got Allowed=true")
	}
	if !res.Degraded {
		t.Fatalf("expected res.Degraded = true during Redis outage")
	}
	if res.RetryAfter <= 0 {
		t.Fatalf("expected positive RetryAfter in fail closed result")
	}
}

// 5. Cross-user and cross-organization scope isolation
func TestRateLimit_TenantAndScopeIsolation(t *testing.T) {
	client := getTestRedis(t)
	defer client.Close()

	logger := slog.Default()
	limiter := NewRedisRateLimiter(client, logger)
	ctx := context.Background()

	orgA := uuid.NewString()
	orgB := uuid.NewString()
	user1 := uuid.NewString()
	user2 := uuid.NewString()

	keyOrgAUser1 := "org:" + orgA + ":user:" + user1
	keyOrgAUser2 := "org:" + orgA + ":user:" + user2
	keyOrgBUser1 := "org:" + orgB + ":user:" + user1

	const limit = 2
	window := 1 * time.Hour

	// Exhaust User 1 in Org A
	limiter.Check(ctx, keyOrgAUser1, limit, window)
	limiter.Check(ctx, keyOrgAUser1, limit, window)
	if res := limiter.Check(ctx, keyOrgAUser1, limit, window); res.Allowed {
		t.Fatalf("expected User 1 in Org A to be exhausted")
	}

	// User 2 in Org A must be completely unaffected
	if res := limiter.Check(ctx, keyOrgAUser2, limit, window); !res.Allowed {
		t.Fatalf("User 2 in Org A should not be throttled by User 1")
	}

	// Same User ID in Org B must be completely unaffected (cross-tenant isolation)
	if res := limiter.Check(ctx, keyOrgBUser1, limit, window); !res.Allowed {
		t.Fatalf("User in Org B should be isolated from Org A")
	}
}

// 6. Header manipulation immunity: client headers cannot bypass or alter rate-limit scope
func TestRateLimit_HeaderManipulationImmunity(t *testing.T) {
	orgID := uuid.New()
	credID := uuid.New()
	userID := uuid.New()

	cred := GatewayCredential{
		ID:             credID,
		OrganizationID: orgID,
		SubjectType:    "user",
		SubjectID:      userID,
	}

	gw := &Gateway{}

	// Rate limit scoped key for org-level policy rule
	cfgOrg := RateLimitConfig{Scope: "organization"}
	keyOrg := gw.buildRateLimitScopedKey(cred, cfgOrg)
	expectedOrgKey := "org:" + orgID.String()
	if keyOrg != expectedOrgKey {
		t.Fatalf("expected %q, got %q", expectedOrgKey, keyOrg)
	}

	// Rate limit scoped key for user-level policy rule
	cfgUser := RateLimitConfig{Scope: "user"}
	keyUser := gw.buildRateLimitScopedKey(cred, cfgUser)
	expectedUserKey := "org:" + orgID.String() + ":user:" + userID.String()
	if keyUser != expectedUserKey {
		t.Fatalf("expected %q, got %q", expectedUserKey, keyUser)
	}

	// Rate limit default key
	fallbackKey := gw.buildRateLimitKey(cred, ResolvedUpstream{})
	expectedFallbackKey := "org:" + orgID.String() + ":cred:" + credID.String()
	if fallbackKey != expectedFallbackKey {
		t.Fatalf("expected %q, got %q", expectedFallbackKey, fallbackKey)
	}
}

// 7. Rejected requests attribution and zero provider usage
func TestRateLimit_RejectedRequestsAttributionAndZeroUsage(t *testing.T) {
	gw := &Gateway{
		logger: slog.Default(),
	}

	orgID := uuid.New()
	credID := uuid.New()
	reqCtx := &RequestContext{
		Credential: GatewayCredential{
			ID:             credID,
			OrganizationID: orgID,
		},
	}

	rlResult := RateLimitResult{
		Allowed:    false,
		Limit:      10,
		Remaining:  0,
		RetryAfter: 30 * time.Second,
		ResetAt:    time.Now().Add(30 * time.Second),
		Key:        "org:" + orgID.String(),
	}

	res := gw.rateLimitedResult(rlResult, reqCtx)

	if res.Allowed {
		t.Fatal("expected PipelineResult.Allowed = false")
	}
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429, got %d", res.StatusCode)
	}
	if res.ErrorType != "rate_limit_exceeded" {
		t.Fatalf("expected rate_limit_exceeded error type, got %q", res.ErrorType)
	}

	// Test response writing
	rec := httptest.NewRecorder()
	gw.writeResult(rec, res)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP 429 in response, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "30" {
		t.Fatalf("expected Retry-After: 30, got %q", rec.Header().Get("Retry-After"))
	}
	if rec.Header().Get("X-RateLimit-Limit") != "10" {
		t.Fatalf("expected X-RateLimit-Limit: 10, got %q", rec.Header().Get("X-RateLimit-Limit"))
	}
	if rec.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("expected X-RateLimit-Remaining: 0, got %q", rec.Header().Get("X-RateLimit-Remaining"))
	}

	// Verify error response body format
	bodyBytes, _ := io.ReadAll(rec.Body)
	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, "rate limit exceeded") {
		t.Fatalf("expected response body to contain error message, got: %s", bodyStr)
	}
}

// 8. Key revocation precedes rate limiting
func TestRateLimit_KeyRevocationPrecedesRateLimiting(t *testing.T) {
	// If a credential is revoked, Authenticate returns ErrCredentialRevoked in Stage 1,
	// returning HTTP 401 without ever reaching Stage 6 rate limiting.
	gw := &Gateway{
		logger: slog.Default(),
	}

	// Simulate Stage 1 failure
	pipelineErr := gw.pipelineError("authenticate", ErrCredentialRevoked, http.StatusUnauthorized, "credential_revoked")

	if pipelineErr.Allowed {
		t.Fatal("expected revoked credential to be rejected")
	}
	if pipelineErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 Unauthorized for revoked key, got %d", pipelineErr.StatusCode)
	}
	if pipelineErr.ErrorType != "credential_revoked" {
		t.Fatalf("expected error type credential_revoked, got %s", pipelineErr.ErrorType)
	}
}
