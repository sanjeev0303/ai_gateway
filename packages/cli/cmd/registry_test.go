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

func TestCommandRegistry_ProviderSpecificFiltering(t *testing.T) {
	reg := NewDefaultCommandRegistry()

	// 1. Anthropic Session
	anthropicSess := &SessionState{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		Registry: reg,
	}
	anthropicCmds := reg.AvailableCommands(anthropicSess)
	var anthropicNames []string
	for _, c := range anthropicCmds {
		anthropicNames = append(anthropicNames, c.Name)
	}
	assertContains(t, anthropicNames, "/max-tokens", "anthropic session should have /max-tokens")
	assertContains(t, anthropicNames, "/temp", "anthropic session should have /temp")
	assertContains(t, anthropicNames, "/compact", "anthropic session should have /compact")
	assertNotContains(t, anthropicNames, "/reasoning", "anthropic session must not have /reasoning")
	assertNotContains(t, anthropicNames, "/safety", "anthropic session must not have /safety")
	assertNotContains(t, anthropicNames, "/endpoint", "anthropic session must not have /endpoint")

	// 2. Gemini Session
	geminiSess := &SessionState{
		Provider: "gemini",
		Model:    "gemini-1.5-flash",
		Registry: reg,
	}
	geminiCmds := reg.AvailableCommands(geminiSess)
	var geminiNames []string
	for _, c := range geminiCmds {
		geminiNames = append(geminiNames, c.Name)
	}
	assertContains(t, geminiNames, "/safety", "gemini session should have /safety")
	assertContains(t, geminiNames, "/temp", "gemini session should have /temp")
	assertNotContains(t, geminiNames, "/max-tokens", "gemini session must not have /max-tokens")
	assertNotContains(t, geminiNames, "/reasoning", "gemini session must not have /reasoning")

	// 3. OpenAI Session with standard model (gpt-4o) - no reasoning capability
	openaiStandardSess := &SessionState{
		Provider: "openai",
		Model:    "gpt-4o",
		Registry: reg,
	}
	openaiStandardCmds := reg.AvailableCommands(openaiStandardSess)
	var standardNames []string
	for _, c := range openaiStandardCmds {
		standardNames = append(standardNames, c.Name)
	}
	assertNotContains(t, standardNames, "/reasoning", "standard model gpt-4o must not have /reasoning")
	assertContains(t, standardNames, "/temp", "openai session should have /temp")

	// 4. OpenAI Session with reasoning model (o1) - reasoning capability enabled
	openaiReasoningSess := &SessionState{
		Provider: "openai",
		Model:    "o1-preview",
		Registry: reg,
	}
	openaiReasoningCmds := reg.AvailableCommands(openaiReasoningSess)
	var reasoningNames []string
	for _, c := range openaiReasoningCmds {
		reasoningNames = append(reasoningNames, c.Name)
	}
	assertContains(t, reasoningNames, "/reasoning", "reasoning model o1-preview must have /reasoning")

	// 5. Hugging Face Session
	hfSess := &SessionState{
		Provider: "huggingface",
		Model:    "meta-llama/Llama-3.3-70B-Instruct",
		Registry: reg,
	}
	hfCmds := reg.AvailableCommands(hfSess)
	var hfNames []string
	for _, c := range hfCmds {
		hfNames = append(hfNames, c.Name)
	}
	assertContains(t, hfNames, "/endpoint", "hugging face session should have /endpoint")
	assertContains(t, hfNames, "/params", "hugging face session should have /params")
	assertNotContains(t, hfNames, "/safety", "hugging face session must not have /safety")
	assertNotContains(t, hfNames, "/max-tokens", "hugging face session must not have /max-tokens")
}

func TestCommandRegistry_SlashMenuRendering(t *testing.T) {
	reg := NewDefaultCommandRegistry()

	sess := &SessionState{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		Registry: reg,
	}

	var out bytes.Buffer
	// Typing "/" renders the menu
	handled, shouldExit, err := HandleSlashCommand(context.Background(), "/", sess, &out)
	if err != nil || !handled || shouldExit {
		t.Fatalf("typing / should render menu: err=%v, handled=%v, shouldExit=%v", err, handled, shouldExit)
	}

	menuStr := out.String()
	if !strings.Contains(menuStr, "Sentinel CLI Slash Commands:") {
		t.Errorf("expected header 'Sentinel CLI Slash Commands:', got:\n%s", menuStr)
	}
	if !strings.Contains(menuStr, "[ANTHROPIC Specific Commands]") {
		t.Errorf("expected anthropic section in menu, got:\n%s", menuStr)
	}
	if !strings.Contains(menuStr, "[Common Sentinel Commands]") {
		t.Errorf("expected common commands section in menu, got:\n%s", menuStr)
	}
	if !strings.Contains(menuStr, "/compact") {
		t.Errorf("expected /compact in menu, got:\n%s", menuStr)
	}
}

