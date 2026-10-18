package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

var _ provider.InferenceProvider = (*ProviderAdapter)(nil)

// Name returns the canonical provider name.
func (a *ProviderAdapter) Name() string {
	return "OpenAI"
}

// Kind returns the provider identifier.
func (a *ProviderAdapter) Kind() string {
	return domain.ProviderKindOpenAI
}

// Metadata returns provider characteristics and feature availability.
func (a *ProviderAdapter) Metadata() provider.ProviderMetadata {
	return provider.ProviderMetadata{
		Kind:           domain.ProviderKindOpenAI,
		DisplayName:    "OpenAI",
		DefaultBaseURL: defaultBaseURL,
		DocURL:         "https://platform.openai.com/docs/api-reference",
		SupportedFeatures: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityStructuredJSON,
			provider.CapabilityMultimodal,
			provider.CapabilityEmbeddings,
			provider.CapabilityReasoning,
		},
		SupportedParameters: []string{
			"reasoning_effort",
			"frequency_penalty",
			"presence_penalty",
			"seed",
			"parallel_tool_calls",
			"logit_bias",
			"user",
			"metadata",
			"store",
			"service_tier",
		},
		AuthHeaderFormat: "Bearer <token>",
	}
}

// GetModelCapabilities returns verified constraints and capabilities for OpenAI models.
func (a *ProviderAdapter) GetModelCapabilities(_ context.Context, modelID string) (provider.ModelCapabilities, error) {
	m := strings.ToLower(modelID)
	contextLimit := 128000
	maxOutput := 16384
	supportsEmbeddings := strings.HasPrefix(m, "text-embedding")
	supportsVision := strings.Contains(m, "gpt-4o") || strings.Contains(m, "o1") || strings.Contains(m, "vision")
	supportsReasoning := strings.Contains(m, "o1") || strings.Contains(m, "o3")

	if strings.Contains(m, "o1") || strings.Contains(m, "o3") {
		maxOutput = 65536
	} else if strings.Contains(m, "gpt-3.5") {
		contextLimit = 16385
		maxOutput = 4096
		supportsVision = false
	}

	caps := provider.ModelCapabilities{
		ModelID:            modelID,
		Provider:           domain.ProviderKindOpenAI,
		ContextLimit:       contextLimit,
		MaxOutputTokens:    maxOutput,
		SupportsStreaming:  !supportsEmbeddings,
		SupportsTools:      !supportsEmbeddings,
		SupportsVision:     supportsVision,
		SupportsJSON:       !supportsEmbeddings,
		SupportsEmbeddings: supportsEmbeddings,
		SupportsReasoning:  supportsReasoning,
		Capabilities: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityStructuredJSON,
		},
	}
	if supportsVision {
		caps.Capabilities = append(caps.Capabilities, provider.CapabilityMultimodal)
	}
	if supportsEmbeddings {
		caps.Capabilities = append(caps.Capabilities, provider.CapabilityEmbeddings)
	}
	if supportsReasoning {
		caps.Capabilities = append(caps.Capabilities, provider.CapabilityReasoning)
	}
	return caps, nil
}

// Execute converts a normalized request into OpenAI chat completions API and executes it.
func (a *ProviderAdapter) Execute(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.NormalizedResponse, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindOpenAI, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	openAIReq, err := convertNormalizedRequestToOpenAI(req)
	if err != nil {
		return nil, err
	}
	openAIReq.Stream = false

	resp, err := a.CreateMessage(ctx, apiKey, openAIReq)
	if err != nil {
		return nil, mapOpenAIError(err)
	}

	return convertOpenAIResponseToNormalized(resp), nil
}

// Stream initiates a streaming completion session converted into normalized chunks.
func (a *ProviderAdapter) Stream(ctx context.Context, apiKey string, req provider.NormalizedRequest) (provider.NormalizedStream, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindOpenAI, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	openAIReq, err := convertNormalizedRequestToOpenAI(req)
	if err != nil {
		return nil, err
	}
	openAIReq.Stream = true

	stream, err := a.CreateMessageStream(ctx, apiKey, openAIReq)
	if err != nil {
		return nil, mapOpenAIError(err)
	}

	return &openAINormalizedStream{stream: stream}, nil
}

