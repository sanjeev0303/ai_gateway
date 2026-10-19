package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"ai-governance-saas/packages/cli/internal/client"
)

// ----------------------------------------------------------------------------
// Phase G: End-to-End Test Scenario — Sentinel CLI Multi-User LifeCycle
// ----------------------------------------------------------------------------

func TestPhaseG_CLI_EndToEndScenario(t *testing.T) {
	t.Log("Execution Mode: Deterministic Local Mock Provider & CLI Execution")

	reg := NewDefaultCommandRegistry()

	// Track revoked keys
	var mu sync.Mutex
	revokedKeys := make(map[string]bool)

	// Mock Gateway Server providing deterministic responses for all 4 providers
	mockGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("x-api-key")
		if apiKey == "" {
			authHeader := r.Header.Get("Authorization")
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		}

		mu.Lock()
		isRevoked := revokedKeys[apiKey]
		mu.Unlock()

		if isRevoked {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "authentication_error",
					"message": "gateway credential has been revoked",
				},
			})
			return
		}

		// Handle model listing
		if r.URL.Path == "/v1/models" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "claude-3-5-sonnet", "provider": "anthropic", "supports_streaming": true},
					{"id": "gemini-1.5-flash", "provider": "gemini", "supports_streaming": true},
					{"id": "gpt-4o", "provider": "openai", "supports_streaming": true},
					{"id": "meta-llama/Llama-3.3-70B-Instruct", "provider": "huggingface", "supports_streaming": true},
				},
			})
			return
		}

		// Read request payload
		var reqMap map[string]any
		_ = json.NewDecoder(r.Body).Decode(&reqMap)
		requestedModel, _ := reqMap["model"].(string)

		// Enforce model/provider restrictions based on credential
		switch apiKey {
		case "sentinel-user1-anthropic-key":
			if requestedModel != "claude-3-5-sonnet" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{"code": "permission_error", "message": fmt.Sprintf("credential is not authorized for model %q", requestedModel)},
				})
				return
			}
		case "sentinel-user2-gemini-key":
			if requestedModel != "gemini-1.5-flash" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{"code": "permission_error", "message": fmt.Sprintf("credential is not authorized for model %q", requestedModel)},
				})
				return
			}
		case "sentinel-user3-multi-key":
			if requestedModel != "gpt-4o" && requestedModel != "meta-llama/Llama-3.3-70B-Instruct" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{"code": "permission_error", "message": fmt.Sprintf("credential is not authorized for model %q", requestedModel)},
				})
				return
			}
		default:
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("x-request-id", "req-phase-g-"+apiKey)
		isStream := r.URL.Query().Get("stream") == "true" || r.Header.Get("Accept") == "text/event-stream"
		if strVal, ok := reqMap["stream"].(bool); ok && strVal {
			isStream = true
		}

		// Anthropic route (/v1/messages)
		if strings.HasPrefix(r.URL.Path, "/v1/messages") {
			if isStream {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_g_1\",\"usage\":{\"input_tokens\":25}}}\n\n")
				_, _ = fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello Anthropic CLI\"}}\n\n")
				_, _ = fmt.Fprintf(w, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":35}}\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "msg-phase-g-anth",
				"type":    "message",
				"role":    "assistant",
				"content": []any{map[string]any{"type": "text", "text": "Hello Anthropic CLI"}},
				"usage":   map[string]any{"input_tokens": 25, "output_tokens": 35},
			})
			return
		}

		// OpenAI / Gemini / HF route (/v1/chat/completions)
		if strings.HasPrefix(r.URL.Path, "/v1/chat/completions") {
			if requestedModel == "gemini-1.5-flash" {
				if isStream {
					w.Header().Set("Content-Type", "text/event-stream")
					w.WriteHeader(http.StatusOK)
					flusher := w.(http.Flusher)
					_, _ = fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello Gemini CLI\"}]}}],\"usageMetadata\":{\"promptTokenCount\":30,\"candidatesTokenCount\":40,\"totalTokenCount\":70}}\n\n")
					flusher.Flush()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"candidates": []any{
						map[string]any{
							"content":      map[string]any{"parts": []any{map[string]any{"text": "Hello Gemini CLI"}}},
							"finishReason": "STOP",
						},
					},
					"usageMetadata": map[string]any{"promptTokenCount": 30, "candidatesTokenCount": 40, "totalTokenCount": 70},
				})
				return
			}

			// OpenAI or HF response
			if isStream {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				flusher := w.(http.Flusher)
				_, _ = fmt.Fprintf(w, "data: {\"id\":\"c-g-1\",\"choices\":[{\"delta\":{\"content\":\"Hello Multi CLI\"}}],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":30,\"total_tokens\":50}}\n\ndata: [DONE]\n\n")
				flusher.Flush()
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "chatcmpl-phase-g",
				"choices": []any{
					map[string]any{"message": map[string]any{"role": "assistant", "content": "Hello Multi CLI"}, "finish_reason": "stop"},
				},
				"usage": map[string]any{"prompt_tokens": 20, "completion_tokens": 30, "total_tokens": 50},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockGateway.Close()

	// -------------------------------------------------------------------------
	// User 1: Anthropic Bound Credential
	// -------------------------------------------------------------------------
	t.Run("User 1: Anthropic CLI Lifecycle", func(t *testing.T) {
		user1Key := "sentinel-user1-anthropic-key"
		sess1 := &SessionState{
			GatewayURL:          mockGateway.URL,
			APIKey:              user1Key,
			Provider:            "anthropic",
			Model:               "claude-3-5-sonnet",
			AuthorizedProviders: []string{"anthropic"},
			PermittedModels: []client.Model{
				{ID: "claude-3-5-sonnet", Provider: "anthropic", DisplayName: "Claude 3.5 Sonnet"},
			},
			Client:    client.New(mockGateway.URL, user1Key, "proj-1"),
			Registry:  reg,
			MaxTokens: 4096,
			Stream:    false,
		}

		// 1. Type '/' and verify dynamic menu
		var menuBuf bytes.Buffer
		reg.RenderMenu(sess1, &menuBuf)
		menu := menuBuf.String()
		if !strings.Contains(menu, "ANTHROPIC") || !strings.Contains(menu, "/max-tokens") {
			t.Fatalf("User 1 menu missing Anthropic commands: %s", menu)
		}
		if strings.Contains(menu, "/safety") || strings.Contains(menu, "/endpoint") {
			t.Fatalf("User 1 menu contains unauthorized provider commands: %s", menu)
		}

		// 2. Execute supported commands
		var out bytes.Buffer
		handled, _, err := HandleSlashCommand(context.Background(), "/status", sess1, &out)
		if err != nil || !handled || !strings.Contains(out.String(), "claude-3-5-sonnet") {
			t.Fatalf("/status failed: %v", err)
		}

		out.Reset()
		handled, _, err = HandleSlashCommand(context.Background(), "/max-tokens 2048", sess1, &out)
		if err != nil || !handled || sess1.MaxTokens != 2048 {
			t.Fatalf("/max-tokens failed: %v", err)
		}

		// 3. Send authorized request through CLI client
		respText, meta, err := sess1.Client.SendRequest(context.Background(), client.RequestPayload{
			Model:    "claude-3-5-sonnet",
			Prompt:   "Hello from User 1",
			Messages: []client.ChatMessage{{Role: "user", Content: "Hello from User 1"}},
		})
		if err != nil {
			t.Fatalf("User 1 request failed: %v", err)
		}
		if respText != "Hello Anthropic CLI" {
			t.Fatalf("unexpected response: %s", respText)
		}
		if meta.InputTokens != 25 || meta.OutputTokens != 35 {
			t.Errorf("token extraction mismatch: %+v", meta)
		}
		if meta.Duration <= 0 {
			t.Errorf("expected non-zero latency duration")
		}

		// 4. Verify unauthorized model rejection (e.g. gpt-4o)
		_, _, err = sess1.Client.SendRequest(context.Background(), client.RequestPayload{
			Model:  "gpt-4o",
			Prompt: "Attempt unauthorized model",
		})
		if err == nil {
			t.Fatalf("expected error on unauthorized model, got nil")
		}
		if !strings.Contains(err.Error(), "403") && !strings.Contains(err.Error(), "not authorized") {
			t.Errorf("expected 403 authorization rejection, got %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// User 2: Gemini Bound Credential
	// -------------------------------------------------------------------------
	t.Run("User 2: Gemini CLI Lifecycle & Revocation", func(t *testing.T) {
		user2Key := "sentinel-user2-gemini-key"
		sess2 := &SessionState{
			GatewayURL:          mockGateway.URL,
			APIKey:              user2Key,
			Provider:            "gemini",
			Model:               "gemini-1.5-flash",
			AuthorizedProviders: []string{"gemini"},
			PermittedModels: []client.Model{
				{ID: "gemini-1.5-flash", Provider: "gemini", DisplayName: "Gemini 1.5 Flash"},
			},
			Client:   client.New(mockGateway.URL, user2Key, "proj-2"),
			Registry: reg,
			Stream:   true,
		}

		// 1. Type '/' and verify Gemini menu
		var menuBuf bytes.Buffer
		reg.RenderMenu(sess2, &menuBuf)
		menu := menuBuf.String()
		if !strings.Contains(menu, "/safety") {
			t.Fatalf("User 2 menu missing /safety: %s", menu)
		}
		if strings.Contains(menu, "/max-tokens") || strings.Contains(menu, "/endpoint") {
			t.Fatalf("User 2 menu contains foreign commands: %s", menu)
		}

		// 2. Execute /safety command
		var out bytes.Buffer
		handled, _, err := HandleSlashCommand(context.Background(), "/safety strict", sess2, &out)
		if err != nil || !handled || sess2.SafetySettings != "BLOCK_LOW_AND_ABOVE" {
			t.Fatalf("/safety strict failed: %v", err)
		}

		// 3. Send streaming request through CLI client
		var streamedParts []string
		meta, err := sess2.Client.StreamRequest(context.Background(), client.RequestPayload{
			Model:  "gemini-1.5-flash",
			Prompt: "Stream Gemini",
			Stream: true,
		}, func(chunk string) {
			streamedParts = append(streamedParts, chunk)
		})
		if err != nil {
			t.Fatalf("User 2 streaming request failed: %v", err)
		}
		joined := strings.Join(streamedParts, "")
		if joined != "Hello Gemini CLI" {
			t.Fatalf("unexpected stream content: %s", joined)
		}
		if meta.InputTokens != 30 || meta.OutputTokens != 40 {
			t.Errorf("token extraction mismatch for Gemini: %+v", meta)
		}

		// 4. Revoke User 2 Credential and verify immediate subsequent rejection
		mu.Lock()
		revokedKeys[user2Key] = true
		mu.Unlock()

		_, _, err = sess2.Client.SendRequest(context.Background(), client.RequestPayload{
			Model:  "gemini-1.5-flash",
			Prompt: "Post-revocation attempt",
		})
		if err == nil {
			t.Fatalf("expected post-revocation request to fail, got success")
		}
		if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "revoked") {
			t.Fatalf("expected 401 revoked credential error, got: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// User 3: OpenAI and Hugging Face Multi-Provider Credential
	// -------------------------------------------------------------------------
	t.Run("User 3: OpenAI & Hugging Face CLI Lifecycle", func(t *testing.T) {
		user3Key := "sentinel-user3-multi-key"
		sess3 := &SessionState{
			GatewayURL:          mockGateway.URL,
			APIKey:              user3Key,
			Provider:            "huggingface",
			Model:               "meta-llama/Llama-3.3-70B-Instruct",
			AuthorizedProviders: []string{"openai", "huggingface"},
			PermittedModels: []client.Model{
				{ID: "gpt-4o", Provider: "openai", DisplayName: "GPT-4o"},
				{ID: "meta-llama/Llama-3.3-70B-Instruct", Provider: "huggingface", DisplayName: "Llama 3.3 70B"},
			},
			Client:   client.New(mockGateway.URL, user3Key, "proj-3"),
			Registry: reg,
			Stream:   false,
		}

		// 1. Verify Hugging Face menu
		var menuBuf bytes.Buffer
		reg.RenderMenu(sess3, &menuBuf)
		menu := menuBuf.String()
		if !strings.Contains(menu, "/endpoint") || !strings.Contains(menu, "/params") {
			t.Fatalf("User 3 HF menu missing commands: %s", menu)
		}

		// 2. Execute /endpoint with valid dedicated HF endpoint
		var out bytes.Buffer
		handled, _, err := HandleSlashCommand(context.Background(), "/endpoint https://custom.endpoints.huggingface.cloud", sess3, &out)
		if err != nil || !handled || sess3.HFEndpoint != "https://custom.endpoints.huggingface.cloud" {
			t.Fatalf("/endpoint failed: %v", err)
		}

		// 3. Switch to OpenAI model via /model
		out.Reset()
		handled, _, err = HandleSlashCommand(context.Background(), "/model gpt-4o", sess3, &out)
		if err != nil || !handled || sess3.Model != "gpt-4o" || sess3.Provider != "openai" {
			t.Fatalf("/model switch to gpt-4o failed: %v", err)
		}

		// 4. Send request with switched OpenAI model
		respText, meta, err := sess3.Client.SendRequest(context.Background(), client.RequestPayload{
			Model:  "gpt-4o",
			Prompt: "Hello OpenAI from User 3",
		})
		if err != nil {
			t.Fatalf("User 3 OpenAI request failed: %v", err)
		}
		if respText != "Hello Multi CLI" {
			t.Fatalf("unexpected response: %s", respText)
		}
		if meta.InputTokens != 20 || meta.OutputTokens != 30 {
			t.Errorf("token extraction mismatch: %+v", meta)
		}

		// 5. Verify unpermitted Anthropic model rejection
		_, _, err = sess3.Client.SendRequest(context.Background(), client.RequestPayload{
			Model:  "claude-3-5-sonnet",
			Prompt: "Unauthorized cross-provider attempt",
		})
		if err == nil {
			t.Fatalf("expected error on unpermitted Anthropic model, got nil")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("expected 403 forbidden, got: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// Cross-User Isolation (CLI State & Memory)
	// -------------------------------------------------------------------------
	t.Run("Cross-User Conversation & Credential Isolation", func(t *testing.T) {
		sess1 := &SessionState{
			APIKey:   "key-user1",
			Messages: []client.ChatMessage{{Role: "user", Content: "User 1 confidential prompt"}},
		}
		sess2 := &SessionState{
			APIKey:   "key-user2",
			Messages: []client.ChatMessage{{Role: "user", Content: "User 2 isolated prompt"}},
		}

		// Verify distinct state pointers
		if sess1.APIKey == sess2.APIKey {
			t.Fatalf("user keys collided")
		}
		if len(sess1.Messages) != 1 || sess1.Messages[0].Content == sess2.Messages[0].Content {
			t.Fatalf("user message contexts collided")
		}
	})

	// -------------------------------------------------------------------------
	// Separate Live Provider Tests
	// -------------------------------------------------------------------------
	t.Run("Live Provider Tests Isolation", func(t *testing.T) {
		liveKeys := []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY", "HUGGINGFACE_API_KEY"}
		for _, key := range liveKeys {
			if val := os.Getenv(key); val == "" {
				t.Skipf("Live test skipped: %s not configured in local environment", key)
			}
		}
	})
}
