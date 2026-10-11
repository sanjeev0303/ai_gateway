package eventbus

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"ai-governance-saas/backend/internal/ports"
)

var ErrQueueFull = errors.New("eventbus: queue full")

// InMemory is a non-durable, ephemeral in-process event transport intended
// ONLY for local unit tests and lightweight mock scenarios.
//
// WARNING: This transport does NOT persist events across process restarts,
// does NOT survive crashes, and will silently drop messages if the process exits
// while events are queued. For durable, production-grade event processing, use
// NATS JetStream (AIGW_EVENTBUS_KIND=nats).
type InMemory struct {
	queue         chan ports.UsageEventMessage
	securityQueue chan ports.SecurityEventMessage
	deadLetters   []ports.DeadLetterUsageMessage
	mu            sync.Mutex
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
		deadLetters:   make([]ports.DeadLetterUsageMessage, 0),
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
				if errors.Is(err, ports.ErrNonRetryable) {
					b.mu.Lock()
					b.deadLetters = append(b.deadLetters, ports.DeadLetterUsageMessage{
						OriginalMessage: msg,
						Error:           err.Error(),
						Deliveries:      1,
						FailedAt:        time.Now().UTC(),
					})
					b.mu.Unlock()
					b.log.Error("eventbus: usage event dead-lettered (non-retryable)", "err", err, "request_id", msg.Usage.RequestID)
				} else {
					b.log.Error("eventbus: usage handler failed", "err", err, "request_id", msg.Usage.RequestID)
				}
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

// DeadLetters returns recorded dead-letter messages (primarily used in tests).
func (b *InMemory) DeadLetters() []ports.DeadLetterUsageMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	res := make([]ports.DeadLetterUsageMessage, len(b.deadLetters))
	copy(res, b.deadLetters)
	return res
}
