package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDetectProvider(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{"meta-llama/Llama-3.3-70B-Instruct", "huggingface"},
		{"hf-mistral-7b", "huggingface"},
		{"gpt-4o", "openai"},
		{"gpt-4o-mini", "openai"},
		{"o1-preview", "openai"},
		{"claude-sonnet-4-5", "anthropic"},
		{"claude-3-5-haiku", "anthropic"},
		{"gemini-2.0-flash", "gemini"},
		{"gemini-1.5-pro", "gemini"},
		{"unknown-custom-model", "unknown"},
	}

	for _, tt := range tests {
		got := DetectProvider(tt.model)
		if got != tt.expected {
			t.Errorf("DetectProvider(%q) = %q, want %q", tt.model, got, tt.expected)
		}
	}
}

func TestHandleSlashCommands(t *testing.T) {
	ctx := context.Background()

	// Mock server for /v1/models
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			resp := map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "claude-sonnet-4-5", "provider": "anthropic", "display_name": "Claude 3.5 Sonnet"},
					{"id": "meta-llama/Llama-3.3-70B-Instruct", "provider": "huggingface", "display_name": "Llama 3.3 70B"},
					{"id": "gpt-4o", "provider": "openai", "display_name": "GPT-4o"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	cfg := Config{
		GatewayURL: mockServer.URL,
		Credential: "aigw_test_secret_key_12345",
		ProjectID:  "00000000-0000-0000-0000-000000000001",
	}

	sess := NewSessionState(cfg, "claude-sonnet-4-5", cfg.ProjectID, true, 1024)

	// 1. /help
	var out bytes.Buffer
	handled, shouldExit, err := HandleSlashCommand(ctx, "/help", sess, &out)
	if err != nil || !handled || shouldExit {
		t.Fatalf("/help failed: err=%v handled=%v shouldExit=%v", err, handled, shouldExit)
	}
	if !strings.Contains(out.String(), "Sentinel CLI Slash Commands:") {
		t.Errorf("expected /help output to list commands, got: %s", out.String())
	}

	// 2. /provider (no args)
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/provider", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/provider get failed: %v", err)
	}
	if !strings.Contains(out.String(), "Current provider: anthropic") {
		t.Errorf("unexpected /provider output: %s", out.String())
	}

	// 3. /provider huggingface
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/provider huggingface", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/provider set failed: %v", err)
	}
	if sess.Provider != "huggingface" || sess.Model != "meta-llama/Llama-3.3-70B-Instruct" {
		t.Errorf("expected provider huggingface and model meta-llama, got prov=%s model=%s", sess.Provider, sess.Model)
	}

	// 4. /provider invalid
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/provider unsupported_foo", sess, &out)
	if err == nil {
		t.Errorf("expected error for unsupported provider, got nil")
	}

	// 5. /stream toggle and explicit
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/stream", sess, &out)
	if err != nil || sess.Stream {
		t.Errorf("expected streaming toggled to false, got %v", sess.Stream)
	}
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/stream on", sess, &out)
	if err != nil || !sess.Stream {
		t.Errorf("expected streaming set to true, got %v", sess.Stream)
	}

	// 6. /model (query gateway)
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/model", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/model list failed: %v", err)
	}
	if !strings.Contains(out.String(), "meta-llama/Llama-3.3-70B-Instruct") || !strings.Contains(out.String(), "huggingface") {
		t.Errorf("expected model table with huggingface model, got: %s", out.String())
	}

	// 7. /model switch
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/model gpt-4o", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/model switch failed: %v", err)
	}
	if sess.Model != "gpt-4o" || sess.Provider != "openai" {
		t.Errorf("expected model gpt-4o and provider openai, got model=%s prov=%s", sess.Model, sess.Provider)
	}

	// 8. /tokens and /cost
	sess.TotalInputTokens = 1000
	sess.TotalOutputTokens = 500
	sess.TotalTokens = 1500
	sess.EstimatedCost = EstimateCost("gpt-4o", 1000, 500)

	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/tokens", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/tokens failed: %v", err)
	}
	if !strings.Contains(out.String(), "1500 tokens") {
		t.Errorf("expected tokens output to show 1500, got: %s", out.String())
	}

	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/cost", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/cost failed: %v", err)
	}
	if !strings.Contains(out.String(), "Session Cumulative Estimated Cost:") {
		t.Errorf("unexpected /cost output: %s", out.String())
	}

	// 9. /status
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/status", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/status failed: %v", err)
	}
	if !strings.Contains(out.String(), "Sentinel Gateway Session Status") || !strings.Contains(out.String(), "aigw...2345") {
		t.Errorf("unexpected /status output: %s", out.String())
	}

	// 10. /clear
	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/clear", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/clear failed: %v", err)
	}
	if sess.TotalTokens != 0 || sess.EstimatedCost != 0 {
		t.Errorf("expected counters reset to 0, got tokens=%d cost=%f", sess.TotalTokens, sess.EstimatedCost)
	}

	// 11. /exit and /quit
	out.Reset()
	handled, shouldExit, err = HandleSlashCommand(ctx, "/exit", sess, &out)
	if err != nil || !handled || !shouldExit {
		t.Fatalf("/exit failed: err=%v handled=%v shouldExit=%v", err, handled, shouldExit)
	}

	// 12. Non-slash command
	handled, shouldExit, err = HandleSlashCommand(ctx, "hello world", sess, &out)
	if handled || shouldExit || err != nil {
		t.Errorf("expected non-slash command to return false, got handled=%v", handled)
	}
}

