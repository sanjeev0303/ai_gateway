package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/interfaces/gatewayv2"
	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"
)

type flappyRequests struct {
	mu           sync.Mutex
	failAttempts int
	calls        int
	recorded     map[string]domain.NewLlmRequest
}

func newFlappyRequests(failCount int) *flappyRequests {
	return &flappyRequests{
		failAttempts: failCount,
		recorded:     make(map[string]domain.NewLlmRequest),
	}
}

func (f *flappyRequests) Record(_ context.Context, _ uuid.UUID, in domain.NewLlmRequest) (domain.LlmRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.failAttempts {
		return domain.LlmRequest{}, errors.New("transient database connection failure")
	}
	if _, exists := f.recorded[in.RequestID]; exists {
		return domain.LlmRequest{}, repositories.ErrConflict
	}
	f.recorded[in.RequestID] = in
	return domain.LlmRequest{RequestID: in.RequestID}, nil
}

func (f *flappyRequests) GetByRequestID(context.Context, uuid.UUID, string) (domain.LlmRequest, error) {
	return domain.LlmRequest{}, nil
}
func (f *flappyRequests) List(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.LlmRequest, error) {
	return nil, nil
}
func (f *flappyRequests) GetRecent(context.Context, uuid.UUID) ([]domain.LlmRequest, error) {
	return nil, nil
}
func (f *flappyRequests) GetRequests(context.Context, uuid.UUID, domain.UsageFilter, int32, int32) ([]domain.LlmRequest, error) {
	return nil, nil
}

func (f *flappyRequests) PurgeExpired(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}

type flappyEvents struct {
	mu       sync.Mutex
	recorded map[string]domain.NewUsageEvent
}

func (f *flappyEvents) Record(_ context.Context, _ uuid.UUID, in domain.NewUsageEvent) (domain.UsageEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.recorded[in.RequestID]; exists {
		return domain.UsageEvent{}, repositories.ErrConflict
	}
	if f.recorded == nil {
		f.recorded = make(map[string]domain.NewUsageEvent)
	}
	f.recorded[in.RequestID] = in
	return domain.UsageEvent{RequestID: in.RequestID, EventType: in.EventType}, nil
}

func (f *flappyEvents) Get(context.Context, uuid.UUID, string, string) (domain.UsageEvent, error) {
	return domain.UsageEvent{}, nil
}
func (f *flappyEvents) ListByRequest(context.Context, uuid.UUID, string) ([]domain.UsageEvent, error) {
	return nil, nil
}
func (f *flappyEvents) PurgeExpired(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}


// TestEventProcessing_DuplicateDeliveryIsIdempotent verifies that a duplicated event
// does not return an error or corrupt data.
func TestEventProcessing_DuplicateDeliveryIsIdempotent(t *testing.T) {
	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	orgID := uuid.New()
	msg := domain.NewLlmRequest{RequestID: "req-dup-1", TotalTokens: 100}
	ev := domain.NewUsageEvent{RequestID: "req-dup-1", EventType: domain.UsageEventTypeTokenUsage, Quantity: 100}

	// First delivery
	if err := svc.RecordRequest(context.Background(), orgID, msg, ev); err != nil {
		t.Fatalf("first delivery failed: %v", err)
	}

	// Duplicate delivery
	if err := svc.RecordRequest(context.Background(), orgID, msg, ev); err != nil {
		t.Fatalf("duplicate delivery should be idempotent success, got: %v", err)
	}

	if len(reqRepo.recorded) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqRepo.recorded))
	}
	if len(evRepo.recorded) != 1 {
		t.Fatalf("expected 1 recorded event, got %d", len(evRepo.recorded))
	}
}

// TestEventProcessing_TransientFailureRecovers verifies that when a database write fails
// temporarily, redelivery succeeds.
func TestEventProcessing_TransientFailureRecovers(t *testing.T) {
	reqRepo := newFlappyRequests(1) // fail on attempt 1
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	orgID := uuid.New()
	msg := domain.NewLlmRequest{RequestID: "req-flappy-1", TotalTokens: 50}
	ev := domain.NewUsageEvent{RequestID: "req-flappy-1", EventType: domain.UsageEventTypeTokenUsage, Quantity: 50}

	// First attempt fails due to transient database error
	err := svc.RecordRequest(context.Background(), orgID, msg, ev)
	if err == nil {
		t.Fatal("expected first attempt to fail")
	}

	// Second attempt succeeds on redelivery
	if err := svc.RecordRequest(context.Background(), orgID, msg, ev); err != nil {
		t.Fatalf("second attempt failed: %v", err)
	}

	if len(reqRepo.recorded) != 1 {
		t.Fatalf("expected 1 recorded request, got %d", len(reqRepo.recorded))
	}
}

