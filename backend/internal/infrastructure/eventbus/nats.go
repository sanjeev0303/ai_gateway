package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"ai-governance-saas/backend/internal/ports"
)

const (
	natsStreamName    = "AIGW_USAGE"
	natsSubject       = "aigw.usage.events"
	natsDurable       = "aigw-usage-worker"
	natsAckWait       = 30 * time.Second
	natsMaxDeliver    = 10
	natsPublishWait   = 5 * time.Second
	natsMaxPending    = 10000
	natsRetryBase     = time.Second
	natsMaxRetryDelay = 30 * time.Second
)

// NATS is a distributed EventBus backed by a JetStream stream. Unlike InMemory
// it survives a restart and can carry messages between processes, so a worker
// may be hosted independently of the gateway that published the usage.
type NATS struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	log    *slog.Logger
}

// NewNATS connects to NATS and ensures the usage stream exists. The durable
// consumer is created by Run instead, so a gateway-only process publishes
// without registering a worker it does not run.
func NewNATS(ctx context.Context, url string, log *slog.Logger) (*NATS, error) {
	if url == "" {
		return nil, fmt.Errorf("eventbus: nats url is required")
	}
	if log == nil {
		log = slog.Default()
	}
	conn, err := nats.Connect(url,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("eventbus: nats connect %s: %w", url, err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("eventbus: jetstream init: %w", err)
	}
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     natsStreamName,
		Subjects: []string{natsSubject, "aigw.security.events"},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("eventbus: ensure stream %s: %w", natsStreamName, err)
	}
	return &NATS{conn: conn, js: js, stream: stream, log: log}, nil
}

// Publish sends the message and waits for JetStream to confirm it was persisted,
// bounded by natsPublishWait. The wait is safe on the data plane because the
// emitter publishes from its flush goroutine, never from the request path.
func (b *NATS) Publish(ctx context.Context, msg ports.UsageEventMessage) error {
	if pending := b.js.PublishAsyncPending(); pending >= natsMaxPending {
		b.log.Error("eventbus publish backlog full, dropped usage event",
			"pending", pending, "request_id", msg.Usage.RequestID)
		return ErrQueueFull
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("eventbus: encode usage event: %w", err)
	}
	ack, err := b.js.PublishAsync(natsSubject, data)
	if err != nil {
		return fmt.Errorf("eventbus: publish usage event: %w", err)
	}

	timer := time.NewTimer(natsPublishWait)
	defer timer.Stop()
	select {
	case <-ack.Ok():
		return nil
	case err := <-ack.Err():
		return fmt.Errorf("eventbus: publish usage event: %w", err)
	case <-timer.C:
		return fmt.Errorf("eventbus: publish usage event: %w", ErrQueueFull)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run consumes from a durable consumer until ctx is cancelled. A handler
// failure is negatively acknowledged so JetStream redelivers it with backoff
// rather than dropping usage; a message that cannot be decoded is terminal.
func (b *NATS) Run(ctx context.Context, handle func(context.Context, ports.UsageEventMessage) error) error {
	consumer, err := b.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       natsDurable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       natsAckWait,
		MaxDeliver:    natsMaxDeliver,
		BackOff:       retryBackoff(),
		FilterSubject: natsSubject,
	})
	if err != nil {
		return fmt.Errorf("eventbus: create consumer %s: %w", natsDurable, err)
	}

	cc, err := consumer.Consume(func(m jetstream.Msg) {
		var msg ports.UsageEventMessage
		if err := json.Unmarshal(m.Data(), &msg); err != nil {
			b.log.Error("eventbus: undecodable usage event, terminating",
				"err", err, "subject", m.Subject())
			m.Term()
			return
		}
		if err := handle(ctx, msg); err != nil {
			b.log.Error("eventbus: usage handler failed, redelivering",
				"err", err, "request_id", msg.Usage.RequestID, "delivery", deliveredCount(m))
			m.NakWithDelay(natsRetryBase)
			return
		}
		if err := m.Ack(); err != nil {
			b.log.Error("eventbus: ack failed", "err", err, "request_id", msg.Usage.RequestID)
		}
	})
	if err != nil {
		return fmt.Errorf("eventbus: consume %s: %w", natsDurable, err)
	}
	defer cc.Stop()

	select {
	case <-ctx.Done():
		cc.Drain()
		return ctx.Err()
	case <-cc.Closed():
		return nil
	}
}

// Close releases the connection. It is safe to call more than once.
func (b *NATS) Close() error {
	b.conn.Close()
	return nil
}

func deliveredCount(m jetstream.Msg) uint64 {
	meta, err := m.Metadata()
	if err != nil {
		return 0
	}
	return meta.NumDelivered
}

func retryBackoff() []time.Duration {
	delays := make([]time.Duration, natsMaxDeliver)
	d := natsRetryBase
	for i := range delays {
		delays[i] = d
		d *= 2
		if d > natsMaxRetryDelay {
			d = natsMaxRetryDelay
		}
	}
	return delays
}

// PublishSecurityEvent sends a security event to NATS.
func (b *NATS) PublishSecurityEvent(ctx context.Context, msg ports.SecurityEventMessage) error {
	// For simplicity, we just reuse the same error/backpressure handling logic
	// but this could be on a separate stream if necessary.
	if pending := b.js.PublishAsyncPending(); pending >= natsMaxPending {
		b.log.Error("eventbus publish backlog full, dropped security event", "pending", pending)
		return ErrQueueFull
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("eventbus: encode security event: %w", err)
	}
	ack, err := b.js.PublishAsync("aigw.security.events", data)
	if err != nil {
		return fmt.Errorf("eventbus: publish security event: %w", err)
	}

	timer := time.NewTimer(natsPublishWait)
	defer timer.Stop()
	select {
	case <-ack.Ok():
		return nil
	case err := <-ack.Err():
		return fmt.Errorf("eventbus: publish security event: %w", err)
	case <-timer.C:
		return fmt.Errorf("eventbus: publish security event: %w", ErrQueueFull)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RunSecurity consumes security events from NATS.
func (b *NATS) RunSecurity(ctx context.Context, handle func(context.Context, ports.SecurityEventMessage) error) error {
	consumer, err := b.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "aigw-security-worker",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       natsAckWait,
		MaxDeliver:    natsMaxDeliver,
		BackOff:       retryBackoff(),
		FilterSubject: "aigw.security.events",
	})
	if err != nil {
		return fmt.Errorf("eventbus: create security consumer: %w", err)
	}

	cc, err := consumer.Consume(func(m jetstream.Msg) {
		var msg ports.SecurityEventMessage
		if err := json.Unmarshal(m.Data(), &msg); err != nil {
			b.log.Error("eventbus: undecodable security event", "err", err)
			m.Term()
			return
		}
		if err := handle(ctx, msg); err != nil {
			m.NakWithDelay(natsRetryBase)
			return
		}
		m.Ack()
	})
	if err != nil {
		return fmt.Errorf("eventbus: consume security: %w", err)
	}
	defer cc.Stop()

	select {
	case <-ctx.Done():
		cc.Drain()
		return ctx.Err()
	case <-cc.Closed():
		return nil
	}
}
