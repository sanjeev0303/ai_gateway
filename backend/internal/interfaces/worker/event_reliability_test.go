package worker

import (
	"context"
	"errors"
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
