package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-governance-saas/backend/internal/ports"
)

const (
	outboxBatchSize     = 100
	outboxPollInterval  = 500 * time.Millisecond
	outboxMaxAttempts   = 10
	outboxRetryBase     = time.Second
	outboxMaxRetryDelay = 30 * time.Second
)

// Outbox is a transactional-outbox EventBus backed by the event_outbox table.
// The data plane inserts a row (Publish); the worker drains it (Run) with
// SELECT ... FOR UPDATE SKIP LOCKED, so many workers can drain in parallel
// without double-processing. Because the durable usage ledger is also in
// Postgres, this removes the dual-write between a broker and the database:
// the event and its eventual record share one store. Delivery is at-least-once
// (a crash between handling and delete re-runs the handler), which the usage
// pipeline already tolerates via idempotent keys.
type Outbox struct {
	pool        *pgxpool.Pool
	log         *slog.Logger
	batchSize   int
	poll        time.Duration
	maxAttempts int
}

// NewOutbox builds an Outbox over the given pool. It does not own the pool, so
// Close is a no-op and the pool is closed by whoever created it.
func NewOutbox(pool *pgxpool.Pool, log *slog.Logger) *Outbox {
	if log == nil {
		log = slog.Default()
	}
	return &Outbox{
		pool:        pool,
		log:         log,
		batchSize:   outboxBatchSize,
		poll:        outboxPollInterval,
		maxAttempts: outboxMaxAttempts,
	}
}

func (o *Outbox) Ping(ctx context.Context) error {
	if o == nil || o.pool == nil {
		return errors.New("outbox: nil pool")
	}
	return o.pool.Ping(ctx)
}

func (o *Outbox) Close() error { return nil }

func (o *Outbox) Publish(ctx context.Context, msg ports.UsageEventMessage) error {
	return o.insert(ctx, "usage", msg)
}

func (o *Outbox) PublishSecurityEvent(ctx context.Context, msg ports.SecurityEventMessage) error {
	return o.insert(ctx, "security", msg)
}

func (o *Outbox) insert(ctx context.Context, kind string, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("eventbus: encode %s event: %w", kind, err)
	}
	_, err = o.pool.Exec(ctx, `INSERT INTO event_outbox (kind, payload) VALUES ($1, $2)`, kind, data)
	if err != nil {
		return fmt.Errorf("eventbus: enqueue %s event: %w", kind, err)
	}
	return nil
}

// Run drains usage events until ctx is cancelled.
func (o *Outbox) Run(ctx context.Context, handle func(context.Context, ports.UsageEventMessage) error) error {
	return o.drainLoop(ctx, "usage", func(ctx context.Context, payload []byte) error {
		var msg ports.UsageEventMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return fmt.Errorf("%w: decode usage event: %v", ports.ErrNonRetryable, err)
		}
		return handle(ctx, msg)
	})
}

// RunSecurity drains security events until ctx is cancelled.
func (o *Outbox) RunSecurity(ctx context.Context, handle func(context.Context, ports.SecurityEventMessage) error) error {
	return o.drainLoop(ctx, "security", func(ctx context.Context, payload []byte) error {
		var msg ports.SecurityEventMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return fmt.Errorf("%w: decode security event: %v", ports.ErrNonRetryable, err)
		}
		return handle(ctx, msg)
	})
}

func (o *Outbox) drainLoop(ctx context.Context, kind string, process func(context.Context, []byte) error) error {
	ticker := time.NewTicker(o.poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Drain full batches back-to-back so a backlog clears without
			// waiting a poll interval per batch; stop once a batch is partial.
			for {
				n, err := o.drainBatch(ctx, kind, process)
				if err != nil {
					o.log.Error("eventbus: outbox drain failed", "kind", kind, "err", err)
					break
				}
				if n < o.batchSize {
					break
				}
			}
		}
	}
}

type outboxRow struct {
	id       int64
	payload  []byte
	attempts int
}

// drainBatch claims up to batchSize due rows under FOR UPDATE SKIP LOCKED,
// runs the handler for each, and commits the results (delete on success or
// dead-letter, bump next_attempt_at with backoff on a retryable failure) in
// one transaction. It returns how many rows it claimed.
func (o *Outbox) drainBatch(ctx context.Context, kind string, process func(context.Context, []byte) error) (int, error) {
	tx, err := o.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(context.Background()) // no-op once committed

	rows, err := tx.Query(ctx, `
		SELECT id, payload, attempts FROM event_outbox
		WHERE kind = $1 AND next_attempt_at <= now()
		ORDER BY next_attempt_at, id
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, kind, o.batchSize)
	if err != nil {
		return 0, err
	}
	var items []outboxRow
	for rows.Next() {
		var it outboxRow
		if err := rows.Scan(&it.id, &it.payload, &it.attempts); err != nil {
			rows.Close()
			return 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, tx.Commit(ctx)
	}

	var done []int64
	for _, it := range items {
		perr := process(ctx, it.payload)
		if perr == nil {
			done = append(done, it.id)
			continue
		}
		attempts := it.attempts + 1
		if errors.Is(perr, ports.ErrNonRetryable) || attempts >= o.maxAttempts {
			if err := o.deadLetter(ctx, tx, kind, it.payload, attempts, perr); err != nil {
				return 0, err
			}
			done = append(done, it.id)
			o.log.Error("eventbus: outbox event dead-lettered", "kind", kind, "attempts", attempts, "err", perr)
			continue
		}
		next := time.Now().UTC().Add(backoff(attempts))
		if _, err := tx.Exec(ctx,
			`UPDATE event_outbox SET attempts = $1, next_attempt_at = $2, last_error = $3 WHERE id = $4`,
			attempts, next, perr.Error(), it.id); err != nil {
			return 0, err
		}
		o.log.Warn("eventbus: outbox handler failed, will retry", "kind", kind, "attempt", attempts, "err", perr)
	}

	if len(done) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM event_outbox WHERE id = ANY($1)`, done); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(items), nil
}

func (o *Outbox) deadLetter(ctx context.Context, tx pgx.Tx, kind string, payload []byte, attempts int, cause error) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO event_outbox_dead_letter (kind, payload, attempts, error) VALUES ($1, $2, $3, $4)`,
		kind, payload, attempts, cause.Error())
	return err
}

// backoff is 1s, 2s, 4s … capped at outboxMaxRetryDelay, matching the NATS bus.
func backoff(attempt int) time.Duration {
	d := outboxRetryBase
	for i := 1; i < attempt; i++ {
		d *= 2
		if d > outboxMaxRetryDelay {
			return outboxMaxRetryDelay
		}
	}
	return d
}