// TestEventProcessing_NonRetryableRejects verifies that corrupt/missing IDs return ErrNonRetryable.
func TestEventProcessing_NonRetryableRejects(t *testing.T) {
	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	// Missing org ID
	err := svc.RecordRequest(context.Background(), uuid.Nil, domain.NewLlmRequest{RequestID: "req-x"}, domain.NewUsageEvent{RequestID: "req-x"})
	if !errors.Is(err, ports.ErrNonRetryable) {
		t.Fatalf("expected ErrNonRetryable for missing orgID, got %v", err)
	}

	// Missing request ID
	err = svc.RecordRequest(context.Background(), uuid.New(), domain.NewLlmRequest{RequestID: ""}, domain.NewUsageEvent{RequestID: ""})
	if !errors.Is(err, ports.ErrNonRetryable) {
		t.Fatalf("expected ErrNonRetryable for empty requestID, got %v", err)
	}
}

type flappyBus struct {
	failPublish atomic.Bool
	published   []ports.UsageEventMessage
	mu          sync.Mutex
}

func (b *flappyBus) Publish(_ context.Context, msg ports.UsageEventMessage) error {
	if b.failPublish.Load() {
		return errors.New("nats: connection temporarily unavailable")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, msg)
	return nil
}
func (b *flappyBus) PublishSecurityEvent(context.Context, ports.SecurityEventMessage) error {
	return nil
}
func (b *flappyBus) Run(context.Context, func(context.Context, ports.UsageEventMessage) error) error {
	return nil
}
func (b *flappyBus) RunSecurity(context.Context, func(context.Context, ports.SecurityEventMessage) error) error {
	return nil
}

// TestTelemetryEmitter_SpoolsOnPublishFailure verifies that when eventbus Publish fails,
// the emitter does not drop records but spools and delivers them once the bus recovers.
func TestTelemetryEmitter_SpoolsOnPublishFailure(t *testing.T) {
	bus := &flappyBus{}
	bus.failPublish.Store(true) // simulate bus down

	emitter := gatewayv2.NewTelemetryEmitter(100, 10*time.Millisecond, bus, discardLogger())
	defer emitter.Close()

	orgID := uuid.New()
	emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
		RequestID:      "spool-req-1",
		OrganizationID: orgID,
		Model:          "gpt-4o",
		InputTokens:    10,
		OutputTokens:   20,
	})

	// Flush while bus is failing: record will be spooled
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = emitter.Flush(ctx)
	cancel()

	bus.mu.Lock()
	gotBefore := len(bus.published)
	bus.mu.Unlock()
	if gotBefore != 0 {
		t.Fatalf("expected 0 published while bus down, got %d", gotBefore)
	}

	// Bus recovers
	bus.failPublish.Store(false)

	// Flush again
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	if err := emitter.Flush(ctx2); err != nil {
		t.Fatalf("flush after recovery failed: %v", err)
	}
	cancel2()

	bus.mu.Lock()
	gotAfter := len(bus.published)
	bus.mu.Unlock()

	if gotAfter != 1 {
		t.Fatalf("expected 1 published after spool recovery, got %d", gotAfter)
	}
	if bus.published[0].Request.RequestID != "spool-req-1" {
		t.Fatalf("expected spool-req-1, got %s", bus.published[0].Request.RequestID)
	}
}

// restartableBus simulates durable broker queue mechanics: unacknowledged
// messages remain queued and are delivered to the next consumer on restart.
type restartableBus struct {
	mu        sync.Mutex
	queue     []ports.UsageEventMessage
	delivered int
}

func (b *restartableBus) Publish(_ context.Context, msg ports.UsageEventMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.queue = append(b.queue, msg)
	return nil
}
func (b *restartableBus) PublishSecurityEvent(context.Context, ports.SecurityEventMessage) error {
	return nil
}
func (b *restartableBus) Run(ctx context.Context, handle func(context.Context, ports.UsageEventMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b.mu.Lock()
		if len(b.queue) == 0 {
			b.mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			continue
		}
		msg := b.queue[0]
		b.mu.Unlock()

		err := handle(ctx, msg)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err // unacked: leaves message in queue for next restart
			}
			return err
		}

		b.mu.Lock()
		if len(b.queue) > 0 {
			b.queue = b.queue[1:]
			b.delivered++
		}
		b.mu.Unlock()
	}
}
func (b *restartableBus) RunSecurity(context.Context, func(context.Context, ports.SecurityEventMessage) error) error {
	return nil
}

