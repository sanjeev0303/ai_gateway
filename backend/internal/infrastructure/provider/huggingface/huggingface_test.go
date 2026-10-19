package huggingface

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/ports/provider"
)

func TestValidateCredentials(t *testing.T) {
	t.Parallel()

	t.Run("valid credentials", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/whoami-v2" {
				if r.Header.Get("Authorization") == "Bearer hf_valid_token" {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"type":"user","name":"test-user"}`))
					return
				}
			}
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		err := adapter.ValidateCredentials(context.Background(), "hf_valid_token")
		if err != nil {
			t.Fatalf("expected valid credentials, got: %v", err)
		}
	})

	t.Run("invalid credentials returns ErrInvalidCredentials", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		err := adapter.ValidateCredentials(context.Background(), "hf_invalid_token")
		if err == nil {
			t.Fatal("expected error for invalid credentials, got nil")
		}
		if !errors.Is(err, provider.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
		}
	})

	t.Run("empty API key", func(t *testing.T) {
		adapter := NewProviderAdapter()
		err := adapter.ValidateCredentials(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty key, got nil")
		}
	})
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
	if err := adapter.HealthCheck(context.Background(), "hf_token"); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

func TestListModels(t *testing.T) {
	t.Parallel()

	t.Run("hub returns model list", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/models" {
				models := []HubModelItem{
					{ID: "meta-llama/Llama-3.3-70B-Instruct", PipelineTag: "text-generation"},
					{ID: "mistralai/Mistral-7B-Instruct-v0.3", PipelineTag: "text-generation"},
				}
				_ = json.NewEncoder(w).Encode(models)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		models, err := adapter.ListModels(context.Background(), "hf_token")
		if err != nil {
			t.Fatalf("list models failed: %v", err)
		}
		if len(models) != 2 {
			t.Fatalf("expected 2 models, got %d", len(models))
		}
		if models[0].ID != "meta-llama/Llama-3.3-70B-Instruct" {
			t.Errorf("unexpected model ID: %s", models[0].ID)
		}
	})

	t.Run("hub error returns failure without fallbacks", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		_, err := adapter.ListModels(context.Background(), "hf_token")
		if err == nil {
			t.Fatal("expected error on hub failure, got nil")
		}
	})

	t.Run("hub empty list returns empty without fallbacks", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode([]HubModelItem{})
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		models, err := adapter.ListModels(context.Background(), "hf_token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(models) != 0 {
			t.Fatalf("expected 0 models, got %d", len(models))
		}
	})

	t.Run("hub 401 returns ErrInvalidCredentials", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		_, err := adapter.ListModels(context.Background(), "hf_token")
		if !errors.Is(err, provider.ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("hub 404 returns ErrDiscoveryUnavailable", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer ts.Close()

		adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
		_, err := adapter.ListModels(context.Background(), "hf_token")
		if !errors.Is(err, provider.ErrDiscoveryUnavailable) {
			t.Fatalf("expected ErrDiscoveryUnavailable, got %v", err)
		}
	})
}

func TestCreateMessage(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer hf_token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		resp := ChatCompletionResponse{
			ID:      "hf-12345",
			Object:  "chat.completion",
			Created: 1700000000,
			Model:   "meta-llama/Llama-3.3-70B-Instruct",
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
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
	req := &ChatCompletionRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hello!"},
		},
	}
	resp, err := adapter.CreateMessage(context.Background(), "hf_token", req)
	if err != nil {
		t.Fatalf("create message failed: %v", err)
	}
	if resp.ID != "hf-12345" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Hello from Hugging Face!" {
		t.Errorf("unexpected message content")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 18 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestCreateMessageStream(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		events := []string{
			`data: {"id":"hf-stream-1","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`,
			`data: {"id":"hf-stream-1","choices":[{"index":0,"delta":{"content":" world"}}]}`,
			`data: {"id":"hf-stream-1","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":3,"total_tokens":8}}`,
			`data: [DONE]`,
		}

		for _, ev := range events {
			_, _ = w.Write([]byte(ev + "\n\n"))
			flusher.Flush()
		}
	}))
	defer ts.Close()

	adapter := NewCustomProviderAdapter(ts.URL, ts.URL)
	req := &ChatCompletionRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hi"},
		},
	}

	stream, err := adapter.CreateMessageStream(context.Background(), "hf_token", req)
	if err != nil {
		t.Fatalf("create message stream failed: %v", err)
	}
	defer stream.Close()

	var assembled string
	for {
		chunk, err := stream.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("stream next failed: %v", err)
		}
		if len(chunk.Choices) > 0 {
			assembled += chunk.Choices[0].Delta.Content
		}
	}

	if assembled != "Hello world!" {
		t.Errorf("expected 'Hello world!', got '%s'", assembled)
	}
}

func TestCountTokens(t *testing.T) {
	t.Parallel()

	adapter := NewProviderAdapter()
	req := &ChatCompletionRequest{
		Model: "meta-llama/Llama-3.3-70B-Instruct",
		Messages: []ChatMessage{
			{Role: "user", Content: "This is a prompt with about twenty characters."},
		},
	}
	res, err := adapter.CountTokens(context.Background(), "hf_token", req)
	if err != nil {
		t.Fatalf("count tokens failed: %v", err)
	}
	if res.TotalTokens <= 0 {
		t.Errorf("expected positive token count, got %d", res.TotalTokens)
	}
}

// TestHuggingFace_LiveSmokeTest executes a live smoke test against Hugging Face only if HUGGINGFACE_API_KEY is configured.
func TestHuggingFace_LiveSmokeTest(t *testing.T) {
	apiKey := os.Getenv("HUGGINGFACE_API_KEY")
	if apiKey == "" {
		t.Skip("skipping live Hugging Face smoke test: HUGGINGFACE_API_KEY not set")
	}
	adapter := NewProviderAdapter()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := adapter.ValidateCredentials(ctx, apiKey); err != nil {
		t.Fatalf("live Hugging Face credential validation failed: %v", err)
	}
	models, err := adapter.ListModels(ctx, apiKey)
	if err != nil {
		t.Fatalf("live Hugging Face ListModels failed: %v", err)
	}
	if len(models) == 0 {
		t.Fatalf("live Hugging Face ListModels returned 0 models")
	}
}
