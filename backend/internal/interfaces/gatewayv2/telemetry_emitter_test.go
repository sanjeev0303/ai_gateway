// Package gatewayv2 contract tests for the telemetry emitter: defaults,
// queue/drop accounting, lifecycle and the no-op variant.
package gatewayv2

import (
	"context"
	"testing"
	"time"
)

func TestNewTelemetryEmitter_DefaultsAndClose(t *testing.T) {
	e := NewTelemetryEmitter(0, 0, nil)
	requireTrue(t, e.bufferSize == 10000, "buffer size default not applied")
	requireTrue(t, e.flushPeriod == 5*time.Second, "flush period default not applied")
	requireTrue(t, e.logger != nil, "logger default not applied")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_StatsStartZero(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, discardLogger())
	s := e.Stats()
	requireTrue(t, s.UsageEmitted == 0 && s.UsageDropped == 0 && s.ErrorsEmitted == 0 && s.ErrorsDropped == 0, "new emitter must have zero stats")
	requireTrue(t, s.FlushCount == 0, "flush count should start zero")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_EmitThenClose(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, discardLogger())
	e.EmitUsage(context.Background(), UsageRecord{Model: "claude-sonnet-4-5"})
	e.EmitError(context.Background(), ErrorRecord{ErrorType: "rate_limit_exceeded"})

	requireTrue(t, e.Stats().UsageEmitted == 1, "usage not counted")
	requireTrue(t, e.Stats().ErrorsEmitted == 1, "error not counted")
	requireNoErr(t, e.Close(), "Close")
}

func TestTelemetryEmitter_Flush(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, discardLogger())
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