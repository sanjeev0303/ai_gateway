package huggingface

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

func TestHuggingFaceInferenceProvider_MetadataAndCapabilities(t *testing.T) {
	adapter := NewProviderAdapter()

	if adapter.Name() != "Hugging Face" {
		t.Errorf("expected name 'Hugging Face', got %s", adapter.Name())
	}
	if adapter.Kind() != domain.ProviderKindHuggingFace {
		t.Errorf("expected kind %s, got %s", domain.ProviderKindHuggingFace, adapter.Kind())
	}

	meta := adapter.Metadata()
	if meta.Kind != domain.ProviderKindHuggingFace || meta.AuthHeaderFormat != "Bearer <token>" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	caps, err := adapter.GetModelCapabilities(context.Background(), "meta-llama/Llama-3.3-70B-Instruct")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if caps.ContextLimit != 131072 || caps.MaxOutputTokens != 8192 {
		t.Errorf("unexpected limits for llama-3.3: %+v", caps)
	}
	if !caps.SupportsStreaming || caps.SupportsTools || caps.SupportsVision {
		t.Errorf("expected llama on hf to support streaming only without tools or vision: %+v", caps)
	}

	// Verify unsupported tool call request fails validation
	reqWithTools := provider.NormalizedRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Tools: []provider.ToolDefinition{
			{Name: "get_weather"},
		},
	}
	if err := caps.ValidateRequest(reqWithTools); err == nil {
		t.Errorf("expected validation failure for tool request on provider without tool support")
	}
}

func TestHuggingFaceInferenceProvider_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer hf-valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp := ChatCompletionResponse{
			ID:    "hf-chat-123",
			Model: "meta-llama/Llama-3.3-70B-Instruct",
			Choices: []ChatChoice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Hello from Hugging Face!",
					},
					FinishReason: "stop",
				},
			},
			Usage: &UsageInfo{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &ProviderAdapter{
		baseURL: server.URL + "/v1",
		hubURL:  server.URL,
	}

	req := provider.NormalizedRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Hello HF"},
		},
		SystemPrompt: "Be helpful.",
		MaxTokens:    512,
	}

	resp, err := adapter.Execute(context.Background(), "hf-valid-token", req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.Text != "Hello from Hugging Face!" {
		t.Errorf("unexpected text: %s", resp.Text)
	}
	if resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 5 || resp.Usage.TotalTokens != 15 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestHuggingFaceInferenceProvider_EstimateTokens(t *testing.T) {
	adapter := NewProviderAdapter()
	req := provider.NormalizedRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Short input"},
		},
	}

	usage, err := adapter.EstimateTokens(context.Background(), "dummy-key", req)
	if err != nil {
		t.Fatalf("EstimateTokens failed: %v", err)
	}
	if usage.TotalTokens == 0 {
		t.Errorf("expected positive token estimate, got 0")
	}
}
