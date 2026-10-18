package provider_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/provider/anthropic"
	"ai-governance-saas/backend/internal/infrastructure/provider/gemini"
	"ai-governance-saas/backend/internal/infrastructure/provider/huggingface"
	"ai-governance-saas/backend/internal/infrastructure/provider/openai"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
)

func TestProviderAdapter_InterfaceSatisfaction(t *testing.T) {
	adapters := []struct {
		name     string
		kind     string
		provider providerclient.Provider
		infer    providerclient.InferenceProvider
	}{
		{
			name:     "Anthropic Claude",
			kind:     domain.ProviderKindAnthropic,
			provider: anthropic.NewProviderAdapter(),
			infer:    anthropic.NewProviderAdapter(),
		},
		{
			name:     "OpenAI",
			kind:     domain.ProviderKindOpenAI,
			provider: openai.NewProviderAdapter(),
			infer:    openai.NewProviderAdapter(),
		},
		{
			name:     "Google Gemini",
			kind:     domain.ProviderKindGemini,
			provider: gemini.NewProviderAdapter(),
			infer:    gemini.NewProviderAdapter(),
		},
		{
			name:     "Hugging Face",
			kind:     domain.ProviderKindHuggingFace,
			provider: huggingface.NewProviderAdapter(),
			infer:    huggingface.NewProviderAdapter(),
		},
	}

	for _, tc := range adapters {
		t.Run(tc.name, func(t *testing.T) {
			if tc.provider == nil {
				t.Fatalf("%s does not implement provider.Provider", tc.name)
			}
			if tc.infer == nil {
				t.Fatalf("%s does not implement provider.InferenceProvider", tc.name)
			}
			if tc.infer.Kind() != tc.kind {
				t.Errorf("%s kind mismatch: got %s, want %s", tc.name, tc.infer.Kind(), tc.kind)
			}
		})
	}
}

func TestProviderAdapter_MetadataContract(t *testing.T) {
	adapters := []providerclient.InferenceProvider{
		anthropic.NewProviderAdapter(),
		openai.NewProviderAdapter(),
		gemini.NewProviderAdapter(),
		huggingface.NewProviderAdapter(),
	}

	for _, adapter := range adapters {
		t.Run(adapter.Name(), func(t *testing.T) {
			meta := adapter.Metadata()
			if meta.Kind == "" {
				t.Errorf("%s missing Kind in metadata", adapter.Name())
			}
			if meta.DisplayName == "" {
				t.Errorf("%s missing DisplayName in metadata", adapter.Name())
			}
			if meta.DefaultBaseURL == "" {
				t.Errorf("%s missing DefaultBaseURL in metadata", adapter.Name())
			}
			if meta.AuthHeaderFormat == "" {
				t.Errorf("%s missing AuthHeaderFormat in metadata", adapter.Name())
			}
			if len(meta.SupportedFeatures) == 0 {
				t.Errorf("%s reports 0 supported features", adapter.Name())
			}
			if len(meta.SupportedParameters) == 0 {
				t.Errorf("%s reports 0 supported parameters", adapter.Name())
			}
		})
	}
}

