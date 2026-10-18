package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

func TestAnthropicInferenceProvider_MetadataAndCapabilities(t *testing.T) {
	adapter := NewProviderAdapter()

	if adapter.Name() != "Anthropic Claude" {
		t.Errorf("expected name 'Anthropic Claude', got %s", adapter.Name())
	}
	if adapter.Kind() != domain.ProviderKindAnthropic {
		t.Errorf("expected kind %s, got %s", domain.ProviderKindAnthropic, adapter.Kind())
	}

	meta := adapter.Metadata()
	if meta.Kind != domain.ProviderKindAnthropic || meta.AuthHeaderFormat != "x-api-key: <token>" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	caps, err := adapter.GetModelCapabilities(context.Background(), "claude-sonnet-4-5")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if caps.ContextLimit != 200000 || caps.MaxOutputTokens != 8192 {
		t.Errorf("unexpected limits for sonnet: %+v", caps)
	}
	if !caps.SupportsStreaming || !caps.SupportsTools || !caps.SupportsVision {
		t.Errorf("expected sonnet to support streaming, tools, vision: %+v", caps)
	}

	haikuCaps, err := adapter.GetModelCapabilities(context.Background(), "claude-3-5-haiku-20241022")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if haikuCaps.MaxOutputTokens != 4096 {
		t.Errorf("expected haiku max output 4096, got %d", haikuCaps.MaxOutputTokens)
	}
}

func TestAnthropicInferenceProvider_Execute(t *testing.T) {
	// Mock server for Anthropic messages API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" {
			if r.Header.Get("x-api-key") != "sk-ant-test" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp := MessageResponse{
				ID:    "msg_test_123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-sonnet-4-5",
				Content: []ContentBlock{
					{Type: "text", Text: "Hello from mock Claude!"},
				},
				StopReason: "end_turn",
				Usage: Usage{
					InputTokens:  25,
					OutputTokens: 10,
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
		version: defaultAnthropicVersion,
	}

	req := provider.NormalizedRequest{
		Model: "claude-sonnet-4-5",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Hello Claude"},
		},
		SystemPrompt: "Be concise.",
		MaxTokens:    1024,
	}

	resp, err := adapter.Execute(context.Background(), "sk-ant-test", req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.ID != "msg_test_123" || resp.Text != "Hello from mock Claude!" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if resp.Usage.PromptTokens != 25 || resp.Usage.CompletionTokens != 10 || resp.Usage.TotalTokens != 35 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}