func TestInteractiveSessionREPL(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		GatewayURL: "http://dummy",
		Credential: "aigw_test_secret_key_12345",
	}
	sess := NewSessionState(cfg, "claude-sonnet-4-5", "", true, 1024)

	// Feed simulated input stream: /provider huggingface \n /status \n /quit
	input := "/provider huggingface\n/status\n/quit\n"
	in := strings.NewReader(input)
	var out bytes.Buffer

	err := RunInteractiveSession(ctx, sess, in, &out)
	if err != nil {
		t.Fatalf("RunInteractiveSession failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Switched provider to huggingface") {
		t.Errorf("expected provider switch in output, got: %s", output)
	}
	if !strings.Contains(output, "Sentinel Gateway Session Status") {
		t.Errorf("expected status output in REPL, got: %s", output)
	}
	if !strings.Contains(output, "Exiting Sentinel session. Goodbye!") {
		t.Errorf("expected graceful goodbye, got: %s", output)
	}
}

func TestProviderBoundSessionScope(t *testing.T) {
	ctx := context.Background()

	// 1. Anthropic-only credential scope
	mockAnthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "claude-sonnet-4-5", "provider": "anthropic", "display_name": "Claude 3.5 Sonnet"},
				{"id": "claude-3-5-haiku", "provider": "anthropic", "display_name": "Claude 3.5 Haiku"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockAnthropicServer.Close()

	sessAnthropic := NewSessionState(Config{
		GatewayURL: mockAnthropicServer.URL,
		Credential: "aigw_anthropic_bound_key",
	}, "claude-sonnet-4-5", "", true, 1024)

	if len(sessAnthropic.AuthorizedProviders) != 1 || sessAnthropic.AuthorizedProviders[0] != "anthropic" {
		t.Fatalf("expected 1 authorized provider (anthropic), got %v", sessAnthropic.AuthorizedProviders)
	}

	var out bytes.Buffer
	// Trying to switch to openai should be rejected
	_, _, err := HandleSlashCommand(ctx, "/provider openai", sessAnthropic, &out)
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected error switching to unauthorized provider 'openai', got: %v", err)
	}

	// Trying to switch to unauthorized model should be rejected
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/model gpt-4o", sessAnthropic, &out)
	if err == nil || !strings.Contains(err.Error(), "not permitted") {
		t.Fatalf("expected error selecting unpermitted model 'gpt-4o', got: %v", err)
	}

	// Permitted model should succeed
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/model claude-3-5-haiku", sessAnthropic, &out)
	if err != nil {
		t.Fatalf("expected selecting permitted model to succeed, got: %v", err)
	}
	if sessAnthropic.Model != "claude-3-5-haiku" {
		t.Errorf("expected model to be claude-3-5-haiku, got %s", sessAnthropic.Model)
	}

	// 2. Gemini-bound session
	mockGeminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gemini-1.5-flash", "provider": "gemini", "display_name": "Gemini 1.5 Flash"},
				{"id": "gemini-2.0-flash", "provider": "gemini", "display_name": "Gemini 2.0 Flash"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockGeminiServer.Close()

	sessGemini := NewSessionState(Config{
		GatewayURL: mockGeminiServer.URL,
		Credential: "aigw_gemini_bound_key",
	}, "", "", true, 1024)

	if sessGemini.Provider != "gemini" {
		t.Errorf("expected Gemini-bound session to default to gemini provider, got %s", sessGemini.Provider)
	}
	if sessGemini.Model != "gemini-1.5-flash" {
		t.Errorf("expected Gemini-bound session to default to gemini-1.5-flash, got %s", sessGemini.Model)
	}

	// 3. Multi-provider credential (anthropic + gemini)
	mockMultiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "claude-sonnet-4-5", "provider": "anthropic", "display_name": "Claude 3.5 Sonnet"},
				{"id": "gemini-1.5-flash", "provider": "gemini", "display_name": "Gemini 1.5 Flash"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockMultiServer.Close()

	sessMulti := NewSessionState(Config{
		GatewayURL: mockMultiServer.URL,
		Credential: "aigw_multi_bound_key",
	}, "claude-sonnet-4-5", "", true, 1024)

	// Can switch to authorized gemini
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/provider gemini", sessMulti, &out)
	if err != nil {
		t.Fatalf("expected switching to authorized gemini provider to succeed, got: %v", err)
	}
	if sessMulti.Provider != "gemini" || sessMulti.Model != "gemini-1.5-flash" {
		t.Errorf("expected provider=gemini model=gemini-1.5-flash, got provider=%s model=%s", sessMulti.Provider, sessMulti.Model)
	}

	// Cannot switch to unauthorized openai
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/provider openai", sessMulti, &out)
	if err == nil || !strings.Contains(err.Error(), "not authorized") {
		t.Fatalf("expected error switching to unauthorized openai, got: %v", err)
	}
}
