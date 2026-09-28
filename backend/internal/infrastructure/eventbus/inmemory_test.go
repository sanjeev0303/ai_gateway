// Package eventbus contract tests for the in-process bus: default sizing,
// delivery, ordering, drop-on-full backpressure, handler-error isolation and
// cancellation. Convention matches the rest of the module: hand-written stubs,
// stdlib testing, no testify/gomock.
package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func requireTrue(t *testing.T, cond bool, format string, args ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, args...)
	}
}

func requireErrIs(t *testing.T, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("expected error %v, got %v", target, err)
	}
}

func message(requestID string) ports.UsageEventMessage {
	return ports.UsageEventMessage{Usage: domain.NewUsageEvent{RequestID: requestID}}
}

// collector accumulates delivered request ids in arrival order.
type collector struct {
	mu  sync.Mutex
	ids []string
}

func (c *collector) handle(_ context.Context, msg ports.UsageEventMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ids = append(c.ids, msg.Usage.RequestID)
	return nil
}

func (c *collector) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.ids...)
}

// runUntil starts Run in the background and waits until at least n messages have
// been handled, so tests never depend on a fixed sleep.
func runUntil(t *testing.T, bus *InMemory, c *collector, n int) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = bus.Run(ctx, c.handle)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if len(c.snapshot()) >= n {
			return cancel
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("timed out waiting for %d messages, got %v", n, c.snapshot())
		case <-done:
			t.Fatalf("consumer stopped early after %v", c.snapshot())
		case <-time.After(time.Millisecond):
		}
	}
}

func TestNewInMemory_Defaults(t *testing.T) {
	b := NewInMemory(0, nil)
	requireTrue(t, cap(b.queue) == 10000, "queue size default not applied, got %d", cap(b.queue))
	requireTrue(t, b.log != nil, "logger default not applied")
}

func TestInMemory_DeliversInOrder(t *testing.T) {
	b := NewInMemory(8, discardLogger())
	c := &collector{}

	ids := []string{"req-1", "req-2", "req-3"}
	for _, id := range ids {
		if err := b.Publish(context.Background(), message(id)); err != nil {
			t.Fatalf("publish %s: %v", id, err)
		}
	}

	cancel := runUntil(t, b, c, len(ids))
	cancel()

	got := c.snapshot()
	requireTrue(t, len(got) == len(ids), "want %d delivered, got %v", len(ids), got)
	for i, want := range ids {
		requireTrue(t, got[i] == want, "out of order at %d: got %s want %s", i, got[i], want)
	}
}

func TestInMemory_PublishIsNonBlockingWhenFull(t *testing.T) {
	b := NewInMemory(1, discardLogger())

	if err := b.Publish(context.Background(), message("req-1")); err != nil {
		t.Fatalf("first publish must succeed: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- b.Publish(context.Background(), message("req-2")) }()
	select {
	case err := <-done:
		requireErrIs(t, err, ErrQueueFull)
	case <-time.After(time.Second):
		t.Fatal("Publish blocked on a full queue; the data plane must never stall")
	}

	requireTrue(t, b.dropped.Load() == 1, "drop must be counted, got %d", b.dropped.Load())
}

func TestInMemory_HandlerErrorDoesNotStopTheLoop(t *testing.T) {
	b := NewInMemory(8, discardLogger())
	boom := errors.New("boom")

	var mu sync.Mutex
	seen := 0
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = b.Run(ctx, func(context.Context, ports.UsageEventMessage) error {
			mu.Lock()
			seen++
			mu.Unlock()
			return boom
		})
	}()

	if err := b.Publish(context.Background(), message("req-1")); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := b.Publish(context.Background(), message("req-2")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		got := seen
		mu.Unlock()
		if got >= 2 {
			break
		}
		select {
		case <-deadline:
			cancel()
			<-done
			t.Fatalf("a failing handler wedged the loop, saw %d of 2", got)
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	<-done
}

func TestInMemory_RunReturnsOnContextCancel(t *testing.T) {
	b := NewInMemory(8, discardLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := b.Run(ctx, func(context.Context, ports.UsageEventMessage) error { return nil })
	requireErrIs(t, err, context.Canceled)
}

func TestInMemory_PublishNeverBlocksWhenNoConsumer(t *testing.T) {
	b := NewInMemory(2, discardLogger())
	for i := range 4 {
		if err := b.Publish(context.Background(), message(string(rune('a'+i)))); err != nil {
			requireErrIs(t, err, ErrQueueFull)
		}
	}
	requireTrue(t, b.dropped.Load() == 2, "want 2 drops, got %d", b.dropped.Load())
}
