package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/testutil"
)

func TestOutbox(t *testing.T) {
	store := testutil.StoreForTest(t, false)
	pool := store.Pool()
	ctx := context.Background()

	truncate := func(t *testing.T) {
		t.Helper()
		if _, err := pool.Exec(ctx, `TRUNCATE event_outbox, event_outbox_dead_letter`); err != nil {
			t.Fatalf("truncate: %v", err)
		}
	}
	count := func(t *testing.T, table, kind string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE kind = $1`, kind).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}

	t.Run("publishes and drains usage events exactly through the handler", func(t *testing.T) {
		truncate(t)
		bus := NewOutbox(pool, slog.New(slog.DiscardHandler))
		bus.poll = 20 * time.Millisecond

		const n = 50
		for i := 0; i < n; i++ {
			if err := bus.Publish(ctx, ports.UsageEventMessage{OrganizationID: uuid.New()}); err != nil {
				t.Fatalf("publish: %v", err)
			}
		}

		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		got := make(chan struct{}, n)
		go func() { _ = bus.Run(runCtx, func(context.Context, ports.UsageEventMessage) error { got <- struct{}{}; return nil }) }()

		for i := 0; i < n; i++ {
			select {
			case <-got:
			case <-time.After(10 * time.Second):
				t.Fatalf("timed out after %d/%d events", i, n)
			}
		}
		// Everything handled must be deleted from the outbox. Check while the
		// drain is still running — cancelling mid-batch races the commit (and,
		// correctly, leaves rows for at-least-once reprocessing).
		eventuallyZero(t, func() int { return count(t, "event_outbox", "usage") })
	})

	t.Run("routes a non-retryable failure to the dead-letter table", func(t *testing.T) {
		truncate(t)
		bus := NewOutbox(pool, slog.New(slog.DiscardHandler))
		bus.poll = 20 * time.Millisecond

		if err := bus.Publish(ctx, ports.UsageEventMessage{OrganizationID: uuid.New()}); err != nil {
			t.Fatalf("publish: %v", err)
		}
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		go func() {
			_ = bus.Run(runCtx, func(context.Context, ports.UsageEventMessage) error {
				return fmt.Errorf("%w: poison", ports.ErrNonRetryable)
			})
		}()

		// It must leave the live table and land in dead-letter, not loop forever.
		eventuallyZero(t, func() int { return count(t, "event_outbox", "usage") })
		if dl := count(t, "event_outbox_dead_letter", "usage"); dl != 1 {
			t.Fatalf("expected 1 dead-lettered event, got %d", dl)
		}
	})

	t.Run("drains security events on its own stream", func(t *testing.T) {
		truncate(t)
		bus := NewOutbox(pool, slog.New(slog.DiscardHandler))
		bus.poll = 20 * time.Millisecond

		if err := bus.PublishSecurityEvent(ctx, ports.SecurityEventMessage{OrganizationID: uuid.New(), Category: "test"}); err != nil {
			t.Fatalf("publish security: %v", err)
		}
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		got := make(chan struct{}, 1)
		go func() { _ = bus.RunSecurity(runCtx, func(context.Context, ports.SecurityEventMessage) error { got <- struct{}{}; return nil }) }()
		select {
		case <-got:
		case <-time.After(10 * time.Second):
			t.Fatal("security event not delivered")
		}
		eventuallyZero(t, func() int { return count(t, "event_outbox", "security") })
	})
}

// eventuallyZero waits up to 5s for f to reach 0 (the drain commits slightly
// after the handler returns).
func eventuallyZero(t *testing.T, f func() int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f() == 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("value did not reach 0, last=%d", f())
}