func TestProviderAdapter_CapabilityDiscoveryAndValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name               string
		adapter            providerclient.InferenceProvider
		model              string
		expectTools        bool
		expectVision       bool
		expectStreaming    bool
		expectContextLimit int
	}{
		{
			name:               "Anthropic Sonnet",
			adapter:            anthropic.NewProviderAdapter(),
			model:              "claude-sonnet-4-5",
			expectTools:        true,
			expectVision:       true,
			expectStreaming:    true,
			expectContextLimit: 200000,
		},
		{
			name:               "OpenAI GPT-4o",
			adapter:            openai.NewProviderAdapter(),
			model:              "gpt-4o",
			expectTools:        true,
			expectVision:       true,
			expectStreaming:    true,
			expectContextLimit: 128000,
		},
		{
			name:               "Gemini 1.5 Pro",
			adapter:            gemini.NewProviderAdapter(),
			model:              "gemini-1.5-pro",
			expectTools:        true,
			expectVision:       true,
			expectStreaming:    true,
			expectContextLimit: 2097152,
		},
		{
			name:               "Hugging Face Llama 3.3",
			adapter:            huggingface.NewProviderAdapter(),
			model:              "meta-llama/Llama-3.3-70B-Instruct",
			expectTools:        false,
			expectVision:       false,
			expectStreaming:    true,
			expectContextLimit: 131072,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps, err := tt.adapter.GetModelCapabilities(ctx, tt.model)
			if err != nil {
				t.Fatalf("failed to get capabilities for %s: %v", tt.model, err)
			}
			if caps.SupportsTools != tt.expectTools {
				t.Errorf("%s tools mismatch: got %v, want %v", tt.model, caps.SupportsTools, tt.expectTools)
			}
			if caps.SupportsVision != tt.expectVision {
				t.Errorf("%s vision mismatch: got %v, want %v", tt.model, caps.SupportsVision, tt.expectVision)
			}
			if caps.SupportsStreaming != tt.expectStreaming {
				t.Errorf("%s streaming mismatch: got %v, want %v", tt.model, caps.SupportsStreaming, tt.expectStreaming)
			}
			if caps.ContextLimit != tt.expectContextLimit {
				t.Errorf("%s context limit mismatch: got %d, want %d", tt.model, caps.ContextLimit, tt.expectContextLimit)
			}

			// Validate unsupported feature rejection
			if !tt.expectTools {
				badReq := providerclient.NormalizedRequest{
					Model: tt.model,
					Tools: []providerclient.ToolDefinition{{Name: "calculator"}},
				}
				if err := caps.ValidateRequest(badReq); !errors.Is(err, providerclient.ErrUnsupportedCapability) {
					t.Errorf("expected ErrUnsupportedCapability when tools requested on %s, got %v", tt.model, err)
				}
			}

			if !tt.expectVision {
				badReq := providerclient.NormalizedRequest{
					Model: tt.model,
					Messages: []providerclient.Message{
						{
							Role: providerclient.RoleUser,
							Parts: []providerclient.ContentPart{
								{Type: providerclient.ContentTypeImage, ImageURL: "https://example.com/image.png"},
							},
						},
					},
				}
				if err := caps.ValidateRequest(badReq); !errors.Is(err, providerclient.ErrUnsupportedCapability) {
					t.Errorf("expected ErrUnsupportedCapability when vision requested on %s, got %v", tt.model, err)
				}
			}
		})
	}
}