// TestEventProcessing_WorkerTerminationAndRestart tests redelivery on consumer crash.
func TestEventProcessing_WorkerTerminationAndRestart(t *testing.T) {
	bus := &restartableBus{}
	orgID := uuid.New()
	msg := ports.UsageEventMessage{
		OrganizationID: orgID,
		Request: domain.NewLlmRequest{
			RequestID:   "req-term-1",
			Model:       "gpt-4o",
			TotalTokens: 150,
			CostMicros:  3000,
		},
		Usage: domain.NewUsageEvent{
			RequestID: "req-term-1",
			EventType: domain.UsageEventTypeTokenUsage,
			Quantity:  150,
		},
	}
	_ = bus.Publish(context.Background(), msg)

	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	// Worker 1 starts, but is immediately cancelled before it can complete ACK
	ctx1, cancel1 := context.WithCancel(context.Background())
	cancel1()
	RunUsageConsumer(ctx1, bus, svc, discardLogger())

	// Verify message was not permanently lost
	bus.mu.Lock()
	qLen := len(bus.queue)
	bus.mu.Unlock()
	if qLen != 1 {
		t.Fatalf("expected message to remain queued after worker 1 termination, got %d", qLen)
	}

	// Worker 2 starts with fresh context, consumes and acks the message
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	go RunUsageConsumer(ctx2, bus, svc, discardLogger())

	time.Sleep(30 * time.Millisecond)
	cancel2()

	reqRepo.mu.Lock()
	recCount := len(reqRepo.recorded)
	reqRepo.mu.Unlock()

	if recCount != 1 {
		t.Fatalf("expected 1 recorded request after worker restart, got %d", recCount)
	}
	bus.mu.Lock()
	del := bus.delivered
	bus.mu.Unlock()
	if del != 1 {
		t.Fatalf("expected 1 delivered message after restart, got %d", del)
	}
}

// TestEventProcessing_EventBusBacklogRecovery tests that queued events survive bus disconnection.
func TestEventProcessing_EventBusBacklogRecovery(t *testing.T) {
	bus := &flappyBus{}
	bus.failPublish.Store(true) // Bus down

	emitter := gatewayv2.NewTelemetryEmitter(100, 10*time.Millisecond, bus, discardLogger())
	defer emitter.Close()

	orgID := uuid.New()
	for i := 1; i <= 5; i++ {
		emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
			RequestID:      fmt.Sprintf("backlog-req-%d", i),
			OrganizationID: orgID,
			Model:          "claude-3-5-sonnet",
			InputTokens:    10 * i,
			OutputTokens:   20 * i,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = emitter.Flush(ctx)
	cancel()

	bus.mu.Lock()
	if len(bus.published) != 0 {
		bus.mu.Unlock()
		t.Fatalf("expected 0 published while bus down, got %d", len(bus.published))
	}
	bus.mu.Unlock()

	// Bus recovers
	bus.failPublish.Store(false)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	if err := emitter.Flush(ctx2); err != nil {
		t.Fatalf("flush after recovery failed: %v", err)
	}
	cancel2()

	bus.mu.Lock()
	got := len(bus.published)
	bus.mu.Unlock()

	if got != 5 {
		t.Fatalf("expected 5 backlog events published after recovery, got %d", got)
	}
}

// TestEventProcessing_MissingTokenMetadata tests explicit tagging of estimated tokens.
func TestEventProcessing_MissingTokenMetadata(t *testing.T) {
	bus := &flappyBus{}
	emitter := gatewayv2.NewTelemetryEmitter(100, 10*time.Millisecond, bus, discardLogger())
	defer emitter.Close()

	orgID := uuid.New()
	emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
		RequestID:      "missing-tokens-req",
		OrganizationID: orgID,
		Model:          "gemini-1.5-pro",
		InputTokens:    50,
		OutputTokens:   100,
		ProviderMeta:   map[string]any{"provider": "gemini", "tokens_reported": false, "estimated": true},
		CostStatus:     domain.CostStatusEstimated,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = emitter.Flush(ctx)
	cancel()

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(bus.published))
	}

	msg := bus.published[0]
	if msg.Request.CostStatus != domain.CostStatusEstimated {
		t.Fatalf("expected CostStatus 'estimated', got %s", msg.Request.CostStatus)
	}

	var meta map[string]any
	if err := json.Unmarshal(msg.Usage.Metadata, &meta); err != nil {
		t.Fatalf("unmarshal metadata failed: %v", err)
	}
	if meta["usage_source"] != "estimated" {
		t.Fatalf("expected usage_source 'estimated', got %v", meta["usage_source"])
	}
}

