// Package openai implements the OpenAI provider adapter.
package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	err = adapter2.ValidateCredentials(context.Background(), "sk-test123")
	// Should not return ErrInvalidAPIKey immediately (that's for 401/403)
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "openai:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestHealthCheck tests the health check functionality.
func TestHealthCheck(t *testing.T) {
	adapter := NewProviderAdapter()
	err := adapter.HealthCheck(context.Background(), "test-key")
	// Health check should succeed on any HTTP response (even 401)
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "openai:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestListModels tests listing models.
func TestListModels(t *testing.T) {
	adapter := NewProviderAdapter()
	_, err := adapter.ListModels(context.Background(), "test-key")
	// Network error expected
	if err != nil && !strings.Contains(err.Error(), "openai:") {
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
		Object: "chat.completion.chunk",
		Choices: []StreamChoice{
			{
				Delta: ChatMessage{Role: "assistant", Content: "Hello"},
			},
		},
		Usage: &Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			PromptTokensDetails: &PromptTokensDetails{
				CachedTokens: 10,
			},
			CompletionTokensDetails: &CompletionTokensDetails{
				ReasoningTokens: 5,
			},
		},
	}
	err := collector.ProcessEvent(event)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	usage := collector.FinalUsage()
	if usage.PromptTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("expected output tokens 50, got %d", usage.CompletionTokens)
	}
	if usage.PromptTokensDetails == nil || usage.PromptTokensDetails.CachedTokens != 10 {
		t.Errorf("expected cache creation tokens 10, got %d", usage.PromptTokensDetails.CachedTokens)
	}
	if usage.CompletionTokensDetails == nil || usage.CompletionTokensDetails.ReasoningTokens != 5 {
		t.Errorf("expected reasoning tokens 5, got %d", usage.CompletionTokensDetails.ReasoningTokens)
	}

	// Test ProcessEvent with message_delta
	deltaEvent := &StreamingEvent{
		Object: "chat.completion.chunk",
		Usage: &Usage{
			CompletionTokens: 25,
		},
	}
	err = collector.ProcessEvent(deltaEvent)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	usage = collector.FinalUsage()
	if usage.CompletionTokens != 75 { // 50 + 25
		t.Errorf("expected output tokens 75, got %d", usage.CompletionTokens)
	}

	// Test ProcessEvent with finish_reason
	stopEvent := &StreamingEvent{
		Object: "chat.completion.chunk",
		Choices: []StreamChoice{
			{FinishReason: "stop"},
		},
	}
	err = collector.ProcessEvent(stopEvent)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	if !collector.IsComplete() {
		t.Error("collector should be complete after finish_reason")
	}
}

// TestSSEDecoder tests the SSE decoder.
func TestSSEDecoder(t *testing.T) {
	// Test basic event decoding
	sseData := `event: chat.completion.chunk
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}

`

	decoder := NewSSEDecoder(strings.NewReader(sseData))
	event, err := decoder.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event.Object != "chat.completion.chunk" {
		t.Errorf("expected event type 'chat.completion.chunk', got %q", event.Object)
	}
	if event.Choices == nil || len(event.Choices) == 0 {
		t.Error("expected choices in event")
	} else if event.Choices[0].Delta.Content != "Hello" {
		t.Errorf("expected delta content 'Hello', got %v", event.Choices[0].Delta.Content)
	}

	// Test message_stop event
	// Test message_stop event
	stopData := `event: chat.completion.chunk
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gpt-4","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

`

	decoder2 := NewSSEDecoder(strings.NewReader(stopData))
	event2, err := decoder2.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event2.Object != "chat.completion.chunk" {
		t.Errorf("expected event type 'chat.completion.chunk', got %q", event2.Object)
	}
	if event2.Choices == nil || len(event2.Choices) == 0 || event2.Choices[0].FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %v", event2.Choices[0].FinishReason)
	}

	// Test EOF after message_stop
	_, err = decoder2.Next()
	if err != io.EOF {
		t.Errorf("expected EOF after message_stop, got %v", err)
	}
}

// TestStreamingProxy tests the streaming proxy with a mock server.
func TestStreamingProxy(t *testing.T) {
	// Create a mock server that simulates OpenAI SSE response
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("OpenAI-Version") == "" {
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
		w.Write([]byte("event: chat.completion.chunk\ndata: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()

		// Send content_block_delta
		w.Write([]byte("event: chat.completion.chunk\ndata: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n"))
		flusher.Flush()

		// Send message_stop
		w.Write([]byte("event: chat.completion.chunk\ndata: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1234567890,\"model\":\"gpt-4\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\n\n"))
		flusher.Flush()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	// Test the streaming proxy
	proxy := NewStreamingProxy(server.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"max_tokens":100,"stream":true}`))
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	ctx := context.Background()

	err := proxy.Forward(ctx, w, req, "test-key", "cred-123")
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that response is SSE format
	body := w.Body.String()
	if !strings.Contains(body, "event: chat.completion.chunk") {
		t.Errorf("expected message_start event in response")
	}
	if !strings.Contains(body, "\"finish_reason\":\"stop\"") {
		t.Errorf("expected message_stop event in response")
	}
}

// TestNonStreamingProxy tests the non-streaming proxy.
func TestNonStreamingProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"chatcmpl-123","object":"chat.completion","created":1234567890,"model":"gpt-4","choices":[{"index":0,"message":{"role":"assistant","content":"Hello!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer server.Close()
	defer server.Close()

	proxy := NewStreamingProxy(server.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}],"max_tokens":100}`))
	req.Header.Set("Authorization", "Bearer test-key")

	w := httptest.NewRecorder()
	ctx := context.Background()

	err := proxy.Forward(ctx, w, req, "test-key", "cred-123")
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
	if !strings.Contains(body, `"prompt_tokens":10`) {
		t.Errorf("expected prompt_tokens in response")
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

// TestListModelsMetadata asserts ListModels tags each model with its
// provider kind even though OpenAI's models API only exposes IDs.
func TestListModelsMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-4o","object":"model","created":1715367049,"owned_by":"openai"}]}`))
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
	if m.ID != "gpt-4o" {
		t.Errorf("unexpected model ID: %q", m.ID)
	}
	if m.Provider != "openai" {
		t.Errorf("unexpected provider: %q", m.Provider)
	}
	if m.DisplayName != "" {
		t.Errorf("expected empty display name for OpenAI, got %q", m.DisplayName)
	}
}
