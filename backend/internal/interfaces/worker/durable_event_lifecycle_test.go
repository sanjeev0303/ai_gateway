package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"ai-governance-saas/backend/internal/application/activity"
	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/eventbus"
	"ai-governance-saas/backend/internal/ports"
)

// -----------------------------------------------------------------------------
// Live NATS & JetStream Publish -> Live Container Worker -> PostgreSQL Persistence
// -----------------------------------------------------------------------------

func TestDurableEvent_LiveEndToEnd_PublishNATS_WorkerIngest_PostgresPersist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bus, err := eventbus.NewNATS(ctx, "nats://localhost:4222", discardLogger())
	if err != nil {
		t.Skipf("skipping live NATS test: connection to nats://localhost:4222 unavailable: %v", err)
	}
	defer bus.Close()

	if err := bus.Ping(ctx); err != nil {
		t.Fatalf("NATS ping failed: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://gateway:gateway@localhost:5432/ai_gateway?sslmode=disable"
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping live postgres check: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping live postgres check (ping failed): %v", err)
	}

	// Fetch a valid organization ID from database
	var orgID uuid.UUID
	err = pool.QueryRow(ctx, "SELECT id FROM organizations LIMIT 1").Scan(&orgID)
	if err != nil {
		t.Fatalf("failed to query valid organization from postgres: %v", err)
	}

	reqID := fmt.Sprintf("live-worker-e2e-%d", time.Now().UnixNano())
	msg := ports.UsageEventMessage{
		OrganizationID: orgID,
		Request: domain.NewLlmRequest{
			RequestID:        reqID,
			ProviderKind:     domain.ProviderKindOpenAI,
			Model:            "gpt-4o",
			Status:           domain.LlmRequestStatusSucceeded,
			PromptTokens:     125,
			CompletionTokens: 250,
			TotalTokens:      375,
			CostMicros:       6250,
			CostStatus:       domain.CostStatusCalculated,
			LatencyMs:        180,
			CreatedAt:        time.Now().UTC(),
		},
		Usage: domain.NewUsageEvent{
			RequestID: reqID,
			EventType: domain.UsageEventTypeTokenUsage,
			Quantity:  375,
			Unit:      "token",
		},
	}

	// 1. Publish to live NATS JetStream
	if err := bus.Publish(ctx, msg); err != nil {
		t.Fatalf("failed to publish usage event to live NATS: %v", err)
	}

	// 2. Poll PostgreSQL to verify the running background container worker consumed and persisted it
	var persistedPrompt, persistedComp, persistedTotal int32
	var persistedCost int64
	var found bool

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		row := pool.QueryRow(ctx,
			"SELECT prompt_tokens, completion_tokens, total_tokens, cost_micros FROM llm_requests WHERE request_id = $1",
			reqID)
		if scanErr := row.Scan(&persistedPrompt, &persistedComp, &persistedTotal, &persistedCost); scanErr == nil {
			found = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if !found {
		t.Fatalf("event %s was published to NATS but not ingested into PostgreSQL by the live worker within 6s", reqID)
	}

	if persistedPrompt != 125 || persistedComp != 250 || persistedTotal != 375 || persistedCost != 6250 {
		t.Fatalf("persisted values mismatch in PostgreSQL: prompt=%d, comp=%d, total=%d, cost=%d",
			persistedPrompt, persistedComp, persistedTotal, persistedCost)
	}

	// 3. Publish duplicate event and verify no double counting in PostgreSQL
	if err := bus.Publish(ctx, msg); err != nil {
		t.Fatalf("failed to publish duplicate usage event: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	var rowCount int
	err = pool.QueryRow(ctx, "SELECT count(*) FROM llm_requests WHERE request_id = $1", reqID).Scan(&rowCount)
	if err != nil {
		t.Fatalf("failed to count rows for request_id %s: %v", reqID, err)
	}
	if rowCount != 1 {
		t.Fatalf("expected exactly 1 row after duplicate NATS delivery, found %d", rowCount)
	}
}

// -----------------------------------------------------------------------------
// Idempotency Under Redelivery (No Double Counting)
// -----------------------------------------------------------------------------

func TestDurableEvent_IdempotentDeduplicationUnderRedelivery(t *testing.T) {
	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	orgID := uuid.New()
	reqID := "req-dedup-redelivery-1"
	req := domain.NewLlmRequest{
		RequestID:        reqID,
		Model:            "claude-3-5-sonnet",
		PromptTokens:     50,
		CompletionTokens: 100,
		TotalTokens:      150,
		CostMicros:       3000,
		Status:           domain.LlmRequestStatusSucceeded,
	}
	ev := domain.NewUsageEvent{
		RequestID: reqID,
		EventType: domain.UsageEventTypeTokenUsage,
		Quantity:  150,
	}

	// 1. Initial delivery
	if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
		t.Fatalf("initial delivery failed: %v", err)
	}

	// 2. Redelivery attempt 2 (e.g. consumer restarted before ACK)
	if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
		t.Fatalf("redelivery 2 should succeed idempotently, got: %v", err)
	}

	// 3. Redelivery attempt 3
	if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
		t.Fatalf("redelivery 3 should succeed idempotently, got: %v", err)
	}

	reqRepo.mu.Lock()
	defer reqRepo.mu.Unlock()
	if len(reqRepo.recorded) != 1 {
		t.Fatalf("expected exactly 1 recorded row in database, got %d", len(reqRepo.recorded))
	}
	if reqRepo.recorded[reqID].TotalTokens != 150 {
		t.Fatalf("expected 150 tokens, got %d", reqRepo.recorded[reqID].TotalTokens)
	}
}

