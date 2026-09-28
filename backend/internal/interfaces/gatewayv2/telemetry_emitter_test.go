// Package gatewayv2 contract tests for the telemetry emitter: defaults,
// queue/drop accounting, lifecycle and the no-op variant.
package gatewayv2

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTelemetryEmitter_DefaultsAndClose(t *testing.T) {
	e := NewTelemetryEmitter(0, 0, nil, nil)
	requireTrue(t, e.bufferSize == 10000, "buffer size default not applied")
	requireTrue(t, e.flushPeriod == 5*time.Second, "flush period default not applied")
	requireTrue(t, e.logger != nil, "logger default not applied")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_StatsStartZero(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, newRecordingBus(), discardLogger())
	s := e.Stats()
	requireTrue(t, s.UsageEmitted == 0 && s.UsageDropped == 0 && s.ErrorsEmitted == 0 && s.ErrorsDropped == 0, "new emitter must have zero stats")
	requireTrue(t, s.FlushCount == 0, "flush count should start zero")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_EmitThenClose(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, newRecordingBus(), discardLogger())
	e.EmitUsage(context.Background(), UsageRecord{Model: "claude-sonnet-4-5"})
	e.EmitError(context.Background(), ErrorRecord{ErrorType: "rate_limit_exceeded"})

	requireTrue(t, e.Stats().UsageEmitted == 1, "usage not counted")
	requireTrue(t, e.Stats().ErrorsEmitted == 1, "error not counted")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_Flush(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, newRecordingBus(), discardLogger())
	requireNoErr(t, e.Flush(context.Background()), "Flush")
	requireNoErr(t, e.Close(), "Close")
}

// Queue-full accounting is deterministic only with the workers stopped, so the
// emitter is built directly without starting goroutines.
func TestTelemetryEmitter_UsageDropsWhenQueueFull(t *testing.T) {
	e := &TelemetryEmitterImpl{
		usageQueue:  make(chan UsageRecord, 1),
		errorQueue:  make(chan ErrorRecord, 1),
		logger:      discardLogger(),
		stopCh:      make(chan struct{}),
		bufferSize:  1,
		flushPeriod: time.Hour,
	}

	e.EmitUsage(context.Background(), UsageRecord{})
	e.EmitUsage(context.Background(), UsageRecord{})

	s := e.Stats()
	requireTrue(t, s.UsageEmitted == 1, "first record should be emitted")
	requireTrue(t, s.UsageDropped == 1, "second record should be dropped")
}

func TestTelemetryEmitter_ErrorDropsWhenQueueFull(t *testing.T) {
	e := &TelemetryEmitterImpl{
		usageQueue:  make(chan UsageRecord, 1),
		errorQueue:  make(chan ErrorRecord, 1),
		logger:      discardLogger(),
		stopCh:      make(chan struct{}),
		bufferSize:  1,
		flushPeriod: time.Hour,
	}

	e.EmitError(context.Background(), ErrorRecord{})
	e.EmitError(context.Background(), ErrorRecord{})

	s := e.Stats()
	requireTrue(t, s.ErrorsEmitted == 1, "first record should be emitted")
	requireTrue(t, s.ErrorsDropped == 1, "second record should be dropped")
}

func TestNullTelemetryEmitter_Noop(t *testing.T) {
	var e NullTelemetryEmitter
	e.EmitUsage(context.Background(), UsageRecord{})
	e.EmitError(context.Background(), ErrorRecord{})
	requireNoErr(t, e.Flush(context.Background()), "Flush")
	requireNoErr(t, e.Close(), "Close")
}

func TestNewLlmRequestFrom_PreservesRequestTime(t *testing.T) {
	observedAt := time.Date(2026, 3, 4, 5, 6, 7, 0, time.FixedZone("plus-two", 2*3600))
	got := newLlmRequestFrom(UsageRecord{
		RequestID:   "req-1",
		SubjectID:   uuid.New(),
		Timestamp:   observedAt,
		LatencyMs:   12,
		InputTokens: 3, OutputTokens: 4, CacheReadTokens: 1,
	}, "openai", "succeeded")

	requireTrue(t, got.CreatedAt.Equal(observedAt), "request time must survive the async hop, got %s want %s", got.CreatedAt, observedAt)
	requireTrue(t, got.CreatedAt.Location() == time.UTC, "created_at must be normalized to UTC")
	requireTrue(t, got.TotalTokens == 7, "total tokens = %d, want 7", got.TotalTokens)
}

func TestNewLlmRequestFrom_AttributionOnlyForUserSubjects(t *testing.T) {
	subjectID := uuid.New()

	user := newLlmRequestFrom(UsageRecord{
		RequestID: "req-user", SubjectType: "user", SubjectID: subjectID,
	}, "openai", "succeeded")
	requireTrue(t, user.UserID != nil && *user.UserID == subjectID, "user subject must populate user_id")

	service := newLlmRequestFrom(UsageRecord{
		RequestID: "req-sa", SubjectType: "service_account", SubjectID: subjectID,
	}, "openai", "succeeded")
	requireTrue(t, service.UserID == nil, "service_account must not be written to user_id")
}
