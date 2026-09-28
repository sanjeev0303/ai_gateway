// Package worker contract tests for the usage event consumer. The consumer is
// exercised against the real in-memory bus and the real usage service, so the
// delegation path is covered end to end without a mock of either.
package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/eventbus"
	"ai-governance-saas/backend/internal/ports"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

type recordingRequests struct {
	mu      sync.Mutex
	records []domain.NewLlmRequest
}

func (f *recordingRequests) Record(_ context.Context, _ uuid.UUID, in domain.NewLlmRequest) (domain.LlmRequest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = append(f.records, in)
	return domain.LlmRequest{RequestID: in.RequestID}, nil
}

func (f *recordingRequests) GetByRequestID(context.Context, uuid.UUID, string) (domain.LlmRequest, error) {
	return domain.LlmRequest{}, errors.New("unused")
}

func (f *recordingRequests) List(context.Context, uuid.UUID, time.Time, time.Time) ([]domain.LlmRequest, error) {
	return nil, nil
}

func (f *recordingRequests) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.records)
}

type recordingEvents struct {
	mu     sync.Mutex
	events []domain.NewUsageEvent
}

func (f *recordingEvents) Record(_ context.Context, _ uuid.UUID, in domain.NewUsageEvent) (domain.UsageEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, in)
	return domain.UsageEvent{RequestID: in.RequestID, EventType: in.EventType}, nil
}

func (f *recordingEvents) Get(context.Context, uuid.UUID, string, string) (domain.UsageEvent, error) {
	return domain.UsageEvent{}, errors.New("unused")
}

func (f *recordingEvents) ListByRequest(context.Context, uuid.UUID, string) ([]domain.UsageEvent, error) {
	return nil, nil
}

func (f *recordingEvents) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func TestRunUsageConsumer_PersistsPublishedUsage(t *testing.T) {
	requests := &recordingRequests{}
	events := &recordingEvents{}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: requests,
		UsageEvents: events,
		Log:         discardLogger(),
	})
	bus := eventbus.NewInMemory(8, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		RunUsageConsumer(ctx, bus, svc, discardLogger())
	}()

	orgID := uuid.New()
	msg := ports.UsageEventMessage{
		OrganizationID: orgID,
		Request:        domain.NewLlmRequest{RequestID: "req-1"},
		Usage:          domain.NewUsageEvent{RequestID: "req-1", EventType: domain.UsageEventTypeTokenUsage},
	}
	if err := bus.Publish(context.Background(), msg); err != nil {
		cancel()
		<-done
		t.Fatalf("publish: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for requests.count() == 0 || events.count() == 0 {
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("consumer did not persist, requests=%d events=%d", requests.count(), events.count())
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not stop on context cancellation")
	}

	if requests.records[0].RequestID != "req-1" {
		t.Fatalf("request id lost, got %q", requests.records[0].RequestID)
	}
	if events.events[0].EventType != domain.UsageEventTypeTokenUsage {
		t.Fatalf("event type lost, got %q", events.events[0].EventType)
	}
}

func TestRunUsageConsumer_ToleratesNilLogger(t *testing.T) {
	requests := &recordingRequests{}
	events := &recordingEvents{}
	svc := usage.NewService(usage.ServiceDeps{
		LlmRequests: requests,
		UsageEvents: events,
	})
	bus := eventbus.NewInMemory(1, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		RunUsageConsumer(ctx, bus, svc, nil)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not return on an already-cancelled context")
	}
}
