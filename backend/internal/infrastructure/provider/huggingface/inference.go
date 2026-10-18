package huggingface

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

var _ provider.InferenceProvider = (*ProviderAdapter)(nil)

// Name returns the canonical provider name.
func (a *ProviderAdapter) Name() string {
	return "Hugging Face"
}

// Kind returns the provider identifier.
func (a *ProviderAdapter) Kind() string {
	return domain.ProviderKindHuggingFace
}

// Metadata returns provider characteristics and feature availability.
func (a *ProviderAdapter) Metadata() provider.ProviderMetadata {
	return provider.ProviderMetadata{
		Kind:           domain.ProviderKindHuggingFace,
		DisplayName:    "Hugging Face",
		DefaultBaseURL: defaultBaseURL,
		DocURL:         "https://huggingface.co/docs/api-inference",
		SupportedFeatures: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
		},
		SupportedParameters: []string{
			"wait_for_model",
			"use_cache",
		},
		AuthHeaderFormat: "Bearer <token>",
	}
}

// GetModelCapabilities returns verified constraints and capabilities for Hugging Face models.
func (a *ProviderAdapter) GetModelCapabilities(_ context.Context, modelID string) (provider.ModelCapabilities, error) {
	m := strings.ToLower(modelID)
	contextLimit := 32768
	maxOutput := 8192
	supportsEmbeddings := strings.Contains(m, "embed") || strings.Contains(m, "sentence-transformers")

	if strings.Contains(m, "llama-3") || strings.Contains(m, "qwen2.5") {
		contextLimit = 131072
	}

	caps := provider.ModelCapabilities{
		ModelID:            modelID,
		Provider:           domain.ProviderKindHuggingFace,
		ContextLimit:       contextLimit,
		MaxOutputTokens:    maxOutput,
		SupportsStreaming:  !supportsEmbeddings,
		SupportsTools:      false,
		SupportsVision:     false,
		SupportsJSON:       false,
		SupportsEmbeddings: supportsEmbeddings,
		Capabilities: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
		},
	}
	if supportsEmbeddings {
		caps.Capabilities = []provider.Capability{provider.CapabilityEmbeddings}
	}
	return caps, nil
}

// Execute converts a normalized request into Hugging Face chat completion API and executes it.
func (a *ProviderAdapter) Execute(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.NormalizedResponse, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindHuggingFace, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	hfReq, err := convertNormalizedRequestToHuggingFace(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.CreateMessage(ctx, apiKey, hfReq)
	if err != nil {
		return nil, mapHuggingFaceError(err)
	}

	return convertHuggingFaceResponseToNormalized(resp), nil
}

// Stream initiates a streaming completion session converted into normalized chunks.
func (a *ProviderAdapter) Stream(ctx context.Context, apiKey string, req provider.NormalizedRequest) (provider.NormalizedStream, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindHuggingFace, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	hfReq, err := convertNormalizedRequestToHuggingFace(req)
	if err != nil {
		return nil, err
	}

	stream, err := a.CreateMessageStream(ctx, apiKey, hfReq)
	if err != nil {
		return nil, mapHuggingFaceError(err)
	}

	return &hfNormalizedStream{stream: stream}, nil
}

// EstimateTokens calculates tokens for a normalized request using the Hugging Face adapter.
func (a *ProviderAdapter) EstimateTokens(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.TokenUsage, error) {
	hfReq, err := convertNormalizedRequestToHuggingFace(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.CountTokens(ctx, apiKey, hfReq)
	if err != nil {
		return nil, mapHuggingFaceError(err)
	}

	return &provider.TokenUsage{
		PromptTokens: resp.TotalTokens,
		TotalTokens:  resp.TotalTokens,
	}, nil
}

type hfNormalizedStream struct {
	stream *MessageStream
}

func (s *hfNormalizedStream) Next() (*provider.StreamChunk, error) {
	event, err := s.stream.Next()
	if err != nil {
		return nil, err
	}

	chunk := &provider.StreamChunk{
		ID: event.ID,
	}

	if len(event.Choices) > 0 {
		ch := event.Choices[0]
		chunk.DeltaText = ch.Delta.Content
		if ch.FinishReason != nil {
			chunk.FinishReason = *ch.FinishReason
		}
	}

	if event.Usage != nil {
		chunk.Usage = &provider.TokenUsage{
			PromptTokens:     event.Usage.PromptTokens,
			CompletionTokens: event.Usage.CompletionTokens,
			TotalTokens:      event.Usage.TotalTokens,
		}
	}

	return chunk, nil
}

func (s *hfNormalizedStream) Close() error {
	return s.stream.Close()
}

func convertNormalizedRequestToHuggingFace(req provider.NormalizedRequest) (*ChatCompletionRequest, error) {
	var msgs []ChatMessage

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
		content := m.Content
		if content == "" && len(m.Parts) > 0 {
			var parts []string
			for _, p := range m.Parts {
				if p.Text != "" {
					parts = append(parts, p.Text)
				}
			}
			content = strings.Join(parts, "\n")
		}
		msgs = append(msgs, ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	hfReq := &ChatCompletionRequest{
		Model:       req.Model,
		Messages:    msgs,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.StopSequences,
	}

	if req.MaxTokens > 0 {
		maxTok := req.MaxTokens
		hfReq.MaxTokens = &maxTok
	}

	return hfReq, nil
}

func convertHuggingFaceResponseToNormalized(resp *ChatCompletionResponse) *provider.NormalizedResponse {
	norm := &provider.NormalizedResponse{
		ID:       resp.ID,
		Model:    resp.Model,
		Provider: domain.ProviderKindHuggingFace,
	}

	if resp.Usage != nil {
		norm.Usage = provider.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		norm.FinishReason = choice.FinishReason
		norm.Text = choice.Message.Content
	}

	return norm
}

func mapHuggingFaceError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrInvalidAPIKey) || errors.Is(err, provider.ErrInvalidCredentials):
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
