package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

func TestGeminiInferenceProvider_MetadataAndCapabilities(t *testing.T) {
	adapter := NewProviderAdapter()

	if adapter.Name() != "Google Gemini" {
		t.Errorf("expected name 'Google Gemini', got %s", adapter.Name())
	}
	if adapter.Kind() != domain.ProviderKindGemini {
		t.Errorf("expected kind %s, got %s", domain.ProviderKindGemini, adapter.Kind())
	}

	meta := adapter.Metadata()
	if meta.Kind != domain.ProviderKindGemini || meta.AuthHeaderFormat != "x-goog-api-key: <token>" {
		t.Errorf("unexpected metadata: %+v", meta)
	}

	caps, err := adapter.GetModelCapabilities(context.Background(), "gemini-1.5-pro")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if caps.ContextLimit != 2097152 || caps.MaxOutputTokens != 8192 {
		t.Errorf("unexpected limits for gemini-1.5-pro: %+v", caps)
	}
	if !caps.SupportsStreaming || !caps.SupportsTools || !caps.SupportsVision {
		t.Errorf("expected gemini-1.5-pro to support streaming, tools, vision: %+v", caps)
	}

	flashCaps, err := adapter.GetModelCapabilities(context.Background(), "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("GetModelCapabilities failed: %v", err)
	}
	if flashCaps.MaxOutputTokens != 65536 {
		t.Errorf("expected gemini-2.0-flash max output 65536, got %d", flashCaps.MaxOutputTokens)
	}
}

func TestGeminiInferenceProvider_Execute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "valid-gemini-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp := GenerateContentResponse{
			Candidates: []Candidate{
				{
					Index: 0,
					Content: &Content{
						Role: "model",
						Parts: []Part{
							{Text: "Hello from mock Gemini!"},
						},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: &UsageMetadata{
				PromptTokenCount:     12,
				CandidatesTokenCount: 6,
				TotalTokenCount:      18,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &ProviderAdapter{
		baseURL: server.URL,
		version: defaultVersion,
	}

	req := provider.NormalizedRequest{
		Model: "gemini-1.5-flash",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Hello Gemini"},
		},
		SystemPrompt: "Be helpful.",
		MaxTokens:    1024,
	}

	resp, err := adapter.Execute(context.Background(), "valid-gemini-key", req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if resp.Text != "Hello from mock Gemini!" {
		t.Errorf("unexpected text: %s", resp.Text)
	}
	if resp.Usage.PromptTokens != 12 || resp.Usage.CompletionTokens != 6 || resp.Usage.TotalTokens != 18 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestGeminiInferenceProvider_EstimateTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CountTokensResponse{
			TotalTokens: 42,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := &ProviderAdapter{
		baseURL: server.URL,
		version: defaultVersion,
	}

	req := provider.NormalizedRequest{
		Model: "gemini-1.5-flash",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "Count these tokens"},
		},
	}

	usage, err := adapter.EstimateTokens(context.Background(), "test-key", req)
	if err != nil {
		t.Fatalf("EstimateTokens failed: %v", err)
	}
	if usage.TotalTokens != 42 || usage.PromptTokens != 42 {
		t.Errorf("unexpected token estimate: %+v", usage)
	}
}