func TestCommandRegistry_ConversationCompaction(t *testing.T) {
	ctx := context.Background()

	// Mock gateway that returns a summary
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"id": "msg_summary_123",
			"content": []map[string]interface{}{
				{"type": "text", "text": "User discussed AI Gateway capabilities and configured rate limits."},
			},
			"usage": map[string]interface{}{
				"input_tokens":  85,
				"output_tokens": 20,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	sess := &SessionState{
		GatewayURL: mockServer.URL,
		APIKey:     "aigw_test_key",
		Provider:   "anthropic",
		Model:      "claude-sonnet-4-5",
		Client:     client.New(mockServer.URL, "aigw_test_key", ""),
		Registry:   NewDefaultCommandRegistry(),
	}

	// 1. Compaction with <= 2 messages -> should report no compaction needed
	sess.Messages = []client.ChatMessage{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}
	var out bytes.Buffer
	handled, _, err := HandleSlashCommand(ctx, "/compact", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/compact failed: %v", err)
	}
	if !strings.Contains(out.String(), "Conversation is already compact") {
		t.Errorf("expected already compact message, got: %s", out.String())
	}
	if len(sess.Messages) != 2 {
		t.Errorf("messages count changed unexpectedly: %d", len(sess.Messages))
	}

	// 2. Compaction with > 2 messages -> should invoke gateway summarization
	sess.Messages = []client.ChatMessage{
		{Role: "user", Content: "Tell me about Sentinel AI Gateway"},
		{Role: "assistant", Content: "Sentinel is an AI governance platform..."},
		{Role: "user", Content: "Does it support Anthropic and Gemini?"},
		{Role: "assistant", Content: "Yes, it supports Anthropic, OpenAI, Gemini, and Hugging Face."},
		{Role: "user", Content: "How do I configure rate limits?"},
	}

	out.Reset()
	handled, _, err = HandleSlashCommand(ctx, "/compact", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/compact failed: %v", err)
	}

	if !strings.Contains(out.String(), "Compaction complete") {
		t.Errorf("expected compaction complete, got: %s", out.String())
	}
	// Verify conversation messages were compacted down to summary + latest turn
	if len(sess.Messages) != 2 {
		t.Fatalf("expected 2 compacted messages, got %d", len(sess.Messages))
	}
	if !strings.Contains(sess.Messages[0].Content, "Compacted Conversation Context") {
		t.Errorf("expected summary in first message, got: %s", sess.Messages[0].Content)
	}
	if sess.Messages[1].Content != "How do I configure rate limits?" {
		t.Errorf("expected last message preserved, got: %s", sess.Messages[1].Content)
	}
}

func TestCommandRegistry_ClearSemantics(t *testing.T) {
	ctx := context.Background()
	sess := &SessionState{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		Registry: NewDefaultCommandRegistry(),
		Messages: []client.ChatMessage{
			{Role: "user", Content: "test 1"},
			{Role: "assistant", Content: "reply 1"},
		},
		TotalInputTokens:  500,
		TotalOutputTokens: 250,
		TotalTokens:       750,
		EstimatedCost:     0.015,
	}

	var out bytes.Buffer
	handled, _, err := HandleSlashCommand(ctx, "/clear", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/clear failed: %v", err)
	}

	if len(sess.Messages) != 0 {
		t.Errorf("expected 0 messages after /clear, got %d", len(sess.Messages))
	}
	if sess.TotalTokens != 0 || sess.EstimatedCost != 0.0 {
		t.Errorf("expected usage counters reset to 0, got tokens=%d cost=%f", sess.TotalTokens, sess.EstimatedCost)
	}
	if !strings.Contains(out.String(), "Conversation history cleared") {
		t.Errorf("unexpected /clear output: %s", out.String())
	}
}

func TestCommandRegistry_ProviderSpecificExecutionAndValidation(t *testing.T) {
	ctx := context.Background()
	reg := NewDefaultCommandRegistry()

	// 1. Anthropic /temp validation
	anthropicSess := &SessionState{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		Registry: reg,
	}
	var out bytes.Buffer
	_, _, err := HandleSlashCommand(ctx, "/temp 0.7", anthropicSess, &out)
	if err != nil {
		t.Fatalf("/temp 0.7 failed: %v", err)
	}
	if anthropicSess.Temperature == nil || *anthropicSess.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", anthropicSess.Temperature)
	}

	// Anthropic temperature > 1.0 is invalid
	_, _, err = HandleSlashCommand(ctx, "/temp 1.5", anthropicSess, &out)
	if err == nil {
		t.Errorf("expected error for temperature 1.5 on Anthropic, got nil")
	}

	// 2. OpenAI /reasoning command
	openaiSess := &SessionState{
		Provider: "openai",
		Model:    "o1-preview",
		Registry: reg,
	}
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/reasoning high", openaiSess, &out)
	if err != nil {
		t.Fatalf("/reasoning high failed: %v", err)
	}
	if openaiSess.ReasoningEffort != "high" {
		t.Errorf("expected reasoning effort high, got %s", openaiSess.ReasoningEffort)
	}

	// Invalid reasoning effort
	_, _, err = HandleSlashCommand(ctx, "/reasoning ultra", openaiSess, &out)
	if err == nil {
		t.Errorf("expected error for invalid reasoning effort, got nil")
	}

	// 3. Gemini /safety command
	geminiSess := &SessionState{
		Provider: "gemini",
		Model:    "gemini-1.5-flash",
		Registry: reg,
	}
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/safety strict", geminiSess, &out)
	if err != nil {
		t.Fatalf("/safety strict failed: %v", err)
	}
	if geminiSess.SafetySettings != "BLOCK_LOW_AND_ABOVE" {
		t.Errorf("expected BLOCK_LOW_AND_ABOVE, got %s", geminiSess.SafetySettings)
	}

	// 4. Hugging Face /endpoint command
	hfSess := &SessionState{
		Provider: "huggingface",
		Model:    "meta-llama/Llama-3.3-70B-Instruct",
		Registry: reg,
	}
	out.Reset()
	_, _, err = HandleSlashCommand(ctx, "/endpoint https://my-endpoint.endpoints.huggingface.cloud", hfSess, &out)
	if err != nil {
		t.Fatalf("/endpoint failed: %v", err)
	}
	if hfSess.HFEndpoint != "https://my-endpoint.endpoints.huggingface.cloud" {
		t.Errorf("expected custom HF endpoint, got %s", hfSess.HFEndpoint)
	}

	// Invalid endpoint URL
	_, _, err = HandleSlashCommand(ctx, "/endpoint invalid-url-no-scheme", hfSess, &out)
	if err == nil {
		t.Errorf("expected error for invalid endpoint URL, got nil")
	}
}

