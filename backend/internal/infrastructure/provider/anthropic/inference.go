package anthropic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

var _ provider.InferenceProvider = (*ProviderAdapter)(nil)

// Name returns the canonical provider name.
func (a *ProviderAdapter) Name() string {
	return "Anthropic Claude"
}

// Kind returns the provider identifier.
func (a *ProviderAdapter) Kind() string {
	return domain.ProviderKindAnthropic
}

// Metadata returns provider characteristics and feature availability.
func (a *ProviderAdapter) Metadata() provider.ProviderMetadata {
	return provider.ProviderMetadata{
		Kind:           domain.ProviderKindAnthropic,
		DisplayName:    "Anthropic Claude",
		DefaultBaseURL: defaultBaseURL,
		DocURL:         "https://docs.anthropic.com/en/api/messages",
		SupportedFeatures: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityMultimodal,
			provider.CapabilityThinking,
		},
		SupportedParameters: []string{
			"thinking",
			"top_k",
			"stop_sequences",
			"metadata",
			"tool_choice",
		},
		AuthHeaderFormat: "x-api-key: <token>",
	}
}

// GetModelCapabilities returns verified constraints and capabilities for Claude models.
func (a *ProviderAdapter) GetModelCapabilities(_ context.Context, modelID string) (provider.ModelCapabilities, error) {
	m := strings.ToLower(modelID)
	contextLimit := 200000
	maxOutput := 8192

	if strings.Contains(m, "haiku") {
		maxOutput = 4096
	}

	supportsReasoning := strings.Contains(m, "3-7") || strings.Contains(m, "thinking")

	caps := provider.ModelCapabilities{
		ModelID:            modelID,
		Provider:           domain.ProviderKindAnthropic,
		ContextLimit:       contextLimit,
		MaxOutputTokens:    maxOutput,
		SupportsStreaming:  true,
		SupportsTools:      true,
		SupportsVision:     true,
		SupportsJSON:       false, // Anthropic enforces JSON via tool calling or prompt instructions rather than json_object mode
		SupportsEmbeddings: false,
		SupportsReasoning:  supportsReasoning,
		Capabilities: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityMultimodal,
		},
	}
	if supportsReasoning {
		caps.Capabilities = append(caps.Capabilities, provider.CapabilityThinking, provider.CapabilityReasoning)
	}
	return caps, nil
}

// Execute converts a normalized request into Anthropic messages API and executes it.
func (a *ProviderAdapter) Execute(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.NormalizedResponse, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindAnthropic, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	anthReq, err := convertNormalizedRequestToAnthropic(req)
	if err != nil {
		return nil, err
	}
	anthReq.Stream = false

	resp, err := a.CreateMessage(ctx, apiKey, anthReq)
	if err != nil {
		return nil, mapAnthropicError(err)
	}

	return convertAnthropicResponseToNormalized(resp), nil
}

// Stream initiates an SSE streaming session converted into normalized chunks.
func (a *ProviderAdapter) Stream(ctx context.Context, apiKey string, req provider.NormalizedRequest) (provider.NormalizedStream, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindAnthropic, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	anthReq, err := convertNormalizedRequestToAnthropic(req)
	if err != nil {
		return nil, err
	}
	anthReq.Stream = true

	stream, err := a.CreateMessageStream(ctx, apiKey, anthReq)
	if err != nil {
		return nil, mapAnthropicError(err)
	}

	return &anthropicNormalizedStream{stream: stream}, nil
}