func TestProviderAdapter_StrictParameterRejection(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		adapter   providerclient.InferenceProvider
		model     string
		alienKey  string
		alienVal  any
	}{
		{
			name:     "Anthropic rejects reasoning_effort",
			adapter:  anthropic.NewProviderAdapter(),
			model:    "claude-3-5-sonnet-20241022",
			alienKey: "reasoning_effort",
			alienVal: "high",
		},
		{
			name:     "Anthropic rejects safety_settings",
			adapter:  anthropic.NewProviderAdapter(),
			model:    "claude-3-5-sonnet-20241022",
			alienKey: "safety_settings",
			alienVal: []string{"strict"},
		},
		{
			name:     "OpenAI rejects thinking",
			adapter:  openai.NewProviderAdapter(),
			model:    "gpt-4o",
			alienKey: "thinking",
			alienVal: map[string]any{"budget_tokens": 1024},
		},
		{
			name:     "OpenAI rejects safety_settings",
			adapter:  openai.NewProviderAdapter(),
			model:    "gpt-4o",
			alienKey: "safety_settings",
			alienVal: "standard",
		},
		{
			name:     "Gemini rejects thinking",
			adapter:  gemini.NewProviderAdapter(),
			model:    "gemini-1.5-pro",
			alienKey: "thinking",
			alienVal: true,
		},
		{
			name:     "Gemini rejects reasoning_effort",
			adapter:  gemini.NewProviderAdapter(),
			model:    "gemini-1.5-pro",
			alienKey: "reasoning_effort",
			alienVal: "medium",
		},
		{
			name:     "Hugging Face rejects reasoning_effort",
			adapter:  huggingface.NewProviderAdapter(),
			model:    "meta-llama/Llama-3.3-70B-Instruct",
			alienKey: "reasoning_effort",
			alienVal: "high",
		},
		{
			name:     "Hugging Face rejects safety_settings",
			adapter:  huggingface.NewProviderAdapter(),
			model:    "meta-llama/Llama-3.3-70B-Instruct",
			alienKey: "safety_settings",
			alienVal: "strict",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := providerclient.NormalizedRequest{
				Model: tc.model,
				Messages: []providerclient.Message{
					{Role: providerclient.RoleUser, Content: "Hello"},
				},
				CustomParameters: map[string]any{
					tc.alienKey: tc.alienVal,
				},
			}

			// 1. Execute must return ErrUnsupportedParameter
			_, err := tc.adapter.Execute(ctx, "test-api-key", req)
			if err == nil || !errors.Is(err, providerclient.ErrUnsupportedParameter) {
				t.Fatalf("expected ErrUnsupportedParameter on Execute, got %v", err)
			}

			// 2. Stream must return ErrUnsupportedParameter
			_, err = tc.adapter.Stream(ctx, "test-api-key", req)
			if err == nil || !errors.Is(err, providerclient.ErrUnsupportedParameter) {
				t.Fatalf("expected ErrUnsupportedParameter on Stream, got %v", err)
			}
		})
	}
}

