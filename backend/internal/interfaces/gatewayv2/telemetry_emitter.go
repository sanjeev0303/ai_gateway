// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
)

// usageFlushTimeout bounds one batch of ledger writes so a stuck database
// cannot pin the worker goroutine indefinitely.
const usageFlushTimeout = 5 * time.Second

var errMissingRequestID = errors.New("gateway: usage record has no request id")

// TelemetryEmitterImpl emits usage and error events asynchronously.
type TelemetryEmitterImpl struct {
	usageQueue  chan UsageRecord
	errorQueue  chan ErrorRecord
	logger      *slog.Logger
	wg          sync.WaitGroup
	stopCh      chan struct{}
	flushReq    chan chan struct{}
	bufferSize  int
	flushPeriod time.Duration
	bus         ports.EventBus
	stats       EmitterStats
}

// EmitterStats tracks emitter performance.
type EmitterStats struct {
	UsageEmitted     int64
	UsageDropped     int64
	UsageWriteFailed int64
	ErrorsEmitted    int64
	ErrorsDropped    int64
	FlushCount       int64
	LastFlushTime    int64 // unix nano
}

// NewTelemetryEmitter creates a telemetry emitter that publishes usage onto the
// event bus off the request path. The data plane derives the records; the worker
// persists them.
func NewTelemetryEmitter(bufferSize int, flushPeriod time.Duration, bus ports.EventBus, logger *slog.Logger) *TelemetryEmitterImpl {
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
		usageQueue:  make(chan UsageRecord, bufferSize),
		errorQueue:  make(chan ErrorRecord, bufferSize),
		logger:      logger,
		stopCh:      make(chan struct{}),
		flushReq:    make(chan chan struct{}),
		bufferSize:  bufferSize,
		flushPeriod: flushPeriod,
		bus:         bus,
	}

	e.wg.Add(2)
	go e.usageWorker()
	go e.errorWorker()

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
			"request_id", usage.RequestID,
			"org_id", usage.OrganizationID,
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

	// select picks a ready case at random, so a flush or a stop can be
	// serviced while records are still queued. Pull in everything buffered
	// first, otherwise the flush can ack an empty batch.
	drain := func() {
		for {
			select {
			case record := <-e.usageQueue:
				batch = append(batch, record)
			default:
				return
			}
		}
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
		case done := <-e.flushReq:
			drain()
			flush()
			close(done)
		case <-e.stopCh:
			drain()
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
			for {
				select {
				case record := <-e.errorQueue:
					batch = append(batch, record)
				default:
					flush()
					return
				}
			}
		}
	}
}

// flushUsage publishes a batch of usage records. The worker owns the ledger
// write, and its idempotency absorbs a redelivered message rather than
// double-counting it.
func (e *TelemetryEmitterImpl) flushUsage(batch []UsageRecord) {
	if e.bus == nil {
		atomic.AddInt64(&e.stats.UsageWriteFailed, int64(len(batch)))
		e.logger.Error("telemetry: usage sink not configured, discarding records", "count", len(batch))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), usageFlushTimeout)
	defer cancel()

	for _, record := range batch {
		if err := e.writeUsage(ctx, record); err != nil {
			atomic.AddInt64(&e.stats.UsageWriteFailed, 1)
			e.logger.Error("telemetry: usage write failed",
				"request_id", record.RequestID,
				"org_id", record.OrganizationID,
				"error", err,
			)
		}
	}
}

// usageEventMetadata is the ledger detail carried alongside the token total.
type usageEventMetadata struct {
	InputTokens      int    `json:"input_tokens"`
	OutputTokens     int    `json:"output_tokens"`
	CacheReadTokens  int    `json:"cache_read_tokens"`
	CacheWriteTokens int    `json:"cache_write_tokens"`
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	CredentialID     string `json:"credential_id,omitempty"`
	StatusCode       int    `json:"status_code"`
	LatencyMs        int64  `json:"latency_ms"`
	UsageSource      string `json:"usage_source"`
}