func TestCommandRegistry_PermissionsCommand(t *testing.T) {
	ctx := context.Background()
	sess := &SessionState{
		APIKey:              "sgw_live_testkey12345678",
		Provider:            "anthropic",
		Model:               "claude-sonnet-4-5",
		AuthorizedProviders: []string{"anthropic"},
		PermittedModels: []client.Model{
			{ID: "claude-sonnet-4-5", Provider: "anthropic"},
			{ID: "claude-3-5-haiku", Provider: "anthropic"},
		},
		Registry: NewDefaultCommandRegistry(),
	}

	var out bytes.Buffer
	handled, _, err := HandleSlashCommand(ctx, "/permissions", sess, &out)
	if err != nil || !handled {
		t.Fatalf("/permissions failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Sentinel Credential Authorization Scope") {
		t.Errorf("expected authorization scope title, got: %s", output)
	}
	if !strings.Contains(output, "Bound Providers:   anthropic") {
		t.Errorf("expected bound provider anthropic, got: %s", output)
	}
	if !strings.Contains(output, "2 models accessible") {
		t.Errorf("expected 2 models accessible, got: %s", output)
	}
}

func assertContains(t *testing.T, list []string, item, msg string) {
	t.Helper()
	for _, x := range list {
		if x == item {
			return
		}
	}
	t.Fatalf("%s: expected item %q in %v", msg, item, list)
}

func assertNotContains(t *testing.T, list []string, item, msg string) {
	t.Helper()
	for _, x := range list {
		if x == item {
			t.Fatalf("%s: item %q should NOT be in %v", msg, item, list)
		}
	}
}