// EstimateTokens calculates tokens for a normalized request.
func (a *ProviderAdapter) EstimateTokens(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.TokenUsage, error) {
	anthReq, err := convertNormalizedRequestToAnthropic(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.CountTokens(ctx, apiKey, &CountTokensRequest{
		Model:    anthReq.Model,
		Messages: anthReq.Messages,
		System:   anthReq.System,
		Tools:    anthReq.Tools,
		Thinking: anthReq.Thinking,
	})
	if err != nil {
		return nil, mapAnthropicError(err)
	}

	return &provider.TokenUsage{
		PromptTokens: resp.InputTokens,
		TotalTokens:  resp.InputTokens,
	}, nil
}

type anthropicNormalizedStream struct {
	stream           *MessageStream
	messageID        string
	model            string
	inputTokens      int
	cachedReadTokens int
	currentToolCall  *provider.ToolCall
}

func (s *anthropicNormalizedStream) Next() (*provider.StreamChunk, error) {
	for {
		event, err := s.stream.Next()
		if err != nil {
			return nil, err
		}

		switch event.Type {
		case "message_start":
			if event.Message != nil {
				s.messageID = event.Message.ID
				s.model = event.Message.Model
				s.inputTokens = event.Message.Usage.InputTokens
				s.cachedReadTokens = event.Message.Usage.CacheReadInputTokens
			}
			return &provider.StreamChunk{
				ID: s.messageID,
				Usage: &provider.TokenUsage{
					PromptTokens: s.inputTokens,
					CachedTokens: s.cachedReadTokens,
					TotalTokens:  s.inputTokens,
				},
			}, nil

		case "content_block_start":
			var cb *ContentBlock
			if event.ContentBlock != nil {
				cb = event.ContentBlock
			} else {
				var startData struct {
					ContentBlock *ContentBlock `json:"content_block"`
				}
				_ = json.Unmarshal(event.Delta, &startData)
				cb = startData.ContentBlock
			}
			if cb != nil && cb.Type == "tool_use" {
				tc := provider.ToolCall{
					ID:        cb.ID,
					Type:      "function",
					Name:      cb.Name,
					Arguments: string(cb.Input),
				}
				s.currentToolCall = &tc
				return &provider.StreamChunk{
					ID:        s.messageID,
					ToolCalls: []provider.ToolCall{tc},
				}, nil
			}

		case "content_block_delta":
			var delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				Thinking    string `json:"thinking"`
			}
			_ = json.Unmarshal(event.Delta, &delta)
			if delta.Text != "" {
				return &provider.StreamChunk{
					ID:        s.messageID,
					DeltaText: delta.Text,
				}, nil
			}
			if delta.PartialJSON != "" && s.currentToolCall != nil {
				s.currentToolCall.Arguments += delta.PartialJSON
				return &provider.StreamChunk{
					ID: s.messageID,
					ToolCalls: []provider.ToolCall{
						{
							ID:        s.currentToolCall.ID,
							Type:      "function",
							Name:      s.currentToolCall.Name,
							Arguments: delta.PartialJSON,
						},
					},
				}, nil
			}

		case "message_delta":
			var delta struct {
				StopReason string `json:"stop_reason"`
			}
			_ = json.Unmarshal(event.Delta, &delta)
			var usage *provider.TokenUsage
			if event.Usage != nil {
				usage = &provider.TokenUsage{
					PromptTokens:     s.inputTokens,
					CompletionTokens: event.Usage.OutputTokens,
					TotalTokens:      s.inputTokens + event.Usage.OutputTokens,
					CachedTokens:     s.cachedReadTokens,
				}
			}
			return &provider.StreamChunk{
				ID:           s.messageID,
				FinishReason: delta.StopReason,
				Usage:        usage,
			}, nil

		case "message_stop":
			return nil, io.EOF
		}
	}
}

func (s *anthropicNormalizedStream) Close() error {
	return s.stream.Close()
}

