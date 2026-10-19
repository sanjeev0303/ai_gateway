// Package anthropic implements the Anthropic provider adapter.
package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestValidateCredentials tests credential validation.
func TestValidateCredentials(t *testing.T) {
	// Test with invalid key
	adapter := NewProviderAdapter()
	err := adapter.ValidateCredentials(context.Background(), "invalid-key")
	if err == nil {
		t.Error("expected error for invalid API key")
	}

	// Test with valid-looking key format (will fail at network level)
	// This tests the validation logic without hitting the real API
	adapter2 := NewProviderAdapter()
	err = adapter2.ValidateCredentials(context.Background(), "sk-ant-test123")
	// Should not return ErrInvalidAPIKey immediately (that's for 401/403)
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "anthropic:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestHealthCheck tests the health check functionality.
func TestHealthCheck(t *testing.T) {
	adapter := NewProviderAdapter()
	err := adapter.HealthCheck(context.Background(), "test-key")
	// Health check should succeed on any HTTP response (even 401)
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "anthropic:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestListModels tests listing models.
func TestListModels(t *testing.T) {
	adapter := NewProviderAdapter()
	_, err := adapter.ListModels(context.Background(), "test-key")
	// Network error expected
	if err != nil && !strings.Contains(err.Error(), "anthropic:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestUsageCollector tests the usage collector.
func TestUsageCollector(t *testing.T) {
	collector := NewUsageCollector()

	// Test initial state
	if collector.IsComplete() {
		t.Error("collector should not be complete initially")
	}

	// Test ProcessEvent with message_start
	event := &StreamingEvent{
		Type: "message_start",
		Message: &MessageResponse{
			Usage: Usage{
				InputTokens:              100,
				OutputTokens:             50,
				CacheCreationInputTokens: 10,
				CacheReadInputTokens:     5,
			},
		},
	}
	err := collector.ProcessEvent(event)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	usage := collector.FinalUsage()
	if usage.InputTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected output tokens 50, got %d", usage.OutputTokens)
	}
	if usage.CacheCreationInputTokens != 10 {
		t.Errorf("expected cache creation tokens 10, got %d", usage.CacheCreationInputTokens)
	}
	if usage.CacheReadInputTokens != 5 {
		t.Errorf("expected cache read tokens 5, got %d", usage.CacheReadInputTokens)
	}

	// Test ProcessEvent with message_delta
	deltaEvent := &StreamingEvent{
		Type: "message_delta",
		Usage: &Usage{
			OutputTokens: 25,
		},
	}
	err = collector.ProcessEvent(deltaEvent)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	usage = collector.FinalUsage()
	if usage.OutputTokens != 75 { // 50 + 25
		t.Errorf("expected output tokens 75, got %d", usage.OutputTokens)
	}

	// Test ProcessEvent with message_stop
	stopEvent := &StreamingEvent{
		Type: "message_stop",
	}
	err = collector.ProcessEvent(stopEvent)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	if !collector.IsComplete() {
		t.Error("collector should be complete after message_stop")
	}
}

// TestSSEDecoder tests the SSE decoder.
func TestSSEDecoder(t *testing.T) {
	// Test basic event decoding
	sseData := `event: message_start
data: {"type":"message_start","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus-20240229","content":[],"usage":{"input_tokens":10,"output_tokens":5}}}

`

	decoder := NewSSEDecoder(strings.NewReader(sseData))
	event, err := decoder.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event.Type != "message_start" {
		t.Errorf("expected event type 'message_start', got %q", event.Type)
	}
	if event.Message == nil {
		t.Error("expected message in event")
	} else if event.Message.Usage.InputTokens != 10 {
		t.Errorf("expected input tokens 10, got %d", event.Message.Usage.InputTokens)
	}

	// Test message_delta event
	deltaData := `event: message_delta
data: {"type":"message_delta","delta":{"type":"text_delta","text":"Hello"},"usage":{"output_tokens":5}}

`
	decoder2 := NewSSEDecoder(strings.NewReader(deltaData))
	event2, err := decoder2.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event2.Type != "message_delta" {
		t.Errorf("expected event type 'message_delta', got %q", event2.Type)
	}
	if event2.Usage == nil || event2.Usage.OutputTokens != 5 {
		t.Errorf("expected output tokens 5 in delta, got %v", event2.Usage)
	}

	// Test message_stop event
	stopData := `event: message_stop
data: {"type":"message_stop"}

`
	decoder3 := NewSSEDecoder(strings.NewReader(stopData))
	event3, err := decoder3.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event3.Type != "message_stop" {
		t.Errorf("expected event type 'message_stop', got %q", event3.Type)
	}

	// Test EOF after message_stop
	_, err = decoder3.Next()
	if err != io.EOF {
		t.Errorf("expected EOF after message_stop, got %v", err)
	}
}

// TestStreamingProxy tests the streaming proxy with a mock server.
func TestStreamingProxy(t *testing.T) {
	// Create a mock server that simulates Anthropic SSE response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-version") == "" {
			http.Error(w, "missing version", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send message_start
		w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_123\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-3-opus-20240229\",\"content\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n"))
		flusher.Flush()

		// Send content_block_start
		w.Write([]byte("event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"))
		flusher.Flush()

		// Send content_block_delta
		w.Write([]byte("event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n"))
		flusher.Flush()

		// Send content_block_stop
		w.Write([]byte("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"))
		flusher.Flush()

		// Send message_delta with usage
		w.Write([]byte("event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"},\"usage\":{\"output_tokens\":11}}\n\n"))
		flusher.Flush()

		// Send message_stop
		w.Write([]byte("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
		flusher.Flush()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Test the streaming proxy
	proxy := NewStreamingProxy(server.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-opus-20240229","messages":[{"role":"user","content":"Hello"}],"max_tokens":100,"stream":true}`))
	req.Header.Set("x-api-key", "test-key")

	w := httptest.NewRecorder()
	ctx := context.Background()

	err := proxy.Forward(ctx, w, req, "test-api-key", "cred-123")
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that response is SSE format
	body := w.Body.String()
	if !strings.Contains(body, "event: message_start") {
		t.Errorf("expected message_start event in response")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Errorf("expected message_stop event in response")
	}
}

// TestNonStreamingProxy tests the non-streaming proxy.
func TestNonStreamingProxy(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus-20240229","content":[{"type":"text","text":"Hello!"}],"usage":{"input_tokens":10,"output_tokens":5}}`))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	proxy := NewStreamingProxy(server.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-3-opus-20240229","messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`))
	req.Header.Set("x-api-key", "test-key")

	w := httptest.NewRecorder()
	ctx := context.Background()

	err := proxy.Forward(ctx, w, req, "test-api-key", "cred-123")
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Hello!") {
		t.Errorf("expected 'Hello!' in response body")
	}
	if !strings.Contains(body, `"input_tokens":10`) {
		t.Errorf("expected input_tokens in response")
	}
}

// TestProviderAdapterIntegration tests the provider adapter integration.
func TestProviderAdapterIntegration(t *testing.T) {
	adapter := NewProviderAdapter()

	// Test that we can create a client
	client, err := New(adapter.baseURL, adapter.version, "test-agent")
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}

	// Verify the client has the expected methods
	_ = client.ValidateCredentials
	_ = client.HealthCheck
	_ = client.ListModels
	_ = client.CreateMessage
	_ = client.CreateMessageStream
	_ = client.CountTokens
}

// TestListModelsMetadata asserts ListModels surfaces the provider-advertised
// display name alongside each model ID.
func TestListModelsMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-5","display_name":"Claude Sonnet 4.5","type":"model","created_at":"2025-06-01T00:00:00Z"}]}`))
	}))
	defer server.Close()

	adapter := NewProviderAdapter()
	adapter.baseURL = server.URL

	models, err := adapter.ListModels(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	m := models[0]
	if m.ID != "claude-sonnet-4-5" {
		t.Errorf("unexpected model ID: %q", m.ID)
	}
	if m.Provider != "anthropic" {
		t.Errorf("unexpected provider: %q", m.Provider)
	}
	if m.DisplayName != "Claude Sonnet 4.5" {
		t.Errorf("unexpected display name: %q", m.DisplayName)
	}
}

// TestAnthropic_LiveSmokeTest executes a live smoke test against Anthropic only if ANTHROPIC_API_KEY is configured.
func TestAnthropic_LiveSmokeTest(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("skipping live Anthropic smoke test: ANTHROPIC_API_KEY not set")
	}
	adapter := NewProviderAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := adapter.ValidateCredentials(ctx, apiKey); err != nil {
		t.Fatalf("live Anthropic credential validation failed: %v", err)
	}
	models, err := adapter.ListModels(ctx, apiKey)
	if err != nil {
		t.Fatalf("live Anthropic ListModels failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatalf("live Anthropic ListModels returned 0 models")
	}
}
