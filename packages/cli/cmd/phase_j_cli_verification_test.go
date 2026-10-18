package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-governance-saas/packages/cli/internal/client"
)

// ----------------------------------------------------------------------------
// Phase J: Sentinel CLI End-to-End Verification Test Suite
// ----------------------------------------------------------------------------

func TestPhaseJ_CLISlashCommandsAndMultiUserProviders(t *testing.T) {
	t.Log("Execution Mode: Deterministic Local Mock CLI Registry and Execution")

	reg := NewDefaultCommandRegistry()

	// Mock Gateway Server for commands that query the gateway
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/v1/messages") || strings.Contains(r.URL.Path, "/chat/completions") {
			resp := map[string]interface{}{
				"id": "msg_summary_test",
				"content": []map[string]interface{}{
					{"type": "text", "text": "Compacted conversation context summarizing previous questions and answers."},
				},
				"usage": map[string]interface{}{
					"input_tokens":  60,
					"output_tokens": 15,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// 1. User 1: Anthropic Bound Credential
	t.Run("User 1: Anthropic Slash Command Menu and Execution", func(t *testing.T) {
		sess1 := &SessionState{
			GatewayURL:          mockServer.URL,
			APIKey:              "sentinel-user1-anthropic-key",
			Provider:            "anthropic",
			Model:               "claude-3-5-sonnet",
			AuthorizedProviders: []string{"anthropic"},
			PermittedModels: []client.Model{
				{ID: "claude-3-5-sonnet", Provider: "anthropic", DisplayName: "Claude 3.5 Sonnet"},
			},
			Client:    client.New(mockServer.URL, "sentinel-user1-anthropic-key", ""),
			Registry:  reg,
			MaxTokens: 4096,
			Stream:    true,
		}

		// Verify menu rendering on '/'
		var menuBuf bytes.Buffer
		reg.RenderMenu(sess1, &menuBuf)
		menuOut := menuBuf.String()

		if !strings.Contains(menuOut, "ANTHROPIC") {
			t.Fatalf("expected header with anthropic provider, got: %s", menuOut)
		}
		if !strings.Contains(menuOut, "/status") || !strings.Contains(menuOut, "/stream") || !strings.Contains(menuOut, "/cost") {
			t.Fatalf("missing common commands in menu: %s", menuOut)
		}
		if !strings.Contains(menuOut, "/max-tokens") {
			t.Fatalf("missing anthropic /max-tokens in menu: %s", menuOut)
		}
		if strings.Contains(menuOut, "/safety") || strings.Contains(menuOut, "/endpoint") {
			t.Fatalf("unauthorized provider commands shown in anthropic menu: %s", menuOut)
		}

		// Execute /status
		var out bytes.Buffer
		handled, shouldExit, err := HandleSlashCommand(context.Background(), "/status", sess1, &out)
		if err != nil || !handled || shouldExit {
			t.Fatalf("/status failed: err=%v, handled=%v", err, handled)
		}
		if !strings.Contains(out.String(), "Session Status") || !strings.Contains(out.String(), "claude-3-5-sonnet") {
			t.Fatalf("unexpected /status output: %s", out.String())
		}

		// Execute /stream toggle
		out.Reset()
		handled, shouldExit, err = HandleSlashCommand(context.Background(), "/stream off", sess1, &out)
		if err != nil || !handled || shouldExit || sess1.Stream {
			t.Fatalf("/stream off failed: err=%v, stream=%v", err, sess1.Stream)
		}
	})

	// 2. User 2: Gemini Bound Credential
	t.Run("User 2: Gemini Slash Command Menu and Safety Command", func(t *testing.T) {
		sess2 := &SessionState{
			GatewayURL:          mockServer.URL,
			APIKey:              "sentinel-user2-gemini-key",
			Provider:            "gemini",
			Model:               "gemini-1.5-flash",
			AuthorizedProviders: []string{"gemini"},
			PermittedModels: []client.Model{
				{ID: "gemini-1.5-flash", Provider: "gemini", DisplayName: "Gemini 1.5 Flash"},
			},
			Client:   client.New(mockServer.URL, "sentinel-user2-gemini-key", ""),
			Registry: reg,
			Stream:   true,
		}

		var menuBuf bytes.Buffer
		reg.RenderMenu(sess2, &menuBuf)
		menuOut := menuBuf.String()

		if !strings.Contains(menuOut, "/safety") {
			t.Fatalf("missing gemini /safety in menu: %s", menuOut)
		}
		if strings.Contains(menuOut, "/max-tokens") || strings.Contains(menuOut, "/endpoint") {
			t.Fatalf("unauthorized commands in gemini menu: %s", menuOut)
		}

		// Execute /safety with strict preset
		var out bytes.Buffer
		handled, shouldExit, err := HandleSlashCommand(context.Background(), "/safety strict", sess2, &out)
		if err != nil || !handled || shouldExit {
			t.Fatalf("/safety execution failed: %v", err)
		}
		if !strings.Contains(out.String(), "BLOCK_LOW_AND_ABOVE") {
			t.Fatalf("expected safety output to mention BLOCK_LOW_AND_ABOVE, got %s", out.String())
		}
	})

	// 3. User 3: OpenAI and Hugging Face Multi-Provider Credential
	t.Run("User 3: OpenAI and Hugging Face Switching and Commands", func(t *testing.T) {
		sess3 := &SessionState{
			GatewayURL:          mockServer.URL,
			APIKey:              "sentinel-user3-multi-key",
			Provider:            "huggingface",
			Model:               "meta-llama/Llama-3.3-70B-Instruct",
			AuthorizedProviders: []string{"openai", "huggingface"},
			PermittedModels: []client.Model{
				{ID: "gpt-4o", Provider: "openai", DisplayName: "GPT-4o"},
				{ID: "meta-llama/Llama-3.3-70B-Instruct", Provider: "huggingface", DisplayName: "Llama 3.3 70B"},
			},
			Client:   client.New(mockServer.URL, "sentinel-user3-multi-key", ""),
			Registry: reg,
			Stream:   true,
		}

		// Verify Hugging Face menu
		var menuBuf bytes.Buffer
		reg.RenderMenu(sess3, &menuBuf)
		menuOut := menuBuf.String()

		if !strings.Contains(menuOut, "/endpoint") || !strings.Contains(menuOut, "/params") {
			t.Fatalf("missing huggingface commands in menu: %s", menuOut)
		}

		// Execute /params
		var out bytes.Buffer
		handled, shouldExit, err := HandleSlashCommand(context.Background(), "/params", sess3, &out)
		if err != nil || !handled || shouldExit {
			t.Fatalf("/params failed: %v", err)
		}
		if !strings.Contains(out.String(), "Hugging Face Model Parameters:") {
			t.Fatalf("unexpected /params output: %s", out.String())
		}

		// Execute /tokens and /cost
		out.Reset()
		sess3.TotalInputTokens = 150
		sess3.TotalOutputTokens = 50
		sess3.TotalTokens = 200
		sess3.EstimatedCost = 0.0012
		handled, _, err = HandleSlashCommand(context.Background(), "/tokens", sess3, &out)
		if err != nil || !handled || !strings.Contains(out.String(), "200") {
			t.Fatalf("/tokens failed: err=%v, out=%s", err, out.String())
		}

		out.Reset()
		handled, _, err = HandleSlashCommand(context.Background(), "/cost", sess3, &out)
		if err != nil || !handled || !strings.Contains(out.String(), "$0.0012") {
			t.Fatalf("/cost failed: err=%v, out=%s", err, out.String())
		}

		// Test conversation history management: /compact and /clear
		sess3.Messages = []client.ChatMessage{
			{Role: "user", Content: "Message 1"},
			{Role: "assistant", Content: "Response 1"},
			{Role: "user", Content: "Message 2"},
			{Role: "assistant", Content: "Response 2"},
			{Role: "user", Content: "Message 3"},
			{Role: "assistant", Content: "Response 3"},
		}

		out.Reset()
		handled, _, err = HandleSlashCommand(context.Background(), "/compact", sess3, &out)
		if err != nil || !handled {
			t.Fatalf("/compact failed: %v", err)
		}
		if len(sess3.Messages) >= 6 {
			t.Fatalf("expected compacted messages, got %d", len(sess3.Messages))
		}

		out.Reset()
		handled, _, err = HandleSlashCommand(context.Background(), "/clear", sess3, &out)
		if err != nil || !handled || len(sess3.Messages) != 0 {
			t.Fatalf("/clear failed: remaining messages = %d", len(sess3.Messages))
		}
	})
}
