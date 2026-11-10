// Package dblog copies a process's warning and error log lines into Postgres
// (service_logs), so the admin console can search them without a log stack.
// Writing is asynchronous and lossy by design: a slow or down database must
// never slow the request path or block logging itself.
package dblog

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ai-governance-saas/backend/internal/platform/logging"
)

const (
	bufferSize = 1000
	batchSize  = 100
	flushEvery = 2 * time.Second
)

type entry struct {
	level   string
	message string
	attrs   map[string]any
	at      time.Time
}

// Sink owns the buffer and the goroutine that drains it into the table.
type Sink struct {
	pool    *pgxpool.Pool
	service string
	queue   chan entry
	dropped atomic.Int64
}

// NewSink starts draining into service_logs until ctx ends.
func NewSink(ctx context.Context, pool *pgxpool.Pool, service string) *Sink {
	s := &Sink{pool: pool, service: service, queue: make(chan entry, bufferSize)}
	go s.run(ctx)
	return s
}

// Wrap returns a logger that logs through base as before and also copies
// WARN and ERROR records into the sink.
func (s *Sink) Wrap(base *slog.Logger) *slog.Logger {
	return slog.New(&handler{next: base.Handler(), sink: s})
}

func (s *Sink) push(e entry) {
	select {
	case s.queue <- e:
	default:
		s.dropped.Add(1)
	}
}

func (s *Sink) run(ctx context.Context) {
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()
	batch := make([]entry, 0, batchSize)
	for {
		select {
		case <-ctx.Done():
			// Last chance for what is buffered; a short deadline so shutdown
			// is never held up by the database.
			flushCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			s.flush(flushCtx, drain(s.queue, batch))
			cancel()
			return
		case e := <-s.queue:
			batch = append(batch, e)
			if len(batch) >= batchSize {
				s.flush(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				s.flush(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

func drain(queue chan entry, batch []entry) []entry {
	for {
		select {
		case e := <-queue:
			batch = append(batch, e)
		default:
			return batch
		}
	}
}

func (s *Sink) flush(ctx context.Context, batch []entry) {
	if len(batch) == 0 {
		return
	}
	rows := make([][]any, 0, len(batch))
	for _, e := range batch {
		attrs, err := json.Marshal(e.attrs)
		if err != nil {
			attrs = []byte("{}")
		}
		rows = append(rows, []any{s.service, e.level, e.message, attrs, e.at})
	}
	_, err := s.pool.CopyFrom(ctx, pgx.Identifier{"service_logs"},
		[]string{"service", "level", "message", "attrs", "logged_at"}, pgx.CopyFromRows(rows))
	if err != nil {
		// Never through slog: that would feed this sink its own failure.
		fmt.Fprintf(os.Stderr, "dblog: dropped %d log lines: %v\n", len(batch), err)
	}
	if n := s.dropped.Swap(0); n > 0 {
		fmt.Fprintf(os.Stderr, "dblog: buffer full, dropped %d log lines\n", n)
	}
}

// handler forwards every record to next and copies WARN and above to the sink.
type handler struct {
	next   slog.Handler
	sink   *Sink
	attrs  []slog.Attr
	prefix string
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *handler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		level := "WARN"
		if r.Level >= slog.LevelError {
			level = "ERROR"
		}
		attrs := make(map[string]any, len(h.attrs)+r.NumAttrs())
		for _, a := range h.attrs {
			addAttr(attrs, "", a)
		}
		r.Attrs(func(a slog.Attr) bool {
			addAttr(attrs, h.prefix, a)
			return true
		})
		at := r.Time
		if at.IsZero() {
			at = time.Now()
		}
		h.sink.push(entry{level: level, message: r.Message, attrs: attrs, at: at})
	}
	return h.next.Handle(ctx, r)
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Keys take the group prefix in force now, not whatever group is opened
	// later, so they are stored already prefixed.
	merged := append([]slog.Attr{}, h.attrs...)
	for _, a := range attrs {
		a.Key = h.prefix + a.Key
		merged = append(merged, a)
	}
	return &handler{next: h.next.WithAttrs(attrs), sink: h.sink, attrs: merged, prefix: h.prefix}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{next: h.next.WithGroup(name), sink: h.sink, attrs: h.attrs, prefix: h.prefix + name + "."}
}

// addAttr flattens groups into dotted keys and stores plain values. Values go
// through the same redaction the stdout handler applies, because secrets must
// not reach the table either.
func addAttr(dst map[string]any, prefix string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Value.Kind() == slog.KindGroup {
		for _, g := range a.Value.Group() {
			addAttr(dst, prefix+a.Key+".", g)
		}
		return
	}
	a = logging.RedactAttr(nil, a)
	value := a.Value.Any()
	// An error marshals to {} in JSON; keep its message instead.
	if err, ok := value.(error); ok {
		value = err.Error()
	}
	dst[prefix+a.Key] = value
}