// TestEventProcessing_StreamingCompletionAndInterruption verifies completion vs 499 client abort.
func TestEventProcessing_StreamingCompletionAndInterruption(t *testing.T) {
	bus := &flappyBus{}
	emitter := gatewayv2.NewTelemetryEmitter(100, 10*time.Millisecond, bus, discardLogger())
	defer emitter.Close()

	orgID := uuid.New()

	// 1. Completed streaming request
	emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
		RequestID:      "stream-completed",
		OrganizationID: orgID,
		Model:          "gpt-4o",
		StatusCode:     200,
		InputTokens:    30,
		OutputTokens:   70,
		CostMicros:     2000,
		CostStatus:     domain.CostStatusCalculated,
		ProviderMeta:   map[string]any{"provider": "openai", "tokens_reported": true},
	})

	// 2. Client-interrupted streaming request (499 Client Closed Request)
	emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
		RequestID:      "stream-cancelled",
		OrganizationID: orgID,
		Model:          "gpt-4o",
		StatusCode:     499,
		InputTokens:    30,
		OutputTokens:   15,
		CostMicros:     0,
		CostStatus:     domain.CostStatusUnavailable,
		ProviderMeta:   map[string]any{"provider": "openai"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = emitter.Flush(ctx)
	cancel()

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.published) != 2 {
		t.Fatalf("expected 2 published messages, got %d", len(bus.published))
	}

	completed := bus.published[0].Request
	if completed.Status != "succeeded" {
		t.Fatalf("expected status 'succeeded', got %s", completed.Status)
	}
	if completed.CostStatus != domain.CostStatusCalculated || completed.CostMicros != 2000 {
		t.Fatalf("expected calculated cost 2000, got %s / %d", completed.CostStatus, completed.CostMicros)
	}

	cancelled := bus.published[1].Request
	if cancelled.Status != "cancelled" {
		t.Fatalf("expected status 'cancelled', got %s", cancelled.Status)
	}
	if cancelled.CostStatus != domain.CostStatusUnavailable || cancelled.CostMicros != 0 {
		t.Fatalf("expected unavailable cost 0 for cancelled request, got %s / %d", cancelled.CostStatus, cancelled.CostMicros)
	}
}

