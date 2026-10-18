package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

func TestOpenAIInferenceProvider_MetadataAndCapabilities(t *testing.T) {
	adapter := NewProviderAdapter()

	if adapter.Name() != "OpenAI" {
		t.Errorf("expected name 'OpenAI', got %s", adapter.Name())
	}
	if adapter.Kind() != domain.ProviderKindOpenAI {
		t.Errorf("expected kind %s, got %s", domain.ProviderKindOpenAI, adapter.Kind())
	}

	meta := adapter.Metadata()
	if meta.Kind != domain.ProviderKindOpenAI || meta.AuthHeaderFormat != "Bearer <token>" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	caps, err := adapter.GetModelCapabilities(context.Background(), "gpt-4o")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if caps.ContextLimit != 128000 || caps.MaxOutputTokens != 16384 {
		t.Errorf("unexpected limits for gpt-4o: %+v", caps)
	}
	if !caps.SupportsStreaming || !caps.SupportsTools || !caps.SupportsVision || !caps.SupportsJSON {
		t.Errorf("expected gpt-4o to support streaming, tools, vision, json: %+v", caps)
	}

	o1Caps, err := adapter.GetModelCapabilities(context.Background(), "o1-mini")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if o1Caps.MaxOutputTokens != 65536 {
		t.Errorf("expected o1 max output 65536, got %d", o1Caps.MaxOutputTokens)
	}
}

func TestOpenAIInferenceProvider_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			if r.Header.Get("Authorization") != "Bearer sk-test-key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp := ChatCompletionResponse{
				ID:    "chatcmpl-test1234",
				Model: "gpt-4o",
				Choices: []Choice{
					{
						Index: 0,
						Message: ChatMessage{
							Role:    "assistant",
							Content: "Hello from mock OpenAI!",
						},
						FinishReason: "stop",
					},
				},
				Usage: &Usage{
					PromptTokens:     15,
					CompletionTokens: 8,
					TotalTokens:      23,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := &ProviderAdapter{
		baseURL: server.URL,
		version: "2024-08-01-preview",
	}

	req := provider.NormalizedRequest{
		Model: "gpt-4o",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Hello OpenAI"},
		},
		SystemPrompt: "You are a helpful assistant.",
		MaxTokens:    1024,
	}

	resp, err := adapter.Execute(context.Background(), "sk-test-key", req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.ID != "chatcmpl-test1234" || resp.Text != "Hello from mock OpenAI!" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if resp.Usage.PromptTokens != 15 || resp.Usage.CompletionTokens != 8 || resp.Usage.TotalTokens != 23 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestOpenAIInferenceProvider_EstimateTokens(t *testing.T) {
	adapter := NewProviderAdapter()
	req := provider.NormalizedRequest{
		Model: "gpt-4o",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "1234567890 1234567890"}, // ~21 chars => ~5 tokens
		},
	}

	usage, err := adapter.EstimateTokens(context.Background(), "dummy-key", req)
	if err != nil {
		t.Fatalf("EstimateTokens failed: %v", err)
	}
	if usage.PromptTokens == 0 {
		t.Errorf("expected positive token estimate, got 0")
	}
}
