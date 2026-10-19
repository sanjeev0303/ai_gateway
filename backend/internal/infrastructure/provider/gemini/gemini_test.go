// Package gemini implements the Gemini provider adapter.
package gemini

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

	// Test with a valid-looking key (will fail at network level)
	adapter2 := NewProviderAdapter()
	err = adapter2.ValidateCredentials(context.Background(), "test-key")
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "gemini:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestHealthCheck tests the health check functionality.
func TestHealthCheck(t *testing.T) {
	adapter := NewProviderAdapter()
	err := adapter.HealthCheck(context.Background(), "test-key")
	// Network error is expected since we're not mocking
	if err != nil && !strings.Contains(err.Error(), "gemini:") {
		t.Errorf("unexpected error type: %v", err)
	}
}

// TestListModels tests listing models.
func TestListModels(t *testing.T) {
	adapter := NewProviderAdapter()
	_, err := adapter.ListModels(context.Background(), "test-key")
	// Network error expected
	if err != nil && !strings.Contains(err.Error(), "gemini:") {
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
	if collector.HasUsage() {
		t.Error("collector should not have usage initially")
	}

	// Test ProcessEvent with a chunk carrying usageMetadata
	event := &StreamingEvent{
		Candidates: []Candidate{
			{
				Content: &Content{
					Role:  "model",
					Parts: []Part{{Text: "Hello"}},
				},
				FinishReason: "STOP",
			},
		},
		UsageMetadata: &UsageMetadata{
			PromptTokenCount:        100,
			CandidatesTokenCount:    50,
			TotalTokenCount:         150,
			CachedContentTokenCount: 10,
			ThoughtsTokenCount:      5,
		},
	}
	err := collector.ProcessEvent(event)
	if err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}

	if !collector.HasUsage() {
		t.Error("expected usage to be captured")
	}
	if !collector.IsComplete() {
		t.Error("collector should be complete after usage chunk")
	}

	usage := collector.FinalUsage()
	if usage.PromptTokens != 100 {
		t.Errorf("expected prompt tokens 100, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 50 {
		t.Errorf("expected completion tokens 50, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("expected total tokens 150, got %d", usage.TotalTokens)
	}
	if usage.CachedTokens != 10 {
		t.Errorf("expected cached tokens 10, got %d", usage.CachedTokens)
	}
	if usage.ThinkingTokens != 5 {
		t.Errorf("expected thinking tokens 5, got %d", usage.ThinkingTokens)
	}

	// After completion, further events are ignored
	later := &StreamingEvent{
		UsageMetadata: &UsageMetadata{
			PromptTokenCount:     999,
			CandidatesTokenCount: 999,
			TotalTokenCount:      1998,
		},
	}
	if err := collector.ProcessEvent(later); err != nil {
		t.Errorf("ProcessEvent failed: %v", err)
	}
	if usage := collector.FinalUsage(); usage.TotalTokens != 150 {
		t.Errorf("expected usage frozen at 150, got %d", usage.TotalTokens)
	}

	// Test Reset
	collector.Reset()
	if collector.HasUsage() {
		t.Error("collector should not have usage after reset")
	}
	if collector.IsComplete() {
		t.Error("collector should not be complete after reset")
	}
}

// TestSSEDecoder tests the SSE decoder.
func TestSSEDecoder(t *testing.T) {
	// Test basic event decoding
	sseData := `data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Hello"}],"index":0},"finishReason":"STOP"}],"modelVersion":"gemini-2.5-flash"}

`

	decoder := NewSSEDecoder(strings.NewReader(sseData))
	event, err := decoder.Next()
	if err != nil {
		t.Errorf("Next() failed: %v", err)
	}
	if event.Candidates == nil || len(event.Candidates) == 0 {
		t.Error("expected candidates in event")
	} else if event.Candidates[0].Content == nil || event.Candidates[0].Content.Parts[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %v", event.Candidates[0].Content)
	}

	// Test [DONE] marker returns EOF
	doneData := "data: [DONE]\n\n"
	decoder2 := NewSSEDecoder(strings.NewReader(doneData))
	_, err = decoder2.Next()
	if err != io.EOF {
		t.Errorf("expected EOF for [DONE] marker, got %v", err)
	}

	// Test empty input returns EOF
	decoder3 := NewSSEDecoder(strings.NewReader(""))
	_, err = decoder3.Next()
	if err != io.EOF {
		t.Errorf("expected EOF for empty input, got %v", err)
	}
}

// TestStreamingProxy tests the streaming proxy with a mock server.
func TestStreamingProxy(t *testing.T) {
	var gotAPIKey string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("x-goog-api-key")
		if gotAPIKey == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		if !strings.Contains(r.URL.Path, ":streamGenerateContent") {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		if !strings.Contains(r.URL.RawQuery, "alt=sse") {
			http.Error(w, "missing alt=sse", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send content chunk
		w.Write([]byte("data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hello\"}]}}]}\n\n"))
		flusher.Flush()

		// Send final chunk with usage metadata
		w.Write([]byte("data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\" world\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":10,\"candidatesTokenCount\":5,\"totalTokenCount\":15}}\n\n"))
		flusher.Flush()

		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	var capturedUsage *Usage
	var capturedCredID string
	proxy := NewStreamingProxy(server.URL, WithUsageCallback(func(usage *Usage, providerCredentialID string) {
		capturedUsage = usage
		capturedCredID = providerCredentialID
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`))
	req.Header.Set("x-goog-api-key", "test-key")

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
	if !strings.Contains(body, "usageMetadata") {
		t.Error("expected usageMetadata in relayed response")
	}
	if !strings.Contains(body, "totalTokenCount") {
		t.Error("expected totalTokenCount in relayed response")
	}

	// Verify usage callback fired
	if capturedUsage == nil {
		t.Fatal("expected usage callback to fire")
	}
	if capturedUsage.PromptTokens != 10 {
		t.Errorf("expected prompt tokens 10, got %d", capturedUsage.PromptTokens)
	}
	if capturedUsage.CompletionTokens != 5 {
		t.Errorf("expected completion tokens 5, got %d", capturedUsage.CompletionTokens)
	}
	if capturedCredID != "cred-123" {
		t.Errorf("expected credential id 'cred-123', got %q", capturedCredID)
	}
}

// TestStreamingProxyNonSSE tests the proxy when the upstream returns non-SSE.
func TestStreamingProxyNonSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") == "" {
			http.Error(w, "missing api key", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[{"content":{"role":"model","parts":[{"text":"Hello!"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5,"totalTokenCount":15}}`))
	}))
	defer server.Close()

	proxy := NewStreamingProxy(server.URL)

	req := httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-2.5-flash:generateContent", strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`))
	req.Header.Set("x-goog-api-key", "test-key")

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
	if !strings.Contains(body, `"promptTokenCount":10`) {
		t.Errorf("expected promptTokenCount in response")
	}
}

// TestStreamingProxyMissingModelPath tests the fallback when the request path
// does not carry a model.
func TestStreamingProxyMissingModelPath(t *testing.T) {
	var gotPath string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"candidates":[],"usageMetadata":{"promptTokenCount":0,"candidatesTokenCount":0,"totalTokenCount":0}}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	proxy := NewStreamingProxy(server.URL)

	// Path without a model — proxy must fall back to the model in the body.
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gemini-2.5-flash","contents":[{"role":"user","parts":[{"text":"Hello"}]}]}`))
	req.Header.Set("x-goog-api-key", "test-key")

	w := httptest.NewRecorder()
	ctx := context.Background()

	err := proxy.Forward(ctx, w, req, "test-key", "cred-123")
	if err != nil {
		t.Errorf("Forward failed: %v", err)
	}

	if !strings.Contains(gotPath, ":streamGenerateContent") {
		t.Errorf("expected streamGenerateContent in upstream path, got %q", gotPath)
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
// display name, capabilities, and context limit alongside each model ID.
func TestListModelsMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"models":[{"name":"models/gemini-2.5-flash","displayName":"Gemini 2.5 Flash","supportedGenerationMethods":["generateContent","countTokens"],"inputTokenLimit":1048576,"outputTokenLimit":8192}]}`))
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
	if m.ID != "gemini-2.5-flash" {
		t.Errorf("unexpected model ID: %q", m.ID)
	}
	if m.Provider != "gemini" {
		t.Errorf("unexpected provider: %q", m.Provider)
	}
	if m.DisplayName != "Gemini 2.5 Flash" {
		t.Errorf("unexpected display name: %q", m.DisplayName)
	}
	if len(m.Capabilities) != 2 || m.Capabilities[0] != "generateContent" {
		t.Errorf("unexpected capabilities: %v", m.Capabilities)
	}
	if m.ContextLimit != 1048576 {
		t.Errorf("unexpected context limit: %d", m.ContextLimit)
	}
}

// TestGemini_LiveSmokeTest executes a live smoke test against Gemini only if GEMINI_API_KEY is configured.
func TestGemini_LiveSmokeTest(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping live Gemini smoke test: GEMINI_API_KEY not set")
	}
	adapter := NewProviderAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := adapter.ValidateCredentials(ctx, apiKey); err != nil {
		t.Fatalf("live Gemini credential validation failed: %v", err)
	}
	models, err := adapter.ListModels(ctx, apiKey)
	if err != nil {
		t.Fatalf("live Gemini ListModels failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatalf("live Gemini ListModels returned 0 models")
	}
}