// -----------------------------------------------------------------------------
// Non-Retryable Error Handling & Dead Letter Observability
// -----------------------------------------------------------------------------

func TestDurableEvent_NonRetryableRejectionAndDeadLetter(t *testing.T) {
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: newFlappyRequests(0),
		UsageEvents: &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)},
		Log:         discardLogger(),
	})

	tests := []struct {
		name    string
		orgID   uuid.UUID
		req     domain.NewLlmRequest
		ev      domain.NewUsageEvent
		wantErr error
	}{
		{
			name:    "missing organization ID",
			orgID:   uuid.Nil,
			req:     domain.NewLlmRequest{RequestID: "req-err-1"},
			ev:      domain.NewUsageEvent{RequestID: "req-err-1"},
			wantErr: ports.ErrNonRetryable,
		},
		{
			name:    "empty request ID",
			orgID:   uuid.New(),
			req:     domain.NewLlmRequest{RequestID: ""},
			ev:      domain.NewUsageEvent{RequestID: ""},
			wantErr: ports.ErrNonRetryable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.RecordRequest(context.Background(), tt.orgID, tt.req, tt.ev)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Worker Restarts & Resumption of Pending Queue
// -----------------------------------------------------------------------------

func TestDurableEvent_WorkerRestartResumesPendingWithoutLoss(t *testing.T) {
	bus := &restartableBus{}
	orgID := uuid.New()

	// Stage 5 messages onto durable queue while no worker is listening
	for i := 1; i <= 5; i++ {
		reqID := fmt.Sprintf("req-pending-restart-%d", i)
		_ = bus.Publish(context.Background(), ports.UsageEventMessage{
			OrganizationID: orgID,
			Request: domain.NewLlmRequest{
				RequestID:   reqID,
				Model:       "gpt-4o",
				TotalTokens: int32(50 * i),
				CostMicros:  int64(100 * i),
			},
			Usage: domain.NewUsageEvent{
				RequestID: reqID,
				EventType: domain.UsageEventTypeTokenUsage,
				Quantity:  int64(50 * i),
			},
		})
	}

	reqRepo := newFlappyRequests(0)
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)},
		Log:         discardLogger(),
	})

	// Worker 1 starts and immediately crashes / is killed
	crashCtx, crashCancel := context.WithCancel(context.Background())
	crashCancel() // killed immediately
	_ = bus.Run(crashCtx, func(ctx context.Context, msg ports.UsageEventMessage) error {
		return svc.RecordRequest(ctx, msg.OrganizationID, msg.Request, msg.Usage)
	})

	// Verify all 5 messages remain in queue
	bus.mu.Lock()
	if len(bus.queue) != 5 {
		t.Fatalf("expected 5 messages remaining in queue after worker crash, got %d", len(bus.queue))
	}
	bus.mu.Unlock()

	// Worker 2 starts up, recovers pending messages
	runCtx, runCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer runCancel()

	go func() {
		_ = bus.Run(runCtx, func(ctx context.Context, msg ports.UsageEventMessage) error {
			return svc.RecordRequest(ctx, msg.OrganizationID, msg.Request, msg.Usage)
		})
	}()

	time.Sleep(50 * time.Millisecond)
	runCancel()

	reqRepo.mu.Lock()
	count := len(reqRepo.recorded)
	reqRepo.mu.Unlock()

	if count != 5 {
		t.Fatalf("expected all 5 messages to be persisted after worker restart, got %d", count)
	}
	bus.mu.Lock()
	remaining := len(bus.queue)
	bus.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("expected queue to be completely drained after worker restart, got %d", remaining)
	}
}

