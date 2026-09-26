// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// TelemetryEmitterImpl emits usage and error events asynchronously.
type TelemetryEmitterImpl struct {
	usageQueue  chan UsageRecord
	errorQueue  chan ErrorRecord
	logger      *slog.Logger
	wg          sync.WaitGroup
	stopCh      chan struct{}
	flusher     *periodicFlusher
	bufferSize  int
	flushPeriod time.Duration
	stats       EmitterStats
}

// EmitterStats tracks emitter performance.
type EmitterStats struct {
	UsageEmitted   int64
	UsageDropped   int64
	ErrorsEmitted  int64
	ErrorsDropped  int64
	FlushCount     int64
	LastFlushTime  int64 // unix nano
}

// NewTelemetryEmitter creates a new telemetry emitter.
func NewTelemetryEmitter(bufferSize int, flushPeriod time.Duration, logger *slog.Logger) *TelemetryEmitterImpl {
	if logger == nil {
		logger = slog.Default()
	}
	if bufferSize <= 0 {
		bufferSize = 10000
	}
	if flushPeriod <= 0 {
		flushPeriod = 5 * time.Second
	}

	e := &TelemetryEmitterImpl{
		usageQueue:   make(chan UsageRecord, bufferSize),
		errorQueue:   make(chan ErrorRecord, bufferSize),
		logger:       logger,
		stopCh:       make(chan struct{}),
		bufferSize:   bufferSize,
		flushPeriod:  flushPeriod,
		flusher:      newPeriodicFlusher(flushPeriod),
	}

	e.wg.Add(2)
	go e.usageWorker()
	go e.errorWorker()
	e.flusher.Start(e.flush)

	return e
}

// EmitUsage queues a usage record for async emission.
func (e *TelemetryEmitterImpl) EmitUsage(ctx context.Context, usage UsageRecord) {
	select {
	case e.usageQueue <- usage:
		atomic.AddInt64(&e.stats.UsageEmitted, 1)
	default:
		atomic.AddInt64(&e.stats.UsageDropped, 1)
		e.logger.Warn("telemetry: usage queue full, dropping record",
			"org_id", usage.OrganizationID,
			"credential_id", usage.CredentialID,
		)
	}
}

// EmitError queues an error record for async emission.
func (e *TelemetryEmitterImpl) EmitError(ctx context.Context, err ErrorRecord) {
	select {
	case e.errorQueue <- err:
		atomic.AddInt64(&e.stats.ErrorsEmitted, 1)
	default:
		atomic.AddInt64(&e.stats.ErrorsDropped, 1)
		e.logger.Warn("telemetry: error queue full, dropping record",
			"org_id", err.OrganizationID,
			"credential_id", err.CredentialID,
		)
	}
}

// usageWorker processes usage records in batches.
func (e *TelemetryEmitterImpl) usageWorker() {
	defer e.wg.Done()

	batch := make([]UsageRecord, 0, 100)
	ticker := time.NewTicker(e.flushPeriod)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		e.flushUsage(batch)
		batch = batch[:0]
		atomic.AddInt64(&e.stats.FlushCount, 1)
		atomic.StoreInt64(&e.stats.LastFlushTime, time.Now().UnixNano())
	}

	for {
		select {
		case record := <-e.usageQueue:
			batch = append(batch, record)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-e.stopCh:
			flush()
			return
		}
	}
}

// errorWorker processes error records in batches.
func (e *TelemetryEmitterImpl) errorWorker() {
	defer e.wg.Done()

	batch := make([]ErrorRecord, 0, 100)
	ticker := time.NewTicker(e.flushPeriod)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		e.flushErrors(batch)
		batch = batch[:0]
	}

	for {
		select {
		case record := <-e.errorQueue:
			batch = append(batch, record)
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-e.stopCh:
			flush()
			return
		}
	}
}

// flushUsage sends a batch of usage records to the backend.
func (e *TelemetryEmitterImpl) flushUsage(batch []UsageRecord) {
	// In production, send to telemetry backend (ClickHouse, TimescaleDB, etc.)
	// For now, log as structured JSON
	for _, record := range batch {
		data, _ := json.Marshal(record)
		e.logger.Debug("telemetry: usage", "data", string(data))
	}
}

// flushErrors sends a batch of error records to the backend.
func (e *TelemetryEmitterImpl) flushErrors(batch []ErrorRecord) {
	for _, record := range batch {
		data, _ := json.Marshal(record)
		e.logger.Debug("telemetry: error", "data", string(data))
	}
}

// flush is called by the periodic flusher.
func (e *TelemetryEmitterImpl) flush() {
	// The workers handle periodic flushing via their ticker
}

// Flush forces an immediate flush of all queues.
func (e *TelemetryEmitterImpl) Flush(ctx context.Context) error {
	// The workers flush periodically; this is a no-op for now
	return nil
}

// Close stops the emitter and drains queues.
func (e *TelemetryEmitterImpl) Close() error {
	close(e.stopCh)
	e.wg.Wait()
	e.flusher.Stop()
	return nil
}

// Stats returns emitter statistics.
func (e *TelemetryEmitterImpl) Stats() EmitterStats {
	return EmitterStats{
		UsageEmitted:  atomic.LoadInt64(&e.stats.UsageEmitted),
		UsageDropped:  atomic.LoadInt64(&e.stats.UsageDropped),
		ErrorsEmitted: atomic.LoadInt64(&e.stats.ErrorsEmitted),
		ErrorsDropped: atomic.LoadInt64(&e.stats.ErrorsDropped),
		FlushCount:    atomic.LoadInt64(&e.stats.FlushCount),
		LastFlushTime: atomic.LoadInt64(&e.stats.LastFlushTime),
	}
}

// periodicFlusher runs a function periodically.
type periodicFlusher struct {
	period time.Duration
	stopCh chan struct{}
	wg     sync.WaitGroup
	fn     func()
}

func newPeriodicFlusher(period time.Duration) *periodicFlusher {
	return &periodicFlusher{
		period: period,
		stopCh: make(chan struct{}),
	}
}

func (p *periodicFlusher) Start(fn func()) {
	p.fn = fn
	p.wg.Add(1)
	go p.run()
}

func (p *periodicFlusher) run() {
	defer p.wg.Done()
	ticker := time.NewTicker(p.period)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if p.fn != nil {
				p.fn()
			}
		case <-p.stopCh:
			return
		}
	}
}

func (p *periodicFlusher) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

// NullTelemetryEmitter is a no-op emitter for testing or when telemetry is disabled.
type NullTelemetryEmitter struct{}

func (NullTelemetryEmitter) EmitUsage(context.Context, UsageRecord)   {}
func (NullTelemetryEmitter) EmitError(context.Context, ErrorRecord)   {}
func (NullTelemetryEmitter) Flush(context.Context) error              { return nil }
func (NullTelemetryEmitter) Close() error                             { return nil }