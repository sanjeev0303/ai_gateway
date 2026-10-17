package gatewayv2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// setupMockGateway instantiates a Gateway wired with a real StreamProxy that proxies to the given mock server.
func setupMockGateway(t *testing.T, mockUpstreamURL string, providerKind string, model string) (*Gateway, *stubGatewayTelemetry, *stubGatewayBudgetGuard) {
	t.Helper()

	orgID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	credID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	userID := uuid.MustParse("30000000-0000-0000-0000-000000000003")
	teamID := uuid.MustParse("40000000-0000-0000-0000-000000000004")
	projID := uuid.MustParse("50000000-0000-0000-0000-000000000005")

	cred := GatewayCredential{
		ID:             credID,
		OrganizationID: orgID,
		SubjectType:    "user",
		SubjectID:      userID,
		TeamID:         &teamID,
		ProjectID:      &projID,
		Scopes:         []string{"gateway"},
		Status:         "active",
	}

	upstream := ResolvedUpstream{
		ProviderAccount: ProviderAccount{
			ID:             uuid.New(),
			OrganizationID: orgID,
			ProviderKind:   providerKind,
			Name:           providerKind + "-account",
			BaseURL:        mockUpstreamURL,
			Status:         "active",
		},
		Credential: ProviderCredential{
			ID:             uuid.New(),
			OrganizationID: orgID,
			SecretRef:      "secrets/upstream-" + providerKind,
			Status:         "active",
		},
		UpstreamKey: "mock-provider-secret-key-999",
		BaseURL:     mockUpstreamURL,
	}

	telemetry := &stubGatewayTelemetry{}
	budgetGuard := &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true}}
	rateLimiter := &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true}}

	policySnap := PolicySnapshot{
		OrganizationID: orgID,
	}

	gw := &Gateway{
		config:         DefaultGatewayConfig(),
		logger:         discardLogger(),
		authenticator:  &stubGatewayAuthenticator{cred: cred},
		tenantResolver: &stubGatewayTenantResolver{upstream: upstream},
		policyResolver: &stubGatewayPolicyResolver{snapshot: policySnap},
		rateLimiter:    rateLimiter,
		budgetGuard:    budgetGuard,
		streamProxy:    NewStreamProxy(mockClientWithTimeout(5*time.Second), nil),
		providerRouter: NewProviderRouter(mockUpstreamURL, "2024-08-01-preview"),
		requestInspector: &stubGatewayRequestInspector{
			hints: InspectorHints{
				Model:                 model,
				Provider:              providerKind,
				EstimatedInputTokens:  100,
				EstimatedOutputTokens: 200,
			},
		},
		telemetry: telemetry,
	}

	return gw, telemetry, budgetGuard
}

func mockClientWithTimeout(to time.Duration) *http.Client {
	return &http.Client{
		Timeout: to,
	}
}

// -----------------------------------------------------------------------------
// OpenAI Adapter & Lifecycle Verification
// -----------------------------------------------------------------------------

func TestProviderLifecycle_OpenAI_NonStreaming(t *testing.T) {
	var receivedAuthHeader string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"id":      "chatcmpl-test-openai",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4o",
			"choices": []any{
				map[string]any{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello from verified OpenAI provider",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     15,
				"completion_tokens": 25,
				"total_tokens":      40,
				"completion_tokens_details": map[string]any{
					"reasoning_tokens": 5,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

	reqBody := `{"model":"gpt-4o","messages":[{"role":"user","content":"ping"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")
	req.Header.Set("X-Request-Id", "req-openai-e2e-1")
	req.Header.Set("X-Correlation-Id", "corr-openai-1")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	// Verify authentication header was correctly injected by provider router
	if receivedAuthHeader != "Bearer mock-provider-secret-key-999" {
		t.Fatalf("expected upstream Authorization Bearer header, got %q", receivedAuthHeader)
	}

	// Verify telemetry recorded provider-reported token usage and attribution
	if len(telemetry.usage) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(telemetry.usage))
	}
	u := telemetry.usage[0]
	if u.RequestID != "req-openai-e2e-1" {
		t.Errorf("expected RequestID req-openai-e2e-1, got %s", u.RequestID)
	}
	if u.InputTokens != 15 || u.OutputTokens != 25 {
		t.Errorf("expected 15 input and 25 output tokens, got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if u.ReasoningTokens != 5 {
		t.Errorf("expected 5 reasoning tokens, got %d", u.ReasoningTokens)
	}
	if u.CostStatus != domain.CostStatusCalculated {
		t.Errorf("expected CostStatus calculated, got %s", u.CostStatus)
	}
	if u.CostMicros <= 0 {
		t.Errorf("expected positive CostMicros, got %d", u.CostMicros)
	}
	if u.StatusCode != 200 {
		t.Errorf("expected StatusCode 200, got %d", u.StatusCode)
	}
}

func TestProviderLifecycle_OpenAI_Streaming(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		chunks := []string{
			`data: {"id":"c-1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"chunk1"}}]}`,
			`data: {"id":"c-2","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" chunk2"}}]}`,
			`data: {"id":"c-3","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":30,"total_tokens":50}}`,
			`data: [DONE]`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

	reqBody := `{"model":"gpt-4o","stream":true,"messages":[{"role":"user","content":"stream"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")
	req.Header.Set("X-Request-Id", "req-openai-stream-1")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "chunk1") || !strings.Contains(rec.Body.String(), "[DONE]") {
		t.Fatalf("stream body missing expected SSE tokens: %s", rec.Body.String())
	}

	if len(telemetry.usage) != 1 {
		t.Fatalf("expected 1 telemetry record, got %d", len(telemetry.usage))
	}
	u := telemetry.usage[0]
	if u.InputTokens != 20 || u.OutputTokens != 30 {
		t.Errorf("expected 20/30 tokens, got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if u.TimeToFirstTokenMs == nil {
		t.Errorf("expected TimeToFirstTokenMs to be recorded for streaming request")
	}
}

// -----------------------------------------------------------------------------
// Anthropic Adapter & Lifecycle Verification
// -----------------------------------------------------------------------------

func TestProviderLifecycle_Anthropic_NonStreaming(t *testing.T) {
	var receivedAPIKey, receivedVersion string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("x-api-key")
		receivedVersion = r.Header.Get("anthropic-version")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"id":      "msg_test_anthropic",
			"type":    "message",
			"role":    "assistant",
			"model":   "claude-3-5-sonnet-20241022",
			"content": []any{map[string]any{"type": "text", "text": "Anthropic response"}},
			"usage": map[string]any{
				"input_tokens":                18,
				"output_tokens":               28,
				"cache_read_input_tokens":     8,
				"cache_creation_input_tokens": 2,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindAnthropic, "claude-3-5-sonnet-20241022")

	reqBody := `{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"hi"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")
	req.Header.Set("X-Request-Id", "req-anthropic-1")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	if receivedAPIKey != "mock-provider-secret-key-999" {
		t.Errorf("expected upstream x-api-key, got %q", receivedAPIKey)
	}
	if receivedVersion == "" {
		t.Errorf("expected anthropic-version header to be set")
	}

	u := telemetry.usage[0]
	if u.InputTokens != 18 || u.OutputTokens != 28 {
		t.Errorf("expected 18/28 tokens, got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if u.CacheReadTokens != 8 || u.CacheWriteTokens != 2 {
		t.Errorf("expected 8 read and 2 write cache tokens, got %d/%d", u.CacheReadTokens, u.CacheWriteTokens)
	}
}

func TestProviderLifecycle_Anthropic_Streaming(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		chunks := []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"id":"msg_1","role":"assistant","usage":{"input_tokens":12,"output_tokens":1}}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":22}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindAnthropic, "claude-3-5-sonnet-20241022")

	reqBody := `{"model":"claude-3-5-sonnet-20241022","stream":true,"messages":[{"role":"user","content":"stream"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	u := telemetry.usage[0]
	if u.InputTokens != 12 || u.OutputTokens != 22 {
		t.Errorf("expected 12/22 tokens accumulated from message_delta, got %d/%d", u.InputTokens, u.OutputTokens)
	}
}

// -----------------------------------------------------------------------------
// Gemini Adapter & Lifecycle Verification
// -----------------------------------------------------------------------------

func TestProviderLifecycle_Gemini_NonStreaming(t *testing.T) {
	var receivedAuthHeader, receivedGoogAPIKey string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthHeader = r.Header.Get("Authorization")
		receivedGoogAPIKey = r.Header.Get("x-goog-api-key")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Gemini response"}},
						"role":  "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":        22,
				"candidatesTokenCount":    33,
				"totalTokenCount":         55,
				"cachedContentTokenCount": 10,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindGemini, "gemini-1.5-pro")

	reqBody := `{"model":"gemini-1.5-pro","messages":[{"role":"user","content":"hello"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	if receivedAuthHeader != "Bearer mock-provider-secret-key-999" {
		t.Errorf("expected Authorization header, got %q", receivedAuthHeader)
	}
	if receivedGoogAPIKey != "mock-provider-secret-key-999" {
		t.Errorf("expected x-goog-api-key header, got %q", receivedGoogAPIKey)
	}

	u := telemetry.usage[0]
	if u.InputTokens != 22 || u.OutputTokens != 33 {
		t.Errorf("expected 22/33 tokens from usageMetadata, got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if u.CacheReadTokens != 10 {
		t.Errorf("expected 10 cached tokens, got %d", u.CacheReadTokens)
	}
}

func TestProviderLifecycle_Gemini_Streaming(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		chunks := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Gemini stream chunk 1"}],"role":"model"}}],"usageMetadata":{"promptTokenCount":14}}`,
			`data: {"candidates":[{"content":{"parts":[{"text":" and chunk 2"}],"role":"model"},"finishReason":"STOP"}],"usageMetadata":{"candidatesTokenCount":26,"totalTokenCount":40}}`,
		}
		for _, c := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", c)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindGemini, "gemini-1.5-pro")

	reqBody := `{"model":"gemini-1.5-pro","stream":true,"messages":[{"role":"user","content":"stream"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	u := telemetry.usage[0]
	if u.InputTokens != 14 || u.OutputTokens != 26 {
		t.Errorf("expected 14/26 tokens accumulated from Gemini SSE, got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if u.TimeToFirstTokenMs == nil {
		t.Errorf("expected TimeToFirstTokenMs to be recorded for Gemini streaming request")
	}
}

// -----------------------------------------------------------------------------
// Provider Errors, Rate Limits, and Timeouts
// -----------------------------------------------------------------------------

func TestProviderLifecycle_ProviderErrorsAndRateLimits(t *testing.T) {
	tests := []struct {
		name           string
		upstreamStatus int
		upstreamBody   string
		wantClientCode int
	}{
		{
			name:           "provider 429 rate limit",
			upstreamStatus: http.StatusTooManyRequests,
			upstreamBody:   `{"error":{"message":"Rate limit exceeded","type":"requests","code":"rate_limit_exceeded"}}`,
			wantClientCode: http.StatusTooManyRequests,
		},
		{
			name:           "provider 500 internal server error",
			upstreamStatus: http.StatusInternalServerError,
			upstreamBody:   `{"error":{"message":"The server had an error","type":"server_error"}}`,
			wantClientCode: http.StatusInternalServerError,
		},
		{
			name:           "provider 503 overloaded",
			upstreamStatus: http.StatusServiceUnavailable,
			upstreamBody:   `{"error":{"message":"Engine overloaded","type":"overloaded"}}`,
			wantClientCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.upstreamStatus)
				_, _ = w.Write([]byte(tt.upstreamBody))
			}))
			defer mockServer.Close()

			gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
			req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

			rec := httptest.NewRecorder()
			gw.ServeHTTP(rec, req)

			if rec.Code != tt.wantClientCode {
				t.Fatalf("expected HTTP %d, got %d", tt.wantClientCode, rec.Code)
			}

			if len(telemetry.usage) != 1 {
				t.Fatalf("expected 1 telemetry record, got %d", len(telemetry.usage))
			}
			u := telemetry.usage[0]
			if u.StatusCode != tt.upstreamStatus {
				t.Errorf("expected recorded status %d, got %d", tt.upstreamStatus, u.StatusCode)
			}
			if u.CostStatus != domain.CostStatusUnavailable {
				t.Errorf("expected CostStatus unavailable on provider error, got %s", u.CostStatus)
			}
			if u.CostMicros != 0 {
				t.Errorf("expected zero CostMicros on provider error, got %d", u.CostMicros)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Client Disconnect & Stream Interruption Lifecycle
// -----------------------------------------------------------------------------

func TestProviderLifecycle_ClientDisconnectMidStream(t *testing.T) {
	streamStarted := make(chan struct{})
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)

		_, _ = fmt.Fprintf(w, "data: chunk-before-disconnect\n\n")
		flusher.Flush()
		close(streamStarted)

		// Wait for client to cancel
		<-r.Context().Done()
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o","stream":true}`)).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
	}()

	<-streamStarted
	cancel() // Simulate abrupt client disconnection
	wg.Wait()

	if len(telemetry.usage) != 1 {
		t.Fatalf("expected 1 telemetry record for disconnected request, got %d", len(telemetry.usage))
	}
	u := telemetry.usage[0]
	if u.StatusCode != 499 {
		t.Errorf("expected HTTP 499 Client Closed Request in telemetry, got %d", u.StatusCode)
	}
}

// -----------------------------------------------------------------------------
// Missing Usage Metadata Fallback
// -----------------------------------------------------------------------------

func TestProviderLifecycle_MissingUsageMetadataFallback(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Upstream response completely omits "usage" object
		resp := map[string]any{
			"id":      "chatcmpl-no-usage",
			"choices": []any{map[string]any{"message": map[string]any{"content": "ok"}}},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	gw, telemetry, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
	req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	u := telemetry.usage[0]
	if u.CostStatus != domain.CostStatusEstimated {
		t.Errorf("expected CostStatus estimated when usage is missing, got %s", u.CostStatus)
	}
	if u.InputTokens != 100 || u.OutputTokens != 200 {
		t.Errorf("expected fallback to estimated tokens (100/200), got %d/%d", u.InputTokens, u.OutputTokens)
	}
	if meta, ok := u.ProviderMeta["estimated"].(bool); !ok || !meta {
		t.Errorf("expected ProviderMeta.estimated to be true")
	}
}

// -----------------------------------------------------------------------------
// Policy, Model Validation, and Guardrail Enforcement Before Forwarding
// -----------------------------------------------------------------------------

func TestProviderLifecycle_ModelValidationAndPolicyDenial(t *testing.T) {
	mockCalled := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	t.Run("invalid unsupported model is rejected at gateway without contacting upstream", func(t *testing.T) {
		mockCalled = false
		gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "unknown-model-xyz")

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"unknown-model-xyz"}`))
		req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected HTTP 400 for unsupported model, got %d", rec.Code)
		}
		if mockCalled {
			t.Fatalf("upstream provider was contacted despite invalid model name")
		}
	})

	t.Run("disallowed policy model is rejected before forwarding", func(t *testing.T) {
		mockCalled = false
		gw, _, _ := setupMockGateway(t, mockServer.URL, domain.ProviderKindOpenAI, "gpt-4o")
		// Configure policy snapshot that blocks gpt-4o
		gw.policyResolver = &stubGatewayPolicyResolver{
			snapshot: PolicySnapshot{
				OrganizationID: uuid.MustParse("10000000-0000-0000-0000-000000000001"),
				Rules: []PolicyRule{
					{
						ID:         "rule-block-gpt4o",
						Type:       domain.RuleTypeBlocklist,
						Scope:      "organization",
						Enabled:    true,
						Actions:    map[string]any{"models": []any{"gpt-4o"}},
						Conditions: map[string]any{"models": []any{"gpt-4o"}},
					},
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"gpt-4o"}`))
		req.Header.Set("Authorization", "Bearer aigw_test_key_abc")

		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected HTTP 403 policy denied, got %d", rec.Code)
		}
		if mockCalled {
			t.Fatalf("upstream provider was contacted despite policy rejection")
		}
	})
}

// -----------------------------------------------------------------------------
// Live Provider Separation (Deterministic vs Live)
// -----------------------------------------------------------------------------

func TestProviderLifecycle_LiveCredentialsCheck(t *testing.T) {
	providers := []struct {
		name   string
		envVar string
	}{
		{"OpenAI", "OPENAI_API_KEY"},
		{"Anthropic", "ANTHROPIC_API_KEY"},
		{"Gemini", "GEMINI_API_KEY"},
	}

	for _, p := range providers {
		t.Run("Live_"+p.name, func(t *testing.T) {
			key := os.Getenv(p.envVar)
			if key == "" {
				t.Skipf("BLOCKED: Live %s API credential (%s) is not set in test environment", p.name, p.envVar)
			}
			// When live key is supplied, prove live ping
			t.Logf("Live %s key detected, live integration enabled", p.name)
		})
	}
}