// -----------------------------------------------------------------------------
// Multi-Tenant Attribution Preservation
// -----------------------------------------------------------------------------

func TestDurableEvent_AttributionPreservation(t *testing.T) {
	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	orgID := uuid.New()
	userID := uuid.New()
	projID := uuid.New()
	teamID := uuid.New()
	credID := uuid.New()
	provReqID := "provider-trace-12345"
	reqID := "req-attribution-check-1"

	req := domain.NewLlmRequest{
		RequestID:           reqID,
		UserID:              &userID,
		ProjectID:           &projID,
		TeamID:              &teamID,
		GatewayCredentialID: &credID,
		ProviderRequestID:   &provReqID,
		ProviderKind:        domain.ProviderKindAnthropic,
		Model:               "claude-3-5-sonnet",
		Status:              domain.LlmRequestStatusSucceeded,
		PromptTokens:        120,
		CompletionTokens:    80,
		CachedInputTokens:   20,
		TotalTokens:         200,
		CostMicros:          4500,
		CostStatus:          domain.CostStatusCalculated,
		LatencyMs:           245,
	}

	ev := domain.NewUsageEvent{
		RequestID: reqID,
		EventType: domain.UsageEventTypeTokenUsage,
		Quantity:  200,
		Unit:      "token",
		Metadata:  []byte(`{"provider":"anthropic","cache_read_tokens":20}`),
	}

	if err := svc.RecordRequest(context.Background(), orgID, req, ev); err != nil {
		t.Fatalf("record request failed: %v", err)
	}

	reqRepo.mu.Lock()
	saved := reqRepo.recorded[reqID]
	reqRepo.mu.Unlock()

	if saved.RequestID != reqID {
		t.Errorf("RequestID mismatch: %s", saved.RequestID)
	}
	if saved.UserID == nil || *saved.UserID != userID {
		t.Errorf("UserID attribution lost: %v", saved.UserID)
	}
	if saved.ProjectID == nil || *saved.ProjectID != projID {
		t.Errorf("ProjectID attribution lost: %v", saved.ProjectID)
	}
	if saved.TeamID == nil || *saved.TeamID != teamID {
		t.Errorf("TeamID attribution lost: %v", saved.TeamID)
	}
	if saved.GatewayCredentialID == nil || *saved.GatewayCredentialID != credID {
		t.Errorf("GatewayCredentialID attribution lost: %v", saved.GatewayCredentialID)
	}
	if saved.ProviderKind != domain.ProviderKindAnthropic {
		t.Errorf("ProviderKind mismatch: %s", saved.ProviderKind)
	}
	if saved.Model != "claude-3-5-sonnet" {
		t.Errorf("Model mismatch: %s", saved.Model)
	}
	if saved.PromptTokens != 120 || saved.CompletionTokens != 80 || saved.CachedInputTokens != 20 || saved.TotalTokens != 200 {
		t.Errorf("Token counters mismatch: %+v", saved)
	}
	if saved.CostMicros != 4500 || saved.CostStatus != domain.CostStatusCalculated {
		t.Errorf("Cost mismatch: %d (%s)", saved.CostMicros, saved.CostStatus)
	}
}

// -----------------------------------------------------------------------------
// Request-Level to Aggregate Reconciliation
// -----------------------------------------------------------------------------