// EstimateTokens calculates tokens for a normalized request.
func (a *ProviderAdapter) EstimateTokens(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.TokenUsage, error) {
	openAIReq, err := convertNormalizedRequestToOpenAI(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.CountTokens(ctx, apiKey, openAIReq)
	if err != nil {
		return nil, mapOpenAIError(err)
	}

	return &provider.TokenUsage{
		PromptTokens: resp.PromptTokens,
		TotalTokens:  resp.PromptTokens,
	}, nil
}

type openAINormalizedStream struct {
	stream *MessageStream
}

func (s *openAINormalizedStream) Next() (*provider.StreamChunk, error) {
	event, err := s.stream.Next()
	if err != nil {
		return nil, err
	}

	chunk := &provider.StreamChunk{
		ID: event.ID,
	}

	if len(event.Choices) > 0 {
		choice := event.Choices[0]
		if contentStr, ok := choice.Delta.Content.(string); ok {
			chunk.DeltaText = contentStr
		}
		chunk.FinishReason = choice.FinishReason

		for _, tc := range choice.Delta.ToolCalls {
			chunk.ToolCalls = append(chunk.ToolCalls, provider.ToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: string(tc.Function.Arguments),
			})
		}
	}

	if event.Usage != nil {
		chunk.Usage = &provider.TokenUsage{
			PromptTokens:     event.Usage.PromptTokens,
			CompletionTokens: event.Usage.CompletionTokens,
			TotalTokens:      event.Usage.TotalTokens,
		}
		if event.Usage.PromptTokensDetails != nil {
			chunk.Usage.CachedTokens = event.Usage.PromptTokensDetails.CachedTokens
		}
		if event.Usage.CompletionTokensDetails != nil {
			chunk.Usage.ReasoningTokens = event.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}

	return chunk, nil
}

func (s *openAINormalizedStream) Close() error {
	return s.stream.Close()
}

func convertNormalizedRequestToOpenAI(req provider.NormalizedRequest) (*ChatCompletionRequest, error) {
	var msgs []ChatMessage

	// If system prompt is provided, prepend as system message
	if req.SystemPrompt != "" {
		msgs = append(msgs, ChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, m := range req.Messages {
		role := string(m.Role)
		if role == "" {
			role = "user"
		}

		if len(m.Parts) > 0 {
			var parts []ContentPart
			for _, p := range m.Parts {
				switch p.Type {
				case provider.ContentTypeText:
					parts = append(parts, ContentPart{
						Type: "text",
						Text: p.Text,
					})
				case provider.ContentTypeImage:
					imageURL := p.ImageURL
					if imageURL == "" && len(p.Data) > 0 {
						mime := p.MimeType
						if mime == "" {
							mime = "image/jpeg"
						}
						imageURL = fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(p.Data))
					}
					parts = append(parts, ContentPart{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: imageURL,
						},
					})
				}
			}
			msgs = append(msgs, ChatMessage{
				Role:    role,
				Content: parts,
			})
		} else {
			var toolCalls []ToolCall
			for _, tc := range m.ToolCalls {
				toolCalls = append(toolCalls, ToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: FunctionCall{
						Name:      tc.Name,
						Arguments: json.RawMessage(tc.Arguments),
					},
				})
			}

			msgs = append(msgs, ChatMessage{
				Role:       role,
				Content:    m.Content,
				ToolCalls:  toolCalls,
				ToolCallID: m.ToolCallID,
			})
		}
	}

	var tools []Tool
	for _, t := range req.Tools {
		tools = append(tools, Tool{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	chatReq := &ChatCompletionRequest{
		Model:       req.Model,
		Messages:    msgs,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Tools:       tools,
		ToolChoice:  req.ToolChoice,
		Stop:        req.StopSequences,
	}

	if req.MaxTokens > 0 {
		maxTok := req.MaxTokens
		chatReq.MaxTokens = &maxTok
	}

	if req.ResponseFormat != nil {
		chatReq.ResponseFormat = &ResponseFormat{
			Type: req.ResponseFormat.Type,
		}
	}

	// Preserve supported custom parameters
	if req.CustomParameters != nil {
		if re, ok := req.CustomParameters["reasoning_effort"].(string); ok && re != "" {
			chatReq.ReasoningEffort = re
		}
		if fp, ok := req.CustomParameters["frequency_penalty"]; ok {
			switch v := fp.(type) {
			case float64:
				chatReq.FrequencyPenalty = &v
			case int:
				fv := float64(v)
				chatReq.FrequencyPenalty = &fv
			}
		}
		if pp, ok := req.CustomParameters["presence_penalty"]; ok {
			switch v := pp.(type) {
			case float64:
				chatReq.PresencePenalty = &v
			case int:
				fv := float64(v)
				chatReq.PresencePenalty = &fv
			}
		}
		if seed, ok := req.CustomParameters["seed"]; ok {
			switch v := seed.(type) {
			case int:
				chatReq.Seed = &v
			case float64:
				iv := int(v)
				chatReq.Seed = &iv
			}
		}
		if ptc, ok := req.CustomParameters["parallel_tool_calls"].(bool); ok {
			chatReq.ParallelToolCalls = &ptc
		}
		if lb, ok := req.CustomParameters["logit_bias"].(map[string]any); ok {
			bias := make(map[string]int, len(lb))
			for k, v := range lb {
				if iv, ok := v.(int); ok {
					bias[k] = iv
				} else if fv, ok := v.(float64); ok {
					bias[k] = int(fv)
				}
			}
			chatReq.LogitBias = bias
		}
		if u, ok := req.CustomParameters["user"].(string); ok {
			chatReq.User = u
		}
		if st, ok := req.CustomParameters["service_tier"].(string); ok {
			chatReq.ServiceTier = st
		}
		if s, ok := req.CustomParameters["store"].(bool); ok {
			chatReq.Store = &s
		}
		if meta, ok := req.CustomParameters["metadata"].(map[string]any); ok {
			mMap := make(map[string]string, len(meta))
			for k, v := range meta {
				mMap[k] = fmt.Sprint(v)
			}
			chatReq.Metadata = mMap
		}
	}

	return chatReq, nil
}

func convertOpenAIResponseToNormalized(resp *ChatCompletionResponse) *provider.NormalizedResponse {
	norm := &provider.NormalizedResponse{
		ID:       resp.ID,
		Model:    resp.Model,
		Provider: domain.ProviderKindOpenAI,
	}

	if resp.Usage != nil {
		norm.Usage = provider.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
		if resp.Usage.PromptTokensDetails != nil {
			norm.Usage.CachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
		}
		if resp.Usage.CompletionTokensDetails != nil {
			norm.Usage.ReasoningTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		norm.FinishReason = choice.FinishReason
		if s, ok := choice.Message.Content.(string); ok {
			norm.Text = s
		}

		for _, tc := range choice.Message.ToolCalls {
			norm.ToolCalls = append(norm.ToolCalls, provider.ToolCall{
				ID:        tc.ID,
				Type:      tc.Type,
				Name:      tc.Function.Name,
				Arguments: string(tc.Function.Arguments),
			})
		}
	}

	return norm
}

func mapOpenAIError(err error) error {
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
