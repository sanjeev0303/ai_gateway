// Package gatewayv2 contract tests for provider usage capture: parsing native
// payloads, preferring reported counts over estimates, and persisting them
// idempotently to the ledgers.
package gatewayv2

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"

	"github.com/google/uuid"
)

// ---- Fakes -------------------------------------------------------------------

type recordingBus struct {
	mu       sync.Mutex
	messages []ports.UsageEventMessage
	err      error
}

func newRecordingBus() *recordingBus { return &recordingBus{} }

func (b *recordingBus) Publish(_ context.Context, msg ports.UsageEventMessage) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return b.err
	}
	b.messages = append(b.messages, msg)
	return nil
}

func (b *recordingBus) Run(context.Context, func(context.Context, ports.UsageEventMessage) error) error {
	return nil
}

func (b *recordingBus) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.messages)
}

func (b *recordingBus) last() ports.UsageEventMessage {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.messages[len(b.messages)-1]
}

// ---- Parsing -----------------------------------------------------------------

func TestUsageObserver_AnthropicSSE(t *testing.T) {
	o := newUsageObserver("text/event-stream")
	o.feed([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":1200,\"cache_read_input_tokens\":300,\"cache_creation_input_tokens\":50}}}\n\n"))
	o.feed([]byte("event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":450}}\n\n"))
	got := o.finish()

	requireTrue(t, got != nil, "anthropic stream should report usage")
	requireTrue(t, got.Provider == "anthropic", "provider mismatch: %s", got.Provider)
	requireTrue(t, got.InputTokens == 1200, "input mismatch: %d", got.InputTokens)
	requireTrue(t, got.OutputTokens == 450, "output mismatch: %d", got.OutputTokens)
	requireTrue(t, got.CacheReadTokens == 300, "cache read mismatch: %d", got.CacheReadTokens)
	requireTrue(t, got.CacheWriteTokens == 50, "cache write mismatch: %d", got.CacheWriteTokens)
}

func TestUsageObserver_OpenAIJSON(t *testing.T) {
	o := newUsageObserver("application/json")
	o.feed([]byte(`{"usage":{"prompt_tokens":800,"completion_tokens":200,"prompt_tokens_details":{"cached_tokens":128}}}`))
	got := o.finish()

	requireTrue(t, got != nil, "openai response should report usage")
	requireTrue(t, got.Provider == "openai", "provider mismatch: %s", got.Provider)
	requireTrue(t, got.InputTokens == 800, "input mismatch: %d", got.InputTokens)
	requireTrue(t, got.OutputTokens == 200, "output mismatch: %d", got.OutputTokens)
	requireTrue(t, got.CacheReadTokens == 128, "cache read mismatch: %d", got.CacheReadTokens)
}

func TestUsageObserver_GeminiJSON(t *testing.T) {
	o := newUsageObserver("application/json")
	o.feed([]byte(`{"usageMetadata":{"promptTokenCount":640,"candidatesTokenCount":160,"cachedContentTokenCount":64}}`))
	got := o.finish()

	requireTrue(t, got != nil, "gemini response should report usage")
	requireTrue(t, got.Provider == "gemini", "provider mismatch: %s", got.Provider)
	requireTrue(t, got.InputTokens == 640, "input mismatch: %d", got.InputTokens)
	requireTrue(t, got.OutputTokens == 160, "output mismatch: %d", got.OutputTokens)
	requireTrue(t, got.CacheReadTokens == 64, "cache read mismatch: %d", got.CacheReadTokens)
}

