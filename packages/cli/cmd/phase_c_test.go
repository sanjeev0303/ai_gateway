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

// TestPhaseC_SlashCommandStateTransitions verifies slash commands update state, enforce capabilities, and reject unauthorized actions.
func TestPhaseC_SlashCommandStateTransitions(t *testing.T) {
	registry := NewDefaultCommandRegistry()

	t.Run("SafetyCommand_GeminiSupportAndBounds", func(t *testing.T) {
		session := &SessionState{
			Registry: registry,
			Provider: "gemini",
			Model:    "gemini-1.5-pro",
		}
		var out bytes.Buffer

		// Test valid presets
		cmd := registry.Find("/safety")
		if cmd == nil {
			t.Fatal("/safety command not found")
		}

		presets := []struct {
			arg      string
			expected string
		}{
			{"standard", "BLOCK_MEDIUM_AND_ABOVE"},
			{"strict", "BLOCK_LOW_AND_ABOVE"},
			{"permissive", "BLOCK_ONLY_HIGH"},
			{"off", "BLOCK_NONE"},
		}

		for _, tc := range presets {
			out.Reset()
			err := cmd.Execute(context.Background(), session, []string{tc.arg}, &out)
			if err != nil {
				t.Fatalf("unexpected error setting safety %s: %v", tc.arg, err)
			}
			if session.SafetySettings != tc.expected {
				t.Fatalf("expected safety setting %q, got %q", tc.expected, session.SafetySettings)
			}
		}

		// Test invalid preset
		err := cmd.Execute(context.Background(), session, []string{"unsupported_level"}, &out)
		if err == nil {
			t.Fatal("expected error for invalid safety preset, got nil")
		}

		// Test rejection on non-Gemini provider
		session.Provider = "openai"
		err = cmd.Execute(context.Background(), session, []string{"strict"}, &out)
		if err == nil {
			t.Fatal("expected error when setting safety on non-Gemini provider, got nil")
		}
		if !strings.Contains(err.Error(), "only supported for Google Gemini") {
			t.Fatalf("unexpected error message: %v", err)
		}

		// Test rejection on Gemini embedding model
		session.Provider = "gemini"
		session.Model = "text-embedding-004"
		err = cmd.Execute(context.Background(), session, []string{"strict"}, &out)
		if err == nil {
			t.Fatal("expected error on embedding model, got nil")
		}
	})

	t.Run("EndpointCommand_SSRFProtectionAndWhitelisting", func(t *testing.T) {
		session := &SessionState{
			Registry: registry,
			Provider: "huggingface",
			Model:    "meta-llama/Llama-3-70b-instruct",
		}
		cmd := registry.Find("/endpoint")
		if cmd == nil {
			t.Fatal("/endpoint command not found")
		}

		var out bytes.Buffer

		// Valid Hugging Face dedicated inference endpoint
		validEp := "https://custom-llama3.us-east-1.aws.endpoints.huggingface.cloud"
		err := cmd.Execute(context.Background(), session, []string{validEp}, &out)
		if err != nil {
			t.Fatalf("unexpected error for valid HF endpoint: %v", err)
		}
		if session.HFEndpoint != validEp {
			t.Fatalf("expected HFEndpoint %q, got %q", validEp, session.HFEndpoint)
		}

		// Valid Hub router endpoint
		validHub := "https://router.huggingface.co/v1"
		err = cmd.Execute(context.Background(), session, []string{validHub}, &out)
		if err != nil {
			t.Fatalf("unexpected error for valid hub endpoint: %v", err)
		}

		// Revert / reset
		err = cmd.Execute(context.Background(), session, []string{"reset"}, &out)
		if err != nil {
			t.Fatalf("unexpected error resetting endpoint: %v", err)
		}
		if session.HFEndpoint != "" {
			t.Fatalf("expected empty HFEndpoint after reset, got %q", session.HFEndpoint)
		}

		// SSRF and unauthorized targets must fail
		maliciousTargets := []string{
			"http://custom.endpoints.huggingface.cloud", // HTTP not allowed
			"http://169.254.169.254/latest/meta-data",   // AWS / cloud metadata
			"https://127.0.0.1:8080/admin",              // Loopback IP
			"https://10.0.0.5/api",                      // Private RFC1918
			"https://192.168.1.1/internal",              // Private RFC1918
			"https://metadata.google.internal/compute",  // GCP metadata
			"https://localhost:443",                     // Localhost
			"https://attacker-controlled-site.com/api",  // Non-HF domain
		}

		for _, target := range maliciousTargets {
			err = cmd.Execute(context.Background(), session, []string{target}, &out)
			if err == nil {
				t.Fatalf("expected SSRF / unauthorized endpoint error for %q, but got nil", target)
			}
		}

		// Non-Hugging Face provider should reject endpoint command
		session.Provider = "anthropic"
		err = cmd.Execute(context.Background(), session, []string{validEp}, &out)
		if err == nil {
			t.Fatal("expected error when setting endpoint on anthropic provider, got nil")
		}
	})

	t.Run("ModelCommand_PermissionValidationAndProviderSwitch", func(t *testing.T) {
		session := &SessionState{
			Registry:            registry,
			Provider:            "anthropic",
			Model:               "claude-3-7-sonnet-20250219",
			AuthorizedProviders: []string{"anthropic", "gemini"},
			PermittedModels: []client.Model{
				{ID: "claude-3-7-sonnet-20250219", Provider: "anthropic"},
				{ID: "gemini-1.5-pro", Provider: "gemini"},
			},
		}
		cmd := registry.Find("/model")
		if cmd == nil {
			t.Fatal("/model command not found")
		}

		var out bytes.Buffer

		// Switch to permitted model
		err := cmd.Execute(context.Background(), session, []string{"gemini-1.5-pro"}, &out)
		if err != nil {
			t.Fatalf("unexpected error switching model: %v", err)
		}
		if session.Model != "gemini-1.5-pro" {
			t.Fatalf("expected active model gemini-1.5-pro, got %s", session.Model)
		}
		if session.Provider != "gemini" {
			t.Fatalf("expected active provider gemini, got %s", session.Provider)
		}

		// Attempt to select unauthorized model
		err = cmd.Execute(context.Background(), session, []string{"gpt-4o"}, &out)
		if err == nil {
			t.Fatal("expected error selecting unpermitted model, got nil")
		}

		// Rejection message
		if !strings.Contains(err.Error(), "not permitted") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("ClearAndUsage_TokenAttributionIntegrity", func(t *testing.T) {
		session := &SessionState{
			Registry: registry,
			Provider: "gemini",
			Model:    "gemini-1.5-pro",
		}

		// Record reported usage
		session.RecordUsage(&client.RequestMetadata{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
			UsageSource:  "reported",
		})

		// Record estimated usage
		session.RecordUsage(&client.RequestMetadata{
			InputTokens:  20,
			OutputTokens: 10,
			TotalTokens:  30,
			UsageSource:  "estimated",
		})

		if session.TotalTokens != 180 {
			t.Fatalf("expected total tokens 180, got %d", session.TotalTokens)
		}
		if session.ReportedTotalTokens != 150 {
			t.Fatalf("expected reported tokens 150, got %d", session.ReportedTotalTokens)
		}
		if session.EstimatedTokens != 30 {
			t.Fatalf("expected estimated tokens 30, got %d", session.EstimatedTokens)
		}

		var out bytes.Buffer
		usageCmd := registry.Find("/usage")
		err := usageCmd.Execute(context.Background(), session, nil, &out)
		if err != nil {
			t.Fatalf("unexpected error executing /usage: %v", err)
		}
		usageOut := out.String()
		if !strings.Contains(usageOut, "Provider-Reported: 150") {
			t.Fatalf("missing provider reported tokens in /usage: %s", usageOut)
		}
		if !strings.Contains(usageOut, "Locally-Estimated: 30") {
			t.Fatalf("missing locally estimated tokens in /usage: %s", usageOut)
		}

		// Cost command has disclaimer
		out.Reset()
		costCmd := registry.Find("/cost")
		err = costCmd.Execute(context.Background(), session, nil, &out)
		if err != nil {
			t.Fatalf("unexpected error executing /cost: %v", err)
		}
		costOut := out.String()
		if !strings.Contains(costOut, "Stripe / Razorpay") {
			t.Fatalf("cost command missing payment disclaimer: %s", costOut)
		}

		// Clear command wipes history and all counters
		out.Reset()
		clearCmd := registry.Find("/clear")
		err = clearCmd.Execute(context.Background(), session, nil, &out)
		if err != nil {
			t.Fatalf("unexpected error executing /clear: %v", err)
		}
		if session.TotalTokens != 0 || session.ReportedTotalTokens != 0 || session.EstimatedTokens != 0 {
			t.Fatalf("clear did not reset all token counters: total=%d, reported=%d, estimated=%d",
				session.TotalTokens, session.ReportedTotalTokens, session.EstimatedTokens)
		}
	})
}

// TestPhaseC_RequestPayloadSerialization verifies gateway request body serialization and header injection.
func TestPhaseC_RequestPayloadSerialization(t *testing.T) {
	t.Run("Gemini_SafetySettingsIncluded", func(t *testing.T) {
		var capturedBody map[string]interface{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "chatcmpl-1",
				"choices": [{"message": {"role": "assistant", "content": "Gemini response"}}],
				"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
			}`))
		}))
		defer srv.Close()

		c := client.New(srv.URL, "test-key", "")
		ctx := context.Background()

		_, meta, err := c.SendRequest(ctx, client.RequestPayload{
			Model:          "gemini-1.5-flash",
			Prompt:         "Test prompt",
			SafetySettings: "BLOCK_LOW_AND_ABOVE",
		})
		if err != nil {
			t.Fatalf("send request failed: %v", err)
		}
		if meta.UsageSource != "reported" {
			t.Fatalf("expected usage source reported, got %s", meta.UsageSource)
		}

		// Inspect serialized payload
		ss, ok := capturedBody["safety_settings"].([]interface{})
		if !ok || len(ss) != 4 {
			t.Fatalf("expected 4 safety settings categories, got %v", capturedBody["safety_settings"])
		}
		firstCat := ss[0].(map[string]interface{})
		if firstCat["threshold"] != "BLOCK_LOW_AND_ABOVE" {
			t.Fatalf("expected threshold BLOCK_LOW_AND_ABOVE, got %v", firstCat["threshold"])
		}
	})

	t.Run("HuggingFace_EndpointHeaderAndBodyInjected", func(t *testing.T) {
		var capturedHeader string
		var capturedBody map[string]interface{}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedHeader = r.Header.Get("x-hf-endpoint")
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "hf-1",
				"choices": [{"message": {"role": "assistant", "content": "HF response"}}],
				"usage": {"prompt_tokens": 8, "completion_tokens": 4, "total_tokens": 12}
			}`))
		}))
		defer srv.Close()

		c := client.New(srv.URL, "test-key", "")
		ctx := context.Background()

		hfURL := "https://my-model.endpoints.huggingface.cloud"
		_, _, err := c.SendRequest(ctx, client.RequestPayload{
			Model:      "meta-llama/Meta-Llama-3-8B-Instruct",
			Prompt:     "Test HF",
			HFEndpoint: hfURL,
		})
		if err != nil {
			t.Fatalf("send request failed: %v", err)
		}

		if capturedHeader != hfURL {
			t.Fatalf("expected x-hf-endpoint header %q, got %q", hfURL, capturedHeader)
		}
		if capturedBody["endpoint"] != hfURL {
			t.Fatalf("expected endpoint field in JSON body %q, got %v", hfURL, capturedBody["endpoint"])
		}
	})

	t.Run("OpenAI_ReasoningEffortAndTemperatureGating", func(t *testing.T) {
		var capturedBody map[string]interface{}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "o1-1",
				"choices": [{"message": {"role": "assistant", "content": "o1 response"}}],
				"usage": {"prompt_tokens": 5, "completion_tokens": 5, "total_tokens": 10}
			}`))
		}))
		defer srv.Close()

		c := client.New(srv.URL, "test-key", "")
		ctx := context.Background()

		tempVal := 0.7
		_, _, err := c.SendRequest(ctx, client.RequestPayload{
			Model:           "o1-preview",
			Prompt:          "Reasoning test",
			ReasoningEffort: "high",
			Temperature:     &tempVal,
		})
		if err != nil {
			t.Fatalf("send request failed: %v", err)
		}

		// o1 must suppress temperature to avoid OpenAI 400 Bad Request
		if _, hasTemp := capturedBody["temperature"]; hasTemp {
			t.Fatalf("temperature must be suppressed for o1 reasoning model, but was included: %v", capturedBody["temperature"])
		}
		if capturedBody["reasoning_effort"] != "high" {
			t.Fatalf("expected reasoning_effort 'high', got %v", capturedBody["reasoning_effort"])
		}
		// safety_settings must not be present on OpenAI requests
		if _, hasSafety := capturedBody["safety_settings"]; hasSafety {
			t.Fatalf("safety_settings should not be sent to OpenAI")
		}
	})
}

// TestPhaseC_SingleShotAndInteractiveConsistency verifies consistency between interactive and single-shot flag-based execution paths.
func TestPhaseC_SingleShotAndInteractiveConsistency(t *testing.T) {
	registry := NewDefaultCommandRegistry()

	// 1. Verify that single-shot and interactive state handling produce identical RequestPayload structures
	cfg := Config{
		GatewayURL: "https://gateway.example.com",
		Credential: "sk-ant-test",
	}

	interactiveSession := NewSessionState(cfg, "gemini-1.5-pro", "proj-123", true, 1024)
	interactiveSession.Registry = registry

	// Execute interactive slash commands
	var out bytes.Buffer
	safetyCmd := registry.Find("/safety")
	if err := safetyCmd.Execute(context.Background(), interactiveSession, []string{"strict"}, &out); err != nil {
		t.Fatalf("failed interactive /safety: %v", err)
	}

	// Build interactive payload
	interactivePayload := client.RequestPayload{
		Model:          interactiveSession.Model,
		Prompt:         "Hello world",
		Stream:         interactiveSession.Stream,
		MaxTokens:      interactiveSession.MaxTokens,
		ProjectID:      interactiveSession.ProjectID,
		SafetySettings: interactiveSession.SafetySettings,
		HFEndpoint:     interactiveSession.HFEndpoint,
	}

	// Simulate single-shot request payload with identical settings
	singleShotSession := NewSessionState(cfg, "gemini-1.5-pro", "proj-123", true, 1024)
	singleShotSession.SafetySettings = "BLOCK_LOW_AND_ABOVE"

	singleShotPayload := client.RequestPayload{
		Model:          singleShotSession.Model,
		Prompt:         "Hello world",
		Stream:         singleShotSession.Stream,
		MaxTokens:      singleShotSession.MaxTokens,
		ProjectID:      singleShotSession.ProjectID,
		SafetySettings: singleShotSession.SafetySettings,
		HFEndpoint:     singleShotSession.HFEndpoint,
	}

	// Compare payloads
	if interactivePayload.Model != singleShotPayload.Model ||
		interactivePayload.SafetySettings != singleShotPayload.SafetySettings ||
		interactivePayload.Stream != singleShotPayload.Stream ||
		interactivePayload.MaxTokens != singleShotPayload.MaxTokens ||
		interactivePayload.ProjectID != singleShotPayload.ProjectID {
		t.Fatalf("interactive and single-shot payloads differ:\nInteractive: %+v\nSingleShot:  %+v",
			interactivePayload, singleShotPayload)
	}

	// 2. Verify single-shot SSRF validation consistency with interactive validation
	badTarget := "https://127.0.0.1:8080/admin"
	errInteractive := registry.Find("/endpoint").Execute(context.Background(), &SessionState{Provider: "huggingface"}, []string{badTarget}, &out)
	errSingleShot := ValidateHFEndpoint(badTarget)

	if errInteractive == nil || errSingleShot == nil {
		t.Fatalf("expected both interactive and single-shot paths to reject SSRF endpoint %q", badTarget)
	}
}