func convertNormalizedRequestToAnthropic(req provider.NormalizedRequest) (*MessageRequest, error) {
	var systemPrompt string
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
	}

	var msgs []Message
	for _, m := range req.Messages {
		if m.Role == provider.RoleSystem {
			if systemPrompt == "" {
				systemPrompt = m.Content
			}
			continue
		}

		if m.Role == provider.RoleTool {
			toolUseID := m.ToolCallID
			if toolUseID == "" {
				toolUseID = m.Name
			}
			resultContent := json.RawMessage(fmt.Sprintf("%q", m.Content))

			toolResultBlock := ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolUseID,
				Content:   resultContent,
			}

			// In Anthropic API, consecutive tool results belong in the same user turn
			if len(msgs) > 0 && msgs[len(msgs)-1].Role == "user" {
				var prevBlocks []ContentBlock
				if err := json.Unmarshal(msgs[len(msgs)-1].Content, &prevBlocks); err == nil {
					prevBlocks = append(prevBlocks, toolResultBlock)
					updatedRaw, err := json.Marshal(prevBlocks)
					if err == nil {
						msgs[len(msgs)-1].Content = updatedRaw
						continue
					}
				}
			}

			blocks := []ContentBlock{toolResultBlock}
			raw, err := json.Marshal(blocks)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, Message{
				Role:    "user",
				Content: raw,
			})
			continue
		}

		role := "user"
		if m.Role == provider.RoleAssistant {
			role = "assistant"
		}

		var blocks []ContentBlock
		if m.Role == provider.RoleAssistant && len(m.ToolCalls) > 0 {
			if m.Content != "" {
				blocks = append(blocks, ContentBlock{
					Type: "text",
					Text: m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				inputRaw := json.RawMessage(tc.Arguments)
				if len(inputRaw) == 0 || !json.Valid(inputRaw) {
					inputRaw = json.RawMessage("{}")
				}
				blocks = append(blocks, ContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: inputRaw,
				})
			}
			raw, err := json.Marshal(blocks)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, Message{
				Role:    role,
				Content: raw,
			})
			continue
		}

		if len(m.Parts) > 0 {
			for _, p := range m.Parts {
				switch p.Type {
				case provider.ContentTypeText:
					blocks = append(blocks, ContentBlock{
						Type: "text",
						Text: p.Text,
					})
				case provider.ContentTypeImage:
					mediaType := p.MimeType
					if mediaType == "" {
						mediaType = "image/jpeg"
					}
					data := p.ImageURL
					if len(p.Data) > 0 {
						data = base64.StdEncoding.EncodeToString(p.Data)
					}
					blocks = append(blocks, ContentBlock{
						Type: "image",
						Source: &ImageSource{
							Type:      "base64",
							MediaType: mediaType,
							Data:      data,
						},
					})
				}
			}
			raw, err := json.Marshal(blocks)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, Message{
				Role:    role,
				Content: raw,
			})
		} else {
			raw, err := json.Marshal(m.Content)
			if err != nil {
				return nil, err
			}
			msgs = append(msgs, Message{
				Role:    role,
				Content: raw,
			})
		}
	}

	var tools []Tool
	for _, t := range req.Tools {
		raw, _ := json.Marshal(t.Parameters)
		tools = append(tools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: raw,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	anthReq := &MessageRequest{
		Model:         req.Model,
		Messages:      msgs,
		MaxTokens:     maxTokens,
		Stream:        req.Stream,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		Tools:         tools,
		StopSequences: req.StopSequences,
	}
	if systemPrompt != "" {
		anthReq.System = systemPrompt
	}

	// Preserve supported custom parameters
	if req.CustomParameters != nil {
		if th, ok := req.CustomParameters["thinking"]; ok {
			switch v := th.(type) {
			case map[string]any:
				cfg := &ThinkingConfig{Type: "enabled"}
				if b, ok := v["budget_tokens"].(float64); ok {
					cfg.BudgetTokens = int(b)
				} else if b, ok := v["budget_tokens"].(int); ok {
					cfg.BudgetTokens = b
				}
				if t, ok := v["type"].(string); ok && t != "" {
					cfg.Type = t
				}
				anthReq.Thinking = cfg
			case bool:
				if v {
					anthReq.Thinking = &ThinkingConfig{Type: "enabled", BudgetTokens: 2048}
				}
			case int:
				anthReq.Thinking = &ThinkingConfig{Type: "enabled", BudgetTokens: v}
			case float64:
				anthReq.Thinking = &ThinkingConfig{Type: "enabled", BudgetTokens: int(v)}
			}
		}

		if tk, ok := req.CustomParameters["top_k"]; ok {
			switch v := tk.(type) {
			case int:
				anthReq.TopK = &v
			case float64:
				iv := int(v)
				anthReq.TopK = &iv
			}
		}

		if stops, ok := req.CustomParameters["stop_sequences"].([]string); ok && len(stops) > 0 {
			anthReq.StopSequences = stops
		}

		if meta, ok := req.CustomParameters["metadata"].(map[string]any); ok {
			m := &Metadata{}
			if uid, ok := meta["user_id"].(string); ok {
				m.UserID = uid
			}
			anthReq.Metadata = m
		}

		if tc, ok := req.CustomParameters["tool_choice"]; ok {
			switch v := tc.(type) {
			case string:
				anthReq.ToolChoice = &ToolChoice{Type: v}
			case map[string]any:
				choice := &ToolChoice{}
				if t, ok := v["type"].(string); ok {
					choice.Type = t
				}
				if n, ok := v["name"].(string); ok {
					choice.Name = n
				}
				if dp, ok := v["disable_parallel_tool_use"].(bool); ok {
					choice.DisableParallelToolUse = &dp
				}
				anthReq.ToolChoice = choice
			}
		}
	}

	return anthReq, nil
}

func convertAnthropicResponseToNormalized(resp *MessageResponse) *provider.NormalizedResponse {
	var textBuilder strings.Builder
	var toolCalls []provider.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textBuilder.WriteString(block.Text)
		case "tool_use":
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:        block.ID,
				Type:      "function",
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}

	return &provider.NormalizedResponse{
		ID:           resp.ID,
		Model:        resp.Model,
		Provider:     domain.ProviderKindAnthropic,
		Text:         textBuilder.String(),
		ToolCalls:    toolCalls,
		FinishReason: resp.StopReason,
		Usage: provider.TokenUsage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CachedTokens:     resp.Usage.CacheReadInputTokens,
		},
	}
}

func mapAnthropicError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidAPIKey):
		return fmt.Errorf("%w: %w", ErrInvalidAPIKey, provider.ErrInvalidCredentials)
	case errors.Is(err, ErrRateLimited):
		return fmt.Errorf("%w: %w", ErrRateLimited, provider.ErrRateLimitExceeded)
	case errors.Is(err, ErrOverloaded):
		return fmt.Errorf("%w: %w", ErrOverloaded, provider.ErrProviderUnavailable)
	case errors.Is(err, ErrModelNotFound):
		return fmt.Errorf("%w: %w", ErrModelNotFound, provider.ErrModelNotFound)
	default:
		return err
	}
}
