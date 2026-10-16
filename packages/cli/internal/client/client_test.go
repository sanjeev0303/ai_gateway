package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/packages/cli/internal/client"
)

func TestGatewayClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			http.Error(w, `{"error":{"message":"unauthorized"}}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintln(w, `{"data":[{"id":"gpt-4o","object":"model","provider":"openai"},{"id":"claude-sonnet-4-5","object":"model","provider":"anthropic"}]}`)
	}))
	defer server.Close()

	c := client.New(server.URL, "test-api-key", "")
	models, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "gpt-4o" || models[1].ID != "claude-sonnet-4-5" {
		t.Errorf("unexpected models returned: %+v", models)
	}
}

func TestGatewayClient_StreamRequest_OpenAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("x-request-id", "req-test-stream-openai")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
			`data: {"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "proj-123")
	var receivedText string
	meta, err := c.StreamRequest(context.Background(), client.RequestPayload{
		Model:  "gpt-4o",
		Prompt: "Say hello",
		Stream: true,
	}, func(chunk string) {
		receivedText += chunk
	})

	if err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if receivedText != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", receivedText)
	}
	if meta.RequestID != "req-test-stream-openai" {
		t.Errorf("expected request id 'req-test-stream-openai', got %q", meta.RequestID)
	}
	if meta.InputTokens != 5 || meta.OutputTokens != 2 || meta.TotalTokens != 7 {
		t.Errorf("unexpected token usage: %+v", meta)
	}
}

func TestGatewayClient_StreamRequest_Anthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("x-request-id", "req-test-stream-anthropic")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			`data: {"type":"message_start","message":{"usage":{"input_tokens":12}}}`,
			`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi there!"}}`,
			`data: {"type":"message_delta","usage":{"output_tokens":4}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "")
	var receivedText string
	meta, err := c.StreamRequest(context.Background(), client.RequestPayload{
		Model:  "claude-sonnet-4-5",
		Prompt: "Say hi",
		Stream: true,
	}, func(chunk string) {
		receivedText += chunk
	})

	if err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if receivedText != "Hi there!" {
		t.Errorf("expected 'Hi there!', got %q", receivedText)
	}
	if meta.InputTokens != 12 || meta.OutputTokens != 4 || meta.TotalTokens != 16 {
		t.Errorf("unexpected token counts: %+v", meta)
	}
}

func TestGatewayClient_SendRequest_NonStreaming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-request-id", "req-nonstream-openai")
			_, _ = fmt.Fprintln(w, `{"choices":[{"message":{"content":"OpenAI answer"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
			return
		}
		if r.URL.Path == "/v1/messages" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("x-request-id", "req-nonstream-anthropic")
			_, _ = fmt.Fprintln(w, `{"content":[{"type":"text","text":"Anthropic answer"}],"usage":{"input_tokens":8,"output_tokens":6}}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "")

	// Test OpenAI route
	out1, meta1, err1 := c.SendRequest(context.Background(), client.RequestPayload{
		Model:  "gpt-4o",
		Prompt: "question 1",
	})
	if err1 != nil {
		t.Fatalf("openai send error: %v", err1)
	}
	if out1 != "OpenAI answer" || meta1.TotalTokens != 15 {
		t.Errorf("unexpected response: %q, tokens=%d", out1, meta1.TotalTokens)
	}

	// Test Anthropic route
	out2, meta2, err2 := c.SendRequest(context.Background(), client.RequestPayload{
		Model:  "claude-sonnet-4-5",
		Prompt: "question 2",
	})
	if err2 != nil {
		t.Fatalf("anthropic send error: %v", err2)
	}
	if out2 != "Anthropic answer" || meta2.TotalTokens != 14 {
		t.Errorf("unexpected response: %q, tokens=%d", out2, meta2.TotalTokens)
	}
}

func TestGatewayClient_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprintln(w, `{"error":{"type":"policy_denied","message":"Model access denied by corporate policy"}}`)
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "")
	_, _, err := c.SendRequest(context.Background(), client.RequestPayload{
		Model:  "blocked-model",
		Prompt: "test",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if expected := "[HTTP 403 policy_denied] Model access denied by corporate policy"; err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestGatewayClient_StreamRequest_Gemini(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("x-request-id", "req-test-stream-gemini")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		chunks := []string{
			`data: {"candidates":[{"content":{"parts":[{"text":"Hello from"}]}}]}`,
			`data: {"candidates":[{"content":{"parts":[{"text":" Gemini!"}]}}],"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":8,"totalTokenCount":20}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "%s\n\n", chunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "proj-gemini")
	var receivedText string
	meta, err := c.StreamRequest(context.Background(), client.RequestPayload{
		Model:  "gemini-2.5-flash",
		Prompt: "Hello Gemini",
		Stream: true,
	}, func(chunk string) {
		receivedText += chunk
	})

	if err != nil {
		t.Fatalf("unexpected stream error: %v", err)
	}

	if receivedText != "Hello from Gemini!" {
		t.Errorf("expected 'Hello from Gemini!', got %q", receivedText)
	}
	if meta.RequestID != "req-test-stream-gemini" {
		t.Errorf("expected request id 'req-test-stream-gemini', got %q", meta.RequestID)
	}
	if meta.InputTokens != 12 || meta.OutputTokens != 8 || meta.TotalTokens != 20 {
		t.Errorf("unexpected token usage: %+v", meta)
	}
}

func TestGatewayClient_SendRequest_Gemini(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-request-id", "req-nonstream-gemini")
		_, _ = fmt.Fprintln(w, `{"candidates":[{"content":{"parts":[{"text":"Gemini non-streaming response"}]}}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":10,"totalTokenCount":25}}`)
	}))
	defer server.Close()

	c := client.New(server.URL, "test-key", "")
	out, meta, err := c.SendRequest(context.Background(), client.RequestPayload{
		Model:  "gemini-2.5-flash",
		Prompt: "Test prompt",
	})
	if err != nil {
		t.Fatalf("gemini send error: %v", err)
	}
	if out != "Gemini non-streaming response" {
		t.Errorf("unexpected output: %q", out)
	}
	if meta.TotalTokens != 25 || meta.InputTokens != 15 || meta.OutputTokens != 10 {
		t.Errorf("unexpected tokens: %+v", meta)
	}
}