func TestProviderAdapter_ParameterPreservationAndToolSemantics(t *testing.T) {
	ctx := context.Background()

	// 1. Anthropic Claude contract test
	t.Run("Anthropic Parameter Preservation and Tool Semantics", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/messages" {
				http.NotFound(w, r)
				return
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "msg_anthropic_test_01",
				"type":        "message",
				"role":        "assistant",
				"model":       "claude-3-7-sonnet-20250219",
				"content":     []map[string]any{{"type": "text", "text": "Result is 42"}},
				"stop_reason": "end_turn",
				"usage": map[string]any{
					"input_tokens":              25,
					"output_tokens":             12,
					"cache_read_input_tokens": 10,
				},
			})
		}))
		defer mockServer.Close()

		adapter := anthropic.NewCustomProviderAdapter(mockServer.URL, "2023-06-01")
		req := providerclient.NormalizedRequest{
			Model: "claude-3-7-sonnet-20250219",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "Calculate 6*7"},
				{
					Role: providerclient.RoleAssistant,
					ToolCalls: []providerclient.ToolCall{
						{ID: "call_abc_1", Type: "function", Name: "calculator", Arguments: `{"expr":"6*7"}`},
					},
				},
				{Role: providerclient.RoleTool, ToolCallID: "call_abc_1", Name: "calculator", Content: "42"},
			},
			CustomParameters: map[string]any{
				"thinking": map[string]any{"budget_tokens": float64(2048), "type": "enabled"},
				"top_k":    40,
				"metadata": map[string]any{"user_id": "user-sentinel-1"},
			},
		}

		resp, err := adapter.Execute(ctx, "mock-key", req)
		if err != nil {
			t.Fatalf("Anthropic Execute error: %v", err)
		}

		// Verify parameter preservation
		thinking, ok := receivedReq["thinking"].(map[string]any)
		if !ok || thinking["budget_tokens"] != float64(2048) {
			t.Errorf("thinking budget_tokens not preserved: %v", receivedReq["thinking"])
		}
		if receivedReq["top_k"] != float64(40) {
			t.Errorf("top_k not preserved: %v", receivedReq["top_k"])
		}
		meta, ok := receivedReq["metadata"].(map[string]any)
		if !ok || meta["user_id"] != "user-sentinel-1" {
			t.Errorf("metadata not preserved: %v", receivedReq["metadata"])
		}

		// Verify tool message semantics
		msgs, ok := receivedReq["messages"].([]any)
		if !ok || len(msgs) != 3 {
			t.Fatalf("expected 3 turns, got %d", len(msgs))
		}
		asstTurn := msgs[1].(map[string]any)
		if asstTurn["role"] != "assistant" {
			t.Errorf("expected assistant turn, got %v", asstTurn["role"])
		}
		toolResultTurn := msgs[2].(map[string]any)
		if toolResultTurn["role"] != "user" {
			t.Errorf("Anthropic tool result turn must have role 'user', got %v", toolResultTurn["role"])
		}

		// Verify normalized response
		if resp.ID != "msg_anthropic_test_01" {
			t.Errorf("expected ID msg_anthropic_test_01, got %s", resp.ID)
		}
		if resp.Usage.CachedTokens != 10 {
			t.Errorf("expected 10 cached tokens, got %d", resp.Usage.CachedTokens)
		}
	})

	// 2. OpenAI contract test
	t.Run("OpenAI Parameter Preservation and Tool Semantics", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" {
				http.NotFound(w, r)
				return
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    "chatcmpl_test_01",
				"model": "o1-preview",
				"choices": []map[string]any{
					{
						"message":       map[string]any{"role": "assistant", "content": "Done"},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     30,
					"completion_tokens": 15,
					"total_tokens":      45,
					"prompt_tokens_details": map[string]any{
						"cached_tokens": 8,
					},
					"completion_tokens_details": map[string]any{
						"reasoning_tokens": 6,
					},
				},
			})
		}))
		defer mockServer.Close()

		adapter := openai.NewCustomProviderAdapter(mockServer.URL, "2024-08-01-preview")
		req := providerclient.NormalizedRequest{
			Model: "o1-preview",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "Solve equation"},
				{
					Role: providerclient.RoleAssistant,
					ToolCalls: []providerclient.ToolCall{
						{ID: "call_xyz", Type: "function", Name: "solver", Arguments: `{"eq":"x+1=2"}`},
					},
				},
				{Role: providerclient.RoleTool, ToolCallID: "call_xyz", Content: "x=1"},
			},
			CustomParameters: map[string]any{
				"reasoning_effort":    "high",
				"frequency_penalty":   0.5,
				"seed":                42,
				"parallel_tool_calls": false,
			},
		}

		resp, err := adapter.Execute(ctx, "mock-key", req)
		if err != nil {
			t.Fatalf("OpenAI Execute error: %v", err)
		}

		// Verify parameter preservation
		if receivedReq["reasoning_effort"] != "high" {
			t.Errorf("reasoning_effort not preserved: %v", receivedReq["reasoning_effort"])
		}
		if receivedReq["frequency_penalty"] != 0.5 {
			t.Errorf("frequency_penalty not preserved: %v", receivedReq["frequency_penalty"])
		}
		if receivedReq["seed"] != float64(42) {
			t.Errorf("seed not preserved: %v", receivedReq["seed"])
		}
		if receivedReq["parallel_tool_calls"] != false {
			t.Errorf("parallel_tool_calls not preserved: %v", receivedReq["parallel_tool_calls"])
		}

		// Verify tool semantics
		msgs, ok := receivedReq["messages"].([]any)
		if !ok || len(msgs) != 3 {
			t.Fatalf("expected 3 turns, got %d", len(msgs))
		}
		toolTurn := msgs[2].(map[string]any)
		if toolTurn["role"] != "tool" || toolTurn["tool_call_id"] != "call_xyz" {
			t.Errorf("OpenAI tool turn mismatch: %v", toolTurn)
		}

		// Verify response metadata
		if resp.ID != "chatcmpl_test_01" {
			t.Errorf("expected ID chatcmpl_test_01, got %s", resp.ID)
		}
		if resp.Usage.CachedTokens != 8 {
			t.Errorf("expected 8 cached tokens, got %d", resp.Usage.CachedTokens)
		}
		if resp.Usage.ReasoningTokens != 6 {
			t.Errorf("expected 6 reasoning tokens, got %d", resp.Usage.ReasoningTokens)
		}
	})

	// 3. Google Gemini contract test
	t.Run("Gemini Parameter Preservation and Tool Semantics", func(t *testing.T) {
		var receivedReq map[string]any
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "generateContent") {
				http.NotFound(w, r)
				return
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedReq)

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modelVersion": "gemini-1.5-pro",
				"candidates": []map[string]any{
					{
						"content": map[string]any{
							"role": "model",
							"parts": []map[string]any{
								{"text": "Gemini tool handled"},
							},
						},
						"finishReason": "STOP",
					},
				},
				"usageMetadata": map[string]any{
					"promptTokenCount":        28,
					"candidatesTokenCount":    14,
					"totalTokenCount":         42,
					"cachedContentTokenCount": 7,
					"thoughtsTokenCount":      5,
				},
			})
		}))
		defer mockServer.Close()

		adapter := gemini.NewCustomProviderAdapter(mockServer.URL, "v1beta")
		req := providerclient.NormalizedRequest{
			Model: "gemini-1.5-pro",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "What is current weather?"},
				{Role: providerclient.RoleTool, ToolCallID: "get_weather", Name: "get_weather", Content: `{"temp":22}`},
			},
			CustomParameters: map[string]any{
				"safety_settings": []map[string]any{
					{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_LOW_AND_ABOVE"},
				},
				"top_k":           30,
				"candidate_count": 1,
			},
		}

		resp, err := adapter.Execute(ctx, "mock-key", req)
		if err != nil {
			t.Fatalf("Gemini Execute error: %v", err)
		}

		// Verify parameter preservation
		safety, ok := receivedReq["safetySettings"].([]any)
		if !ok || len(safety) == 0 {
			t.Fatalf("safetySettings not preserved: %v", receivedReq["safetySettings"])
		}
		genConfig, ok := receivedReq["generationConfig"].(map[string]any)
		if !ok || genConfig["topK"] != float64(30) || genConfig["candidateCount"] != float64(1) {
			t.Errorf("generationConfig not preserved: %v", receivedReq["generationConfig"])
		}

		// Verify tool semantics
		contents, ok := receivedReq["contents"].([]any)
		if !ok || len(contents) != 2 {
			t.Fatalf("expected 2 contents turns, got %d", len(contents))
		}
		toolTurn := contents[1].(map[string]any)
		if toolTurn["role"] != "function" {
			t.Errorf("Gemini tool response role must be 'function', got %v", toolTurn["role"])
		}

		// Verify response metadata
		if !strings.Contains(resp.ID, "gemini-") {
			t.Errorf("expected gemini prefix ID, got %s", resp.ID)
		}
		if resp.Usage.CachedTokens != 7 {
			t.Errorf("expected 7 cached tokens, got %d", resp.Usage.CachedTokens)
		}
		if resp.Usage.ReasoningTokens != 5 {
			t.Errorf("expected 5 reasoning thoughts tokens, got %d", resp.Usage.ReasoningTokens)
		}
	})
}

