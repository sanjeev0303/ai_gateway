// Package eventbus provides the transports that decouple the gateway data plane
// from the worker's usage ledger. It currently ships a single in-process bus,
// which serves local and single-node deployments. A distributed transport is
// not implemented yet: the ports.EventBus contract is the seam it will fill.
package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"ai-governance-saas/backend/internal/ports"
)

// ErrQueueFull reports that the bounded queue had no room, so the message was
// dropped rather than blocking the caller.
var ErrQueueFull = errors.New("eventbus: queue full")

// InMemory is a single-process EventBus backed by a bounded channel.
type InMemory struct {
	queue   chan ports.UsageEventMessage
	dropped atomic.Int64
	log     *slog.Logger
}

// NewInMemory returns a bounded in-process bus. Log is optional.
func NewInMemory(size int, log *slog.Logger) *InMemory {
	if size <= 0 {
		size = 10000
	}
	if log == nil {
		log = slog.Default()
	}
	return &InMemory{
		queue: make(chan ports.UsageEventMessage, size),
		log:   log,
	}
}

// Publish enqueues without blocking. A full queue drops the message and returns
// ErrQueueFull so the data plane is never stalled by analytics backpressure.
func (b *InMemory) Publish(_ context.Context, msg ports.UsageEventMessage) error {
	select {
	case b.queue <- msg:
		return nil
	default:
		if n := b.dropped.Add(1); n%100 == 1 {
			b.log.Error("eventbus queue full, dropped usage events", "dropped_total", n)
		}
		return ErrQueueFull
	}
}

// Run consumes queued messages until ctx is cancelled. A handler failure is
// logged and does not stop the loop, so one bad message cannot wedge the ledger.
func (b *InMemory) Run(ctx context.Context, handle func(context.Context, ports.UsageEventMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-b.queue:
			if err := handle(ctx, msg); err != nil {
				b.log.Error("eventbus: usage handler failed",
					"err", err, "request_id", msg.Usage.RequestID)
			}
		}
	}
}
