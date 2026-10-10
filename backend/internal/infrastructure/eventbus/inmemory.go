package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"

	"ai-governance-saas/backend/internal/ports"
)

var ErrQueueFull = errors.New("eventbus: queue full")

type InMemory struct {
	queue         chan ports.UsageEventMessage
	securityQueue chan ports.SecurityEventMessage
	dropped       atomic.Int64
	log           *slog.Logger
}

func NewInMemory(size int, log *slog.Logger) *InMemory {
	if size <= 0 {
		size = 10000
	}
	if log == nil {
		log = slog.Default()
	}
	return &InMemory{
		queue:         make(chan ports.UsageEventMessage, size),
		securityQueue: make(chan ports.SecurityEventMessage, size),
		log:           log,
	}
}

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

func (b *InMemory) PublishSecurityEvent(_ context.Context, msg ports.SecurityEventMessage) error {
	select {
	case b.securityQueue <- msg:
		return nil
	default:
		if n := b.dropped.Add(1); n%100 == 1 {
			b.log.Error("eventbus queue full, dropped security events", "dropped_total", n)
		}
		return ErrQueueFull
	}
}

func (b *InMemory) Run(ctx context.Context, handle func(context.Context, ports.UsageEventMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-b.queue:
			if err := handle(ctx, msg); err != nil {
				b.log.Error("eventbus: usage handler failed", "err", err, "request_id", msg.Usage.RequestID)
			}
		}
	}
}

func (b *InMemory) RunSecurity(ctx context.Context, handle func(context.Context, ports.SecurityEventMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-b.securityQueue:
			if err := handle(ctx, msg); err != nil {
				b.log.Error("eventbus: security handler failed", "err", err, "category", msg.Category)
			}
		}
	}
}