func TestProviderAdapter_StreamingConversionContract(t *testing.T) {
	ctx := context.Background()

	// 1. Anthropic streaming
	t.Run("Anthropic Streaming Conversion", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected flusher")
			}

			events := []string{
				`{"type":"message_start","message":{"id":"msg_stream_100","model":"claude-3-5-sonnet","usage":{"input_tokens":18,"cache_read_input_tokens":6}}}` + "\n",
				`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"call_str_1","name":"lookup","input":{}}}` + "\n",
				`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"val\"}"}}` + "\n",
				`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Streaming chunk"}}` + "\n",
				`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":9}}` + "\n",
				`{"type":"message_stop"}` + "\n",
			}

			for _, ev := range events {
				_, _ = w.Write([]byte(ev))
				flusher.Flush()
			}
		}))
		defer mockServer.Close()

		adapter := anthropic.NewCustomProviderAdapter(mockServer.URL, "2023-06-01")
		stream, err := adapter.Stream(ctx, "mock-key", providerclient.NormalizedRequest{
			Model: "claude-3-5-sonnet-20241022",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "Stream test"},
			},
		})
		if err != nil {
			t.Fatalf("Anthropic Stream error: %v", err)
		}
		defer stream.Close()

		var chunks []*providerclient.StreamChunk
		for {
			chunk, err := stream.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("stream.Next error: %v", err)
			}
			chunks = append(chunks, chunk)
		}

		if len(chunks) == 0 {
			t.Fatal("expected non-empty chunks from Anthropic stream")
		}

		// First chunk carries message_start ID and prompt usage
		if chunks[0].ID != "msg_stream_100" {
			t.Errorf("expected ID msg_stream_100 on first chunk, got %s", chunks[0].ID)
		}
		if chunks[0].Usage == nil || chunks[0].Usage.PromptTokens != 18 || chunks[0].Usage.CachedTokens != 6 {
			t.Errorf("prompt usage not propagated on first chunk: %+v", chunks[0].Usage)
		}

		// Tool call chunk
		if len(chunks[1].ToolCalls) == 0 || chunks[1].ToolCalls[0].Name != "lookup" {
			t.Errorf("expected tool call on second chunk, got %+v", chunks[1].ToolCalls)
		}

		// Text delta chunk
		foundText := false
		for _, c := range chunks {
			if c.DeltaText == "Streaming chunk" {
				foundText = true
				break
			}
		}
		if !foundText {
			t.Errorf("expected 'Streaming chunk' delta text in chunks")
		}

		// Final chunk has finish reason and output usage
		last := chunks[len(chunks)-1]
		if last.FinishReason != "end_turn" {
			t.Errorf("expected finish_reason 'end_turn', got %s", last.FinishReason)
		}
		if last.Usage == nil || last.Usage.CompletionTokens != 9 {
			t.Errorf("expected completion tokens 9 on last chunk, got %+v", last.Usage)
		}
	})

	// 2. OpenAI streaming
	t.Run("OpenAI Streaming Conversion", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected flusher")
			}

			sseEvents := []string{
				`data: {"id":"chatcmpl_str_200","choices":[{"index":0,"delta":{"content":"OpenAI chunk"},"finish_reason":null}]}` + "\n\n",
				`data: {"id":"chatcmpl_str_200","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":6,"total_tokens":18,"prompt_tokens_details":{"cached_tokens":4},"completion_tokens_details":{"reasoning_tokens":3}}}` + "\n\n",
				`data: [DONE]` + "\n\n",
			}

			for _, ev := range sseEvents {
				_, _ = w.Write([]byte(ev))
				flusher.Flush()
			}
		}))
		defer mockServer.Close()

		adapter := openai.NewCustomProviderAdapter(mockServer.URL, "2024-08-01-preview")
		stream, err := adapter.Stream(ctx, "mock-key", providerclient.NormalizedRequest{
			Model: "gpt-4o",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "Stream test"},
			},
		})
		if err != nil {
			t.Fatalf("OpenAI Stream error: %v", err)
		}
		defer stream.Close()

		var chunks []*providerclient.StreamChunk
		for {
			chunk, err := stream.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("stream.Next error: %v", err)
			}
			chunks = append(chunks, chunk)
		}

		if len(chunks) < 2 {
			t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
		}
		if chunks[0].ID != "chatcmpl_str_200" || chunks[0].DeltaText != "OpenAI chunk" {
			t.Errorf("first chunk mismatch: %+v", chunks[0])
		}
		last := chunks[len(chunks)-1]
		if last.FinishReason != "stop" {
			t.Errorf("expected finish_reason 'stop', got %s", last.FinishReason)
		}
		if last.Usage == nil || last.Usage.CachedTokens != 4 || last.Usage.ReasoningTokens != 3 {
			t.Errorf("expected cached 4, reasoning 3 tokens on last chunk, got %+v", last.Usage)
		}
	})

	// 3. Gemini streaming
	t.Run("Gemini Streaming Conversion", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected flusher")
			}

			sseEvents := []string{
				`data: {"candidates":[{"content":{"parts":[{"text":"Gemini streaming"}]},"finishReason":null}]}` + "\n\n",
				`data: {"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":15,"candidatesTokenCount":8,"totalTokenCount":23,"cachedContentTokenCount":5,"thoughtsTokenCount":2}}` + "\n\n",
			}

			for _, ev := range sseEvents {
				_, _ = w.Write([]byte(ev))
				flusher.Flush()
			}
		}))
		defer mockServer.Close()

		adapter := gemini.NewCustomProviderAdapter(mockServer.URL, "v1beta")
		stream, err := adapter.Stream(ctx, "mock-key", providerclient.NormalizedRequest{
			Model: "gemini-1.5-pro",
			Messages: []providerclient.Message{
				{Role: providerclient.RoleUser, Content: "Stream test"},
			},
		})
		if err != nil {
			t.Fatalf("Gemini Stream error: %v", err)
		}
		defer stream.Close()

		var chunks []*providerclient.StreamChunk
		for {
			chunk, err := stream.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("stream.Next error: %v", err)
			}
			chunks = append(chunks, chunk)
		}

		if len(chunks) < 2 {
			t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
		}
		if chunks[0].DeltaText != "Gemini streaming" {
			t.Errorf("first chunk delta mismatch: %s", chunks[0].DeltaText)
		}
		last := chunks[len(chunks)-1]
		if last.FinishReason != "STOP" {
			t.Errorf("expected finish_reason 'STOP', got %s", last.FinishReason)
		}
		if last.Usage == nil || last.Usage.CachedTokens != 5 || last.Usage.ReasoningTokens != 2 {
			t.Errorf("expected cached 5, reasoning 2 tokens on last chunk, got %+v", last.Usage)
		}
	})
}