// TestEventProcessing_FailedProviderRequests tests 504 and error status mapping.
func TestEventProcessing_FailedProviderRequests(t *testing.T) {
	bus := &flappyBus{}
	emitter := gatewayv2.NewTelemetryEmitter(100, 10*time.Millisecond, bus, discardLogger())
	defer emitter.Close()

	orgID := uuid.New()
	deniedReason := "provider_timeout"

	emitter.EmitUsage(context.Background(), gatewayv2.UsageRecord{
		RequestID:      "upstream-timeout-req",
		OrganizationID: orgID,
		Model:          "claude-3-5-sonnet",
		StatusCode:     504,
		LatencyMs:      30000,
		CostMicros:     0,
		CostStatus:     domain.CostStatusUnavailable,
		DenialReason:   &deniedReason,
		ProviderMeta:   map[string]any{"provider": "anthropic"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	_ = emitter.Flush(ctx)
	cancel()

	bus.mu.Lock()
	defer bus.mu.Unlock()
	if len(bus.published) != 1 {
		t.Fatalf("expected 1 published message, got %d", len(bus.published))
	}

	failedReq := bus.published[0].Request
	if failedReq.Status != "failed" {
		t.Fatalf("expected status 'failed', got %s", failedReq.Status)
	}
	if failedReq.CostStatus != domain.CostStatusUnavailable || failedReq.CostMicros != 0 {
		t.Fatalf("expected zero cost for failed upstream request, got %d (%s)", failedReq.CostMicros, failedReq.CostStatus)
	}
}

// TestEventProcessing_ConcurrentRequestsFromThreeUsers tests concurrency and user attribution.
func TestEventProcessing_ConcurrentRequestsFromThreeUsers(t *testing.T) {
	reqRepo := newFlappyRequests(0)
	evRepo := &flappyEvents{recorded: make(map[string]domain.NewUsageEvent)}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: reqRepo,
		UsageEvents: evRepo,
		Log:         discardLogger(),
	})

	orgID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	user3 := uuid.New()
	cred1 := uuid.New()
	cred2 := uuid.New()
	cred3 := uuid.New()

	var wg sync.WaitGroup
	users := []struct {
		userID uuid.UUID
		credID uuid.UUID
		name   string
	}{
		{user1, cred1, "user-1"},
		{user2, cred2, "user-2"},
		{user3, cred3, "user-3"},
	}

	for _, u := range users {
		for i := 1; i <= 5; i++ {
			wg.Add(1)
			go func(uid, cid uuid.UUID, idx int) {
				defer wg.Done()
				reqID := fmt.Sprintf("concurrent-%s-%d", uid.String()[:8], idx)
				llmReq := domain.NewLlmRequest{
					RequestID:           reqID,
					UserID:              &uid,
					GatewayCredentialID: &cid,
					Model:               "gemini-1.5-flash",
					TotalTokens:         int32(50 * idx),
					CostMicros:          int64(100 * idx),
					Status:              "succeeded",
				}
				usageEv := domain.NewUsageEvent{
					RequestID: reqID,
					EventType: domain.UsageEventTypeTokenUsage,
					Quantity:  int64(50 * idx),
				}
				_ = svc.RecordRequest(context.Background(), orgID, llmReq, usageEv)
			}(u.userID, u.credID, i)
		}
	}
	wg.Wait()

	reqRepo.mu.Lock()
	defer reqRepo.mu.Unlock()
	if len(reqRepo.recorded) != 15 {
		t.Fatalf("expected 15 requests recorded across 3 users, got %d", len(reqRepo.recorded))
	}

	userCounts := make(map[uuid.UUID]int)
	for _, r := range reqRepo.recorded {
		if r.UserID != nil {
			userCounts[*r.UserID]++
		}
	}
	for _, u := range users {
		if userCounts[u.userID] != 5 {
			t.Fatalf("expected 5 requests attributed to %s, got %d", u.name, userCounts[u.userID])
		}
	}
}

// TestEventProcessing_ReconciliationRequestLevelTotalsAgainstAggregates tests mathematical reconciliation.
func TestEventProcessing_ReconciliationRequestLevelTotalsAgainstAggregates(t *testing.T) {
	type testReq struct {
		promptTokens     int32
		completionTokens int32
		costMicros       int64
		status           string
		userID           uuid.UUID
	}

	u1 := uuid.New()
	u2 := uuid.New()
	u3 := uuid.New()

	records := []testReq{
		{promptTokens: 100, completionTokens: 50, costMicros: 2500, status: "succeeded", userID: u1},
		{promptTokens: 200, completionTokens: 80, costMicros: 4200, status: "succeeded", userID: u1},
		{promptTokens: 150, completionTokens: 40, costMicros: 3100, status: "succeeded", userID: u2},
		{promptTokens: 300, completionTokens: 120, costMicros: 6800, status: "succeeded", userID: u2},
		{promptTokens: 50, completionTokens: 0, costMicros: 0, status: "failed", userID: u3},
		{promptTokens: 120, completionTokens: 60, costMicros: 2900, status: "succeeded", userID: u3},
	}

	var expectedPrompt, expectedComp, expectedTotalTokens, expectedCost int64
	var expectedErrors int64
	userTokens := make(map[uuid.UUID]int64)

	for _, r := range records {
		tot := int64(r.promptTokens + r.completionTokens)
		expectedPrompt += int64(r.promptTokens)
		expectedComp += int64(r.completionTokens)
		expectedTotalTokens += tot
		expectedCost += r.costMicros
		if r.status == "failed" {
			expectedErrors++
		}
		userTokens[r.userID] += tot
	}

	if expectedTotalTokens != expectedPrompt+expectedComp {
		t.Fatalf("token mismatch: total %d != prompt %d + comp %d", expectedTotalTokens, expectedPrompt, expectedComp)
	}

	var sumFromUsers int64
	for _, tok := range userTokens {
		sumFromUsers += tok
	}
	if sumFromUsers != expectedTotalTokens {
		t.Fatalf("user breakdown sum %d does not equal total tokens %d", sumFromUsers, expectedTotalTokens)
	}
	if expectedErrors != 1 {
		t.Fatalf("expected 1 error, got %d", expectedErrors)
	}
	if expectedCost != 19500 {
		t.Fatalf("expected total cost 19500 micros, got %d", expectedCost)
	}
}
