package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"ai-governance-saas/backend/internal/infrastructure/eventbus"
	"ai-governance-saas/backend/internal/platform/config"
	"ai-governance-saas/backend/internal/ports"
)

const usageEventQueueSize = 10000

// usageBus is the transport the data plane publishes usage onto. inProcess
// records the topology that transport implies: a single-process bus cannot
// reach another container, so its consumer has to be hosted beside its
// publisher. A shared transport flips this and the consumer moves to the
// worker, with no other wiring changing.
type usageBus struct {
	ports.EventBus
	inProcess bool
	closer    io.Closer
	ping      func(context.Context) error
}

func (b usageBus) Close() error {
	if b.closer == nil {
		return nil
	}
	return b.closer.Close()
}

func (b usageBus) Ping(ctx context.Context) error {
	if b.ping != nil {
		return b.ping(ctx)
	}
	return nil
}

// pool may be nil for processes without a database; the postgres kind requires
// it and errors clearly when it is missing.
func newUsageBus(ctx context.Context, cfg *config.EventbusConfig, pool *pgxpool.Pool, log *slog.Logger) (usageBus, error) {
	switch cfg.Kind {
	case "inmemory":
		return usageBus{EventBus: eventbus.NewInMemory(usageEventQueueSize, log), inProcess: true}, nil
	case "nats":
		bus, err := eventbus.NewNATS(ctx, cfg.URL, log)
		if err != nil {
			return usageBus{}, err
		}
		return usageBus{
			EventBus:  bus,
			inProcess: false,
			closer:    bus,
			ping:      bus.Ping,
		}, nil
	case "postgres":
		if pool == nil {
			return usageBus{}, fmt.Errorf("eventbus kind %q requires the database (AIGW_DATABASE_ENABLED=true)", cfg.Kind)
		}
		bus := eventbus.NewOutbox(pool, log)
		// The outbox reuses the service pool, so it is shared/durable (consumer
		// runs in the worker) and does not own a connection to close.
		return usageBus{EventBus: bus, inProcess: false, ping: bus.Ping}, nil
	default:
		return usageBus{}, fmt.Errorf("eventbus kind %q is not available in this build", cfg.Kind)
	}
}