func TestProviderAdapter_EstimateTokensContract(t *testing.T) {
	ctx := context.Background()
	req := providerclient.NormalizedRequest{
		Model: "test-model",
		Messages: []providerclient.Message{
			{Role: providerclient.RoleUser, Content: "Sentinel AI Governance Gateway test prompt."},
		},
	}

	// 1. OpenAI and Hugging Face estimate locally
	localAdapters := []providerclient.InferenceProvider{
		openai.NewProviderAdapter(),
		huggingface.NewProviderAdapter(),
	}

	for _, adapter := range localAdapters {
		t.Run(adapter.Name(), func(t *testing.T) {
			usage, err := adapter.EstimateTokens(ctx, "test-key", req)
			if err != nil {
				t.Fatalf("%s EstimateTokens error: %v", adapter.Name(), err)
			}
			if usage.TotalTokens == 0 {
				t.Errorf("%s EstimateTokens returned 0 tokens", adapter.Name())
			}
		})
	}

	// 2. Anthropic & Gemini token count via mock servers
	anthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"input_tokens": 14})
	}))
	defer anthropicServer.Close()

	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"totalTokens": 18})
	}))
	defer geminiServer.Close()

	t.Run("Anthropic Claude Mock Token Count", func(t *testing.T) {
		adapter := anthropic.NewCustomProviderAdapter(anthropicServer.URL, "2023-06-01")
		usage, err := adapter.EstimateTokens(ctx, "mock-key", req)
		if err != nil {
			t.Fatalf("Anthropic EstimateTokens error: %v", err)
		}
		if usage.PromptTokens != 14 {
			t.Errorf("expected 14 tokens, got %d", usage.PromptTokens)
		}
	})

	t.Run("Google Gemini Mock Token Count", func(t *testing.T) {
		adapter := gemini.NewCustomProviderAdapter(geminiServer.URL, "v1beta")
		usage, err := adapter.EstimateTokens(ctx, "mock-key", req)
		if err != nil {
			t.Fatalf("Gemini EstimateTokens error: %v", err)
		}
		if usage.TotalTokens != 18 {
			t.Errorf("expected 18 tokens, got %d", usage.TotalTokens)
		}
	})
}
