package client

import "testing"

func TestUsesChatCompletions(t *testing.T) {
	cases := map[string]bool{
		"claude-sonnet-4-5":                true,
		"gpt-4o":                           true,
		"gemini-2.5-flash":                 true,
		"mistral-small-latest":             true,
		"meta-llama/Llama-3.1-8B-Instruct": true,
	}
	for model := range cases {
		want := model != "claude-sonnet-4-5"
		if got := usesChatCompletions(model); got != want {
			t.Errorf("usesChatCompletions(%q) = %v, want %v", model, got, want)
		}
	}
}

func TestBuildRequestBody_SamplingKnobs(t *testing.T) {
	c := &GatewayClient{}
	tp := 0.9
	tk := 40
	seed := 7
	temp := 0.5

	// Anthropic (/v1/messages): native knobs; thinking suppresses temperature.
	claude := c.buildRequestBody(RequestPayload{
		Model:          "claude-sonnet-4-5",
		Prompt:         "hi",
		MaxTokens:      2048,
		Temperature:    &temp,
		TopP:           &tp,
		TopK:           &tk,
		StopSequences:  []string{"STOP"},
		ThinkingBudget: 1024,
	}, false, usesChatCompletions("claude-sonnet-4-5"))

	if claude["top_p"] != 0.9 {
		t.Errorf("claude top_p: got %v", claude["top_p"])
	}
	if claude["top_k"] != 40 {
		t.Errorf("claude top_k: got %v", claude["top_k"])
	}
	if _, ok := claude["stop_sequences"]; !ok {
		t.Errorf("claude should emit stop_sequences")
	}
	th, ok := claude["thinking"].(map[string]any)
	if !ok || th["budget_tokens"] != 1024 {
		t.Errorf("claude thinking not emitted correctly: %v", claude["thinking"])
	}
	if _, ok := claude["temperature"]; ok {
		t.Errorf("claude must omit temperature while thinking is on")
	}

	// OpenAI (/v1/chat/completions): seed + stop + top_p.
	openai := c.buildRequestBody(RequestPayload{
		Model:         "gpt-4o",
		Prompt:        "hi",
		MaxTokens:     256,
		TopP:          &tp,
		StopSequences: []string{"END"},
		Seed:          &seed,
	}, false, usesChatCompletions("gpt-4o"))
	if openai["seed"] != 7 {
		t.Errorf("openai seed: got %v", openai["seed"])
	}
	if _, ok := openai["stop"]; !ok {
		t.Errorf("openai should emit stop")
	}
	if openai["top_p"] != 0.9 {
		t.Errorf("openai top_p: got %v", openai["top_p"])
	}
	if _, ok := openai["stop_sequences"]; ok {
		t.Errorf("openai must use stop, not stop_sequences")
	}

	// Gemini rides the OpenAI surface but takes no seed.
	gemini := c.buildRequestBody(RequestPayload{
		Model:     "gemini-2.0-flash",
		Prompt:    "hi",
		MaxTokens: 256,
		Seed:      &seed,
	}, false, usesChatCompletions("gemini-2.0-flash"))
	if _, ok := gemini["seed"]; ok {
		t.Errorf("gemini must not emit seed")
	}
}