func TestUsageObserver_NoUsageIsNil(t *testing.T) {
	o := newUsageObserver("application/json")
	o.feed([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	requireTrue(t, o.finish() == nil, "payload without usage must report nil")
}

func TestUsageObserver_MalformedDoesNotPanic(t *testing.T) {
	o := newUsageObserver("text/event-stream")
	o.feed([]byte("data: {not json\n\n"))
	o.feed([]byte("data: \n\n"))
	o.feed([]byte("\x00\x01binary"))
	_ = o.finish()
}

func TestUsageObserver_SplitAcrossChunks(t *testing.T) {
	body := `{"usage":{"prompt_tokens":42,"completion_tokens":7}}`
	o := newUsageObserver("application/json")
	o.feed([]byte(body[:10]))
	o.feed([]byte(body[10:]))
	got := o.finish()
	requireTrue(t, got != nil && got.InputTokens == 42 && got.OutputTokens == 7, "chunk split lost usage")
}

// ---- Telemetry precedence ----------------------------------------------------

func TestEmitTelemetry_PrefersReportedOverEstimates(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	g.emitTelemetry(context.Background(), "req-1", res, time.Now(), &StreamUsage{
		Provider:         "anthropic",
		InputTokens:      1200,
		OutputTokens:     450,
		CacheReadTokens:  300,
		CacheWriteTokens: 50,
	})

	requireTrue(t, len(stubs.telemetry.usage) == 1, "want one usage record")
	u := stubs.telemetry.usage[0]
	requireTrue(t, u.RequestID == "req-1", "request id must be carried for idempotency")
	requireTrue(t, u.InputTokens == 1200 && u.OutputTokens == 450, "reported counts must win over estimates")
	requireTrue(t, u.CacheReadTokens == 300 && u.CacheWriteTokens == 50, "cache tokens must be carried")
	requireTrue(t, u.ProviderMeta != nil && u.ProviderMeta["provider"] == "anthropic", "provider must be recorded")
}

func TestEmitTelemetry_FallsBackToEstimates(t *testing.T) {
	g, stubs := newGatewayForPipeline()
	res := g.executePipeline(context.Background(), pipelineRequest("key-1", "{}"), "req-1")

	g.emitTelemetry(context.Background(), "req-1", res, time.Now(), nil)

	u := stubs.telemetry.usage[0]
	requireTrue(t, u.InputTokens == 1000 && u.OutputTokens == 2000, "estimates must be the fallback")
}

// ---- Durable sink ------------------------------------------------------------

func newSinkFixture() (*TelemetryEmitterImpl, *recordingBus) {
	bus := newRecordingBus()
	return NewTelemetryEmitter(8, time.Hour, bus, discardLogger()), bus
}

func TestSink_PublishesRequestAndTokenUsage(t *testing.T) {
	e, bus := newSinkFixture()
	orgID := uuid.New()
	credID := uuid.New()

	e.writeUsage(context.Background(), UsageRecord{
		RequestID:       "req-1",
		OrganizationID:  orgID,
		CredentialID:    credID,
		Model:           "claude-sonnet-4-5",
		InputTokens:     1200,
		OutputTokens:    450,
		CacheReadTokens: 300,
		StatusCode:      200,
		LatencyMs:       42,
		ProviderMeta:    map[string]any{"provider": "anthropic"},
	})

	requireTrue(t, bus.count() == 1, "want one published message, got %d", bus.count())

	msg := bus.last()
	requireTrue(t, msg.OrganizationID == orgID, "org scoping lost")
	requireTrue(t, msg.Request.RequestID == "req-1", "request id mismatch")
	requireTrue(t, msg.Usage.RequestID == "req-1", "usage event must key the same request")

	req := msg.Request
	requireTrue(t, req.ProviderKind == "anthropic", "provider kind mismatch: %s", req.ProviderKind)
	requireTrue(t, req.Status == "success", "status mismatch: %s", req.Status)
	requireTrue(t, req.PromptTokens == 1200 && req.CompletionTokens == 450, "token mapping mismatch")
	requireTrue(t, req.TotalTokens == 1650, "total token mismatch: %d", req.TotalTokens)
	requireTrue(t, req.CachedInputTokens == 300, "cached token mismatch")
	requireTrue(t, req.GatewayCredentialID != nil && *req.GatewayCredentialID == credID, "credential id must be recorded")

	ev := msg.Usage
	requireTrue(t, ev.EventType == domain.UsageEventTypeTokenUsage, "event type mismatch: %s", ev.EventType)
	requireTrue(t, ev.Unit == domain.UsageUnitTokens, "unit mismatch: %s", ev.Unit)
	requireTrue(t, ev.Quantity == 1650, "quantity must be total tokens: %d", ev.Quantity)

	var meta usageEventMetadata
	requireNoErr(t, json.Unmarshal(ev.Metadata, &meta), "metadata must be valid json")
	requireTrue(t, meta.InputTokens == 1200 && meta.OutputTokens == 450, "metadata breakdown mismatch")
	requireTrue(t, meta.Provider == "anthropic" && meta.UsageSource == "provider_reported", "metadata provenance mismatch")
}

func TestSink_MarksErrorStatus(t *testing.T) {
	e, bus := newSinkFixture()
	e.writeUsage(context.Background(), UsageRecord{
		RequestID: "req-1", OrganizationID: uuid.New(),
		InputTokens: 10, OutputTokens: 0, StatusCode: 500,
	})
	requireTrue(t, bus.last().Request.Status == "error", "5xx must record error status")
	requireTrue(t, bus.count() == 1, "usage must still be billed on upstream error")
}

func TestSink_RejectsRecordWithoutRequestID(t *testing.T) {
	e, bus := newSinkFixture()
	e.flushUsage([]UsageRecord{{OrganizationID: uuid.New(), InputTokens: 10}})
	requireTrue(t, e.Stats().UsageWriteFailed == 1, "unkeyed record must fail")
	requireTrue(t, bus.count() == 0, "nothing may be published without a request id")
}

func TestSink_PropagatesPublishFailure(t *testing.T) {
	e, bus := newSinkFixture()
	bus.err = context.DeadlineExceeded
	e.flushUsage([]UsageRecord{{RequestID: "req-1", OrganizationID: uuid.New(), InputTokens: 10}})
	requireTrue(t, e.Stats().UsageWriteFailed == 1, "publish failure must be counted, never silent")
}

func TestFlushUsage_WithoutSinkCountsFailure(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, nil, discardLogger())
	e.flushUsage([]UsageRecord{
		{RequestID: "req-1", OrganizationID: uuid.New(), InputTokens: 10},
		{RequestID: "req-2", OrganizationID: uuid.New(), InputTokens: 20},
	})
	requireTrue(t, e.Stats().UsageWriteFailed == 2, "unconfigured sink must surface as failures")
	requireNoErr(t, e.Close(), "Close")
}

func TestEmitter_WritesThroughQueueAndClose(t *testing.T) {
	e, bus := newSinkFixture()
	e.EmitUsage(context.Background(), UsageRecord{
		RequestID: "req-1", OrganizationID: uuid.New(),
		Model: "claude-sonnet-4-5", InputTokens: 100, OutputTokens: 20,
	})
	requireNoErr(t, e.Close(), "Close must drain queued records")
	requireTrue(t, bus.count() == 1, "queued record must survive shutdown, got %d", bus.count())
}

// Flush used to be a no-op that also returned nil, so asserting the return
// value proves nothing. This pins the behaviour that matters: by the time Flush
// returns, the record has already been handed to the bus.
func TestEmitter_FlushPublishesBeforeReturning(t *testing.T) {
	e, bus := newSinkFixture()
	defer func() { requireNoErr(t, e.Close(), "Close") }()

	e.EmitUsage(context.Background(), UsageRecord{
		RequestID: "req-2", OrganizationID: uuid.New(),
		Model: "claude-sonnet-4-5", InputTokens: 100, OutputTokens: 20,
	})

	requireNoErr(t, e.Flush(context.Background()), "Flush")
	requireTrue(t, bus.count() == 1, "Flush must publish before returning, got %d", bus.count())
	requireTrue(t, e.Stats().FlushCount >= 1, "Flush must count the flush")
}

func TestEmitter_FlushRespectsContextCancellation(t *testing.T) {
	e := NewTelemetryEmitter(8, time.Hour, newRecordingBus(), discardLogger())
	requireNoErr(t, e.Close(), "Close")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := e.Flush(ctx); err == nil {
		t.Fatal("Flush on a closed emitter must report an error, not silently succeed")
	}
}