func (e *TelemetryEmitterImpl) writeUsage(ctx context.Context, record UsageRecord) error {
	if record.RequestID == "" {
		return errMissingRequestID
	}

	providerKind := "unknown"
	if record.ProviderMeta != nil {
		if name, ok := record.ProviderMeta["provider"].(string); ok && name != "" {
			providerKind = name
		}
	}
	status := "success"
	if record.StatusCode >= 400 {
		status = "error"
	}
	source := "estimated"
	if providerKind != "unknown" {
		source = "provider_reported"
	}

	metadata, err := json.Marshal(usageEventMetadata{
		InputTokens:      record.InputTokens,
		OutputTokens:     record.OutputTokens,
		CacheReadTokens:  record.CacheReadTokens,
		CacheWriteTokens: record.CacheWriteTokens,
		Provider:         providerKind,
		Model:            record.Model,
		CredentialID:     record.CredentialID.String(),
		StatusCode:       record.StatusCode,
		LatencyMs:        record.LatencyMs,
		UsageSource:      source,
	})
	if err != nil {
		return err
	}

	return e.bus.Publish(ctx, ports.UsageEventMessage{
		OrganizationID: record.OrganizationID,
		Request:        newLlmRequestFrom(record, providerKind, status),
		Usage: domain.NewUsageEvent{
			RequestID: record.RequestID,
			EventType: domain.UsageEventTypeTokenUsage,
			Quantity:  int64(record.InputTokens) + int64(record.OutputTokens),
			Unit:      domain.UsageUnitTokens,
			Metadata:  metadata,
		},
	})
}

func newLlmRequestFrom(record UsageRecord, providerKind, status string) domain.NewLlmRequest {
	return domain.NewLlmRequest{
		ProjectID:          record.ProjectID,
		UserID:             subjectUserID(record),
		ServiceAccountID:    subjectServiceAccountID(record),
		TeamID:              record.TeamID,
		GatewayCredentialID: &record.CredentialID,
		RequestID:          record.RequestID,
		ProviderKind:       providerKind,
		Model:              record.Model,
		Status:             status,
		PromptTokens:       int32(record.InputTokens),
		CompletionTokens:   int32(record.OutputTokens),
		CachedInputTokens:  int32(record.CacheReadTokens),
		TotalTokens:        int32(record.InputTokens + record.OutputTokens),
		LatencyMs:          int32(record.LatencyMs),
		CreatedAt:          record.Timestamp.UTC(),
		RequestStartedAt:   &record.RequestStartedAt,
		FirstTokenAt:       record.FirstTokenAt,
		CompletedAt:        &record.CompletedAt,
		TimeToFirstTokenMs: record.TimeToFirstTokenMs,
		TotalLatencyMs:     record.TotalLatencyMs,
	}
}

func subjectUserID(record UsageRecord) *uuid.UUID {
	if domain.SubjectType(record.SubjectType) != domain.SubjectTypeUser {
		return nil
	}
	return &record.SubjectID
}

func subjectServiceAccountID(record UsageRecord) *uuid.UUID {
	if domain.SubjectType(record.SubjectType) != domain.SubjectTypeServiceAccount {
		return nil
	}
	return &record.SubjectID
}

// flushErrors sends a batch of error records to the backend.
func (e *TelemetryEmitterImpl) flushErrors(batch []ErrorRecord) {
	for _, record := range batch {
		e.logger.Warn("telemetry: error", "org_id", record.OrganizationID, "error_type", record.ErrorType, "status", record.StatusCode)
	}
}

// Flush blocks until the usage worker has persisted everything queued so far.
// The worker owns the batch, so the request has to be serviced by it rather
// than raced for from here.
func (e *TelemetryEmitterImpl) Flush(ctx context.Context) error {
	done := make(chan struct{})
	select {
	case e.flushReq <- done:
	case <-ctx.Done():
		return ctx.Err()
	case <-e.stopCh:
		return errors.New("gateway: telemetry emitter closed")
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the emitter and drains queues.
func (e *TelemetryEmitterImpl) Close() error {
	close(e.stopCh)
	e.wg.Wait()
	return nil
}

// Stats returns emitter statistics.
func (e *TelemetryEmitterImpl) Stats() EmitterStats {
	return EmitterStats{
		UsageEmitted:     atomic.LoadInt64(&e.stats.UsageEmitted),
		UsageDropped:     atomic.LoadInt64(&e.stats.UsageDropped),
		UsageWriteFailed: atomic.LoadInt64(&e.stats.UsageWriteFailed),
		ErrorsEmitted:    atomic.LoadInt64(&e.stats.ErrorsEmitted),
		ErrorsDropped:    atomic.LoadInt64(&e.stats.ErrorsDropped),
		FlushCount:       atomic.LoadInt64(&e.stats.FlushCount),
		LastFlushTime:    atomic.LoadInt64(&e.stats.LastFlushTime),
	}
}

// NullTelemetryEmitter is a no-op emitter for testing or when telemetry is disabled.
type NullTelemetryEmitter struct{}

func (NullTelemetryEmitter) EmitUsage(context.Context, UsageRecord) {}
func (NullTelemetryEmitter) EmitError(context.Context, ErrorRecord) {}
func (NullTelemetryEmitter) Flush(context.Context) error            { return nil }
func (NullTelemetryEmitter) Close() error                           { return nil }