func TestDurableEvent_RequestReconciliationWithRollupAggregates(t *testing.T) {
	type reqItem struct {
		model            string
		promptTokens     int32
		completionTokens int32
		cachedTokens     int32
		costMicros       int64
		userID           uuid.UUID
		projID           uuid.UUID
	}

	u1 := uuid.New()
	u2 := uuid.New()
	p1 := uuid.New()
	p2 := uuid.New()

	requests := []reqItem{
		{"gpt-4o", 100, 50, 10, 2500, u1, p1},
		{"gpt-4o", 200, 100, 20, 5000, u1, p1},
		{"claude-3-5-sonnet", 150, 75, 0, 3500, u2, p2},
		{"gemini-1.5-pro", 300, 150, 50, 4200, u2, p2},
	}

	var sumPrompt, sumComp, sumCached, sumTotal, sumCost int64
	projRollup := make(map[uuid.UUID]int64)
	userRollup := make(map[uuid.UUID]int64)

	for _, r := range requests {
		tot := int64(r.promptTokens + r.completionTokens)
		sumPrompt += int64(r.promptTokens)
		sumComp += int64(r.completionTokens)
		sumCached += int64(r.cachedTokens)
		sumTotal += tot
		sumCost += r.costMicros

		projRollup[r.projID] += r.costMicros
		userRollup[r.userID] += tot
	}

	// Mathematical reconciliation
	if sumTotal != sumPrompt+sumComp {
		t.Fatalf("total token mismatch: %d != %d + %d", sumTotal, sumPrompt, sumComp)
	}
	if sumCost != 15200 {
		t.Fatalf("expected 15200 cost micros, got %d", sumCost)
	}

	var sumFromProjects int64
	for _, cost := range projRollup {
		sumFromProjects += cost
	}
	if sumFromProjects != sumCost {
		t.Fatalf("project breakdown sum %d != total cost %d", sumFromProjects, sumCost)
	}

	var sumFromUsers int64
	for _, tokens := range userRollup {
		sumFromUsers += tokens
	}
	if sumFromUsers != sumTotal {
		t.Fatalf("user breakdown sum %d != total tokens %d", sumFromUsers, sumTotal)
	}
}

// -----------------------------------------------------------------------------
// Cost Distinction: Exact vs Estimated vs Error
// -----------------------------------------------------------------------------

func TestDurableEvent_CostCalculationExactVsEstimatedVsError(t *testing.T) {
	tests := []struct {
		name               string
		statusCode         int
		tokensReported     bool
		wantCostStatus     string
		wantZeroCostMicros bool
	}{
		{
			name:               "provider reports exact usage metadata",
			statusCode:         200,
			tokensReported:     true,
			wantCostStatus:     domain.CostStatusCalculated,
			wantZeroCostMicros: false,
		},
		{
			name:               "provider omits usage metadata fallback to estimates",
			statusCode:         200,
			tokensReported:     false,
			wantCostStatus:     domain.CostStatusEstimated,
			wantZeroCostMicros: false,
		},
		{
			name:               "provider returns 429 rate limit error",
			statusCode:         429,
			tokensReported:     false,
			wantCostStatus:     domain.CostStatusUnavailable,
			wantZeroCostMicros: true,
		},
		{
			name:               "provider returns 500 internal server error",
			statusCode:         500,
			tokensReported:     false,
			wantCostStatus:     domain.CostStatusUnavailable,
			wantZeroCostMicros: true,
		},
		{
			name:               "client disconnects mid-stream (499)",
			statusCode:         499,
			tokensReported:     false,
			wantCostStatus:     domain.CostStatusUnavailable,
			wantZeroCostMicros: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var costMicros int64
			var costStatus string

			if tt.statusCode >= 400 || tt.statusCode == 499 {
				costStatus = domain.CostStatusUnavailable
				costMicros = 0
			} else if tt.tokensReported {
				costStatus = domain.CostStatusCalculated
				costMicros = 5000
			} else {
				costStatus = domain.CostStatusEstimated
				costMicros = 5000
			}

			if costStatus != tt.wantCostStatus {
				t.Fatalf("expected CostStatus %s, got %s", tt.wantCostStatus, costStatus)
			}
			if tt.wantZeroCostMicros && costMicros != 0 {
				t.Fatalf("expected CostMicros 0, got %d", costMicros)
			}
			if !tt.wantZeroCostMicros && costMicros <= 0 {
				t.Fatalf("expected positive CostMicros, got %d", costMicros)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Redis Multi-Tenant Isolation & Stale Pruning
// -----------------------------------------------------------------------------

func TestDurableEvent_RedisTenantIsolationAndStalePruning(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	password := os.Getenv("AIGW_REDIS_PASSWORD")
	if password == "" {
		password = os.Getenv("REDIS_PASSWORD")
	}
	if password == "" {
		password = "change-me"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: password,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping Redis live test: connection to localhost:6379 failed: %v", err)
	}

	actSvc := activity.NewService(redisClient, discardLogger())

	orgA := uuid.New()
	orgB := uuid.New()

	reqA1 := "req-a-active-1"
	reqA2_stale := "req-a-stale-2"
	reqB1 := "req-b-active-1"

	keyA := fmt.Sprintf("gateway:activity:active:%s", orgA.String())
	keyB := fmt.Sprintf("gateway:activity:active:%s", orgB.String())

	now := time.Now()
	freshTime := now.Add(-30 * time.Second)
	staleTime := now.Add(-15 * time.Minute) // Older than 10-minute pruning threshold

	evA1, _ := json.Marshal(domain.ActivityEvent{
		RequestID:      reqA1,
		OrganizationID: orgA,
		Model:          "gpt-4o",
		CurrentState:   domain.ActivityStateActive,
		StartedAt:      freshTime,
	})
	evA2Stale, _ := json.Marshal(domain.ActivityEvent{
		RequestID:      reqA2_stale,
		OrganizationID: orgA,
		Model:          "claude-3-5-sonnet",
		CurrentState:   domain.ActivityStateActive,
		StartedAt:      staleTime,
	})
	evB1, _ := json.Marshal(domain.ActivityEvent{
		RequestID:      reqB1,
		OrganizationID: orgB,
		Model:          "gemini-1.5-pro",
		CurrentState:   domain.ActivityStateActive,
		StartedAt:      freshTime,
	})

	// Seed Redis hashes
	_ = redisClient.HSet(ctx, keyA, reqA1, evA1).Err()
	_ = redisClient.HSet(ctx, keyA, reqA2_stale, evA2Stale).Err()
	_ = redisClient.HSet(ctx, keyB, reqB1, evB1).Err()
	defer func() {
		_ = redisClient.Del(context.Background(), keyA, keyB).Err()
	}()

	// Query ActiveRequests for Org A
	activeA, err := actSvc.ActiveRequests(ctx, orgA)
	if err != nil {
		t.Fatalf("ActiveRequests for Org A failed: %v", err)
	}

	// Verify Tenant Isolation: Org A results must NEVER contain Org B requests
	for _, ev := range activeA {
		if ev.OrganizationID != orgA {
			t.Fatalf("cross-tenant leak: received event for %s while querying %s", ev.OrganizationID, orgA)
		}
		if ev.RequestID == reqB1 {
			t.Fatalf("cross-tenant leak: Org B request %s present in Org A query results", reqB1)
		}
	}

	// Verify Stale Pruning: Stale request (> 10 min) must be pruned automatically
	foundA1 := false
	for _, ev := range activeA {
		if ev.RequestID == reqA1 {
			foundA1 = true
		}
		if ev.RequestID == reqA2_stale {
			t.Fatalf("stale request %s was not pruned by activity service", reqA2_stale)
		}
	}
	if !foundA1 {
		t.Fatalf("expected fresh request %s to be returned for Org A", reqA1)
	}

	// Query ActiveRequests for Org B
	activeB, err := actSvc.ActiveRequests(ctx, orgB)
	if err != nil {
		t.Fatalf("ActiveRequests for Org B failed: %v", err)
	}
	if len(activeB) != 1 || activeB[0].RequestID != reqB1 {
		t.Fatalf("expected 1 active request %s for Org B, got %+v", reqB1, activeB)
	}
}
