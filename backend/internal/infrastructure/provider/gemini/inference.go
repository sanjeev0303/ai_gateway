package gemini

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

var _ provider.InferenceProvider = (*ProviderAdapter)(nil)

// Name returns the canonical provider name.
func (a *ProviderAdapter) Name() string {
	return "Google Gemini"
}

// Kind returns the provider identifier.
func (a *ProviderAdapter) Kind() string {
	return domain.ProviderKindGemini
}

// Metadata returns provider characteristics and feature availability.
func (a *ProviderAdapter) Metadata() provider.ProviderMetadata {
	return provider.ProviderMetadata{
		Kind:           domain.ProviderKindGemini,
		DisplayName:    "Google Gemini",
		DefaultBaseURL: defaultBaseURL,
		DocURL:         "https://ai.google.dev/api/rest",
		SupportedFeatures: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityStructuredJSON,
			provider.CapabilityMultimodal,
			provider.CapabilityEmbeddings,
			provider.CapabilitySafetySettings,
		},
		SupportedParameters: []string{
			"safety_settings",
			"top_k",
			"candidate_count",
			"stop_sequences",
		},
		AuthHeaderFormat: "x-goog-api-key: <token>",
	}
}

// GetModelCapabilities returns verified constraints and capabilities for Gemini models.
func (a *ProviderAdapter) GetModelCapabilities(_ context.Context, modelID string) (provider.ModelCapabilities, error) {
	m := strings.ToLower(strings.TrimPrefix(modelID, "models/"))
	contextLimit := 1048576
	maxOutput := 8192
	supportsEmbeddings := strings.Contains(m, "embedding")

	if strings.Contains(m, "1.5-pro") || strings.Contains(m, "2.5-pro") {
		contextLimit = 2097152
	}
	if strings.Contains(m, "2.0") || strings.Contains(m, "2.5") {
		maxOutput = 65536
	}

	caps := provider.ModelCapabilities{
		ModelID:            modelID,
		Provider:           domain.ProviderKindGemini,
		ContextLimit:       contextLimit,
		MaxOutputTokens:    maxOutput,
		SupportsStreaming:  !supportsEmbeddings,
		SupportsTools:      !supportsEmbeddings,
		SupportsVision:     !supportsEmbeddings,
		SupportsJSON:       !supportsEmbeddings,
		SupportsEmbeddings: supportsEmbeddings,
		Capabilities: []provider.Capability{
			provider.CapabilityTextGeneration,
			provider.CapabilityStreaming,
			provider.CapabilitySystemPrompt,
			provider.CapabilityChatHistory,
			provider.CapabilityToolCalling,
			provider.CapabilityStructuredJSON,
			provider.CapabilityMultimodal,
		},
	}
	if supportsEmbeddings {
		caps.Capabilities = []provider.Capability{provider.CapabilityEmbeddings}
	}
	return caps, nil
}

// Execute converts a normalized request into Gemini generateContent API and executes it.
func (a *ProviderAdapter) Execute(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.NormalizedResponse, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindGemini, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	geminiReq, err := convertNormalizedRequestToGemini(req)
	if err != nil {
		return nil, err
	}

	resp, err := a.CreateMessage(ctx, apiKey, geminiReq)
	if err != nil {
		return nil, mapGeminiError(err)
	}

	return convertGeminiResponseToNormalized(req.Model, resp), nil
}

// Stream initiates a streaming completion session converted into normalized chunks.
func (a *ProviderAdapter) Stream(ctx context.Context, apiKey string, req provider.NormalizedRequest) (provider.NormalizedStream, error) {
	if err := provider.ValidateSupportedParameters(domain.ProviderKindGemini, a.Metadata().SupportedParameters, req.CustomParameters); err != nil {
		return nil, err
	}
	caps, err := a.GetModelCapabilities(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if err := caps.ValidateRequest(req); err != nil {
		return nil, err
	}

	geminiReq, err := convertNormalizedRequestToGemini(req)
	if err != nil {
		return nil, err
	}

	stream, err := a.CreateMessageStream(ctx, apiKey, geminiReq)
	if err != nil {
		return nil, mapGeminiError(err)
	}

	return &geminiNormalizedStream{stream: stream}, nil
}

// EstimateTokens calculates tokens for a normalized request using Gemini countTokens API.
func (a *ProviderAdapter) EstimateTokens(ctx context.Context, apiKey string, req provider.NormalizedRequest) (*provider.TokenUsage, error) {
	geminiReq, err := convertNormalizedRequestToGemini(req)
	if err != nil {
		return nil, err
	}

	countReq := &CountTokensRequest{
		Contents: geminiReq.Contents,
	}

	resp, err := a.CountTokens(ctx, apiKey, req.Model, countReq)
	if err != nil {
		return nil, mapGeminiError(err)
	}

	return &provider.TokenUsage{
		PromptTokens: resp.TotalTokens,
		TotalTokens:  resp.TotalTokens,
	}, nil
}

type geminiNormalizedStream struct {
	stream *MessageStream
}

func (s *geminiNormalizedStream) Next() (*provider.StreamChunk, error) {
	event, err := s.stream.Next()
	if err != nil {
		return nil, err
	}

	chunk := &provider.StreamChunk{}

	if len(event.Candidates) > 0 {
		cand := event.Candidates[0]
		chunk.FinishReason = cand.FinishReason

		if cand.Content != nil {
			var textParts []string
			for _, p := range cand.Content.Parts {
				if p.Text != "" {
					textParts = append(textParts, p.Text)
				}
				if p.FunctionCall != nil {
					argsJSON, _ := json.Marshal(p.FunctionCall.Args)
					chunk.ToolCalls = append(chunk.ToolCalls, provider.ToolCall{
						ID:        p.FunctionCall.Name,
						Type:      "function",
						Name:      p.FunctionCall.Name,
						Arguments: string(argsJSON),
					})
				}
			}
			chunk.DeltaText = strings.Join(textParts, "")
		}
	}

	if event.UsageMetadata != nil {
		chunk.Usage = &provider.TokenUsage{
			PromptTokens:     event.UsageMetadata.PromptTokenCount,
			CompletionTokens: event.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      event.UsageMetadata.TotalTokenCount,
			CachedTokens:     event.UsageMetadata.CachedContentTokenCount,
			ReasoningTokens:  event.UsageMetadata.ThoughtsTokenCount,
		}
	}

	return chunk, nil
}

func (s *geminiNormalizedStream) Close() error {
	return s.stream.Close()
}

func convertNormalizedRequestToGemini(req provider.NormalizedRequest) (*GenerateContentRequest, error) {
	geminiReq := &GenerateContentRequest{
		Model: strings.TrimPrefix(req.Model, "models/"),
	}

	if req.SystemPrompt != "" {
		geminiReq.SystemInstruction = &Content{
			Role: "user",
			Parts: []Part{
				{Text: req.SystemPrompt},
			},
		}
	}

	for _, m := range req.Messages {
		if m.Role == provider.RoleTool {
			name := m.Name
			if name == "" {
				name = m.ToolCallID
			}
			var respMap map[string]interface{}
			if err := json.Unmarshal([]byte(m.Content), &respMap); err != nil {
				respMap = map[string]interface{}{"output": m.Content}
			}
			geminiReq.Contents = append(geminiReq.Contents, Content{
				Role: "function",
				Parts: []Part{
					{
						FunctionResponse: &FunctionResponse{
							Name:     name,
							Response: respMap,
						},
					},
				},
			})
			continue
		}

		role := "user"
		if m.Role == provider.RoleAssistant {
			role = "model"
		}

		var parts []Part
		if len(m.Parts) > 0 {
			for _, p := range m.Parts {
				switch p.Type {
				case provider.ContentTypeText:
					parts = append(parts, Part{Text: p.Text})
				case provider.ContentTypeImage:
					dataB64 := ""
					if len(p.Data) > 0 {
						dataB64 = base64.StdEncoding.EncodeToString(p.Data)
					}
					mime := p.MimeType
					if mime == "" {
						mime = "image/jpeg"
					}
					parts = append(parts, Part{
						InlineData: &Blob{
							MimeType: mime,
							Data:     dataB64,
						},
					})
				}
			}
		} else {
			if m.Content != "" {
				parts = append(parts, Part{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var argsMap map[string]interface{}
				if tc.Arguments != "" {
					_ = json.Unmarshal([]byte(tc.Arguments), &argsMap)
				}
				parts = append(parts, Part{
					FunctionCall: &FunctionCall{
						Name: tc.Name,
						Args: argsMap,
					},
				})
			}
		}

		geminiReq.Contents = append(geminiReq.Contents, Content{
			Role:  role,
			Parts: parts,
		})
	}

	cfg := &GenerationConfig{
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}
	if req.MaxTokens > 0 {
		maxTok := req.MaxTokens
		cfg.MaxOutputTokens = &maxTok
	}
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		cfg.ResponseMimeType = "application/json"
	}

	// Preserve supported custom parameters
	if req.CustomParameters != nil {
		if ss, ok := req.CustomParameters["safety_settings"]; ok {
			switch v := ss.(type) {
			case string:
				threshold := v
				categories := []string{
					"HARM_CATEGORY_HARASSMENT",
					"HARM_CATEGORY_HATE_SPEECH",
					"HARM_CATEGORY_SEXUALLY_EXPLICIT",
					"HARM_CATEGORY_DANGEROUS_CONTENT",
				}
				var settings []SafetySetting
				for _, cat := range categories {
					settings = append(settings, SafetySetting{
						Category:  cat,
						Threshold: threshold,
					})
				}
				geminiReq.SafetySettings = settings
			case []map[string]string:
				var settings []SafetySetting
				for _, item := range v {
					settings = append(settings, SafetySetting{
						Category:  item["category"],
						Threshold: item["threshold"],
					})
				}
				geminiReq.SafetySettings = settings
			case []SafetySetting:
				geminiReq.SafetySettings = v
			case []map[string]any:
				var settings []SafetySetting
				for _, item := range v {
					s := SafetySetting{}
					if cat, ok := item["category"].(string); ok {
						s.Category = cat
					}
					if th, ok := item["threshold"].(string); ok {
						s.Threshold = th
					}
					settings = append(settings, s)
				}
				geminiReq.SafetySettings = settings
			case []any:
				var settings []SafetySetting
				for _, item := range v {
					if itemMap, ok := item.(map[string]any); ok {
						s := SafetySetting{}
						if cat, ok := itemMap["category"].(string); ok {
							s.Category = cat
						}
						if th, ok := itemMap["threshold"].(string); ok {
							s.Threshold = th
						}
						settings = append(settings, s)
					}
				}
				geminiReq.SafetySettings = settings
			}
		}

		if tk, ok := req.CustomParameters["top_k"]; ok {
			switch v := tk.(type) {
			case int:
				cfg.TopK = &v
			case float64:
				iv := int(v)
				cfg.TopK = &iv
			}
		}

		if cc, ok := req.CustomParameters["candidate_count"]; ok {
			switch v := cc.(type) {
			case int:
				cfg.CandidateCount = &v
			case float64:
				iv := int(v)
				cfg.CandidateCount = &iv
			}
		}

		if len(req.StopSequences) > 0 {
			cfg.StopSequences = req.StopSequences
		} else if stops, ok := req.CustomParameters["stop_sequences"].([]string); ok && len(stops) > 0 {
			cfg.StopSequences = stops
		}
	}

	geminiReq.GenerationConfig = cfg

	if len(req.Tools) > 0 {
		var decls []FunctionDeclaration
		for _, t := range req.Tools {
			decls = append(decls, FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			})
		}
		geminiReq.Tools = []Tool{
			{FunctionDeclarations: decls},
		}
	}

	return geminiReq, nil
}

func convertGeminiResponseToNormalized(model string, resp *GenerateContentResponse) *provider.NormalizedResponse {
	reqID := fmt.Sprintf("gemini-%d", time.Now().UnixNano())
	if resp.ModelVersion != "" {
		reqID = fmt.Sprintf("gemini-%s-%d", resp.ModelVersion, time.Now().UnixNano())
	}
	norm := &provider.NormalizedResponse{
		ID:       reqID,
		Model:    model,
		Provider: domain.ProviderKindGemini,
	}

	if resp.UsageMetadata != nil {
		norm.Usage = provider.TokenUsage{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
			CachedTokens:     resp.UsageMetadata.CachedContentTokenCount,
			ReasoningTokens:  resp.UsageMetadata.ThoughtsTokenCount,
		}
	}

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		norm.FinishReason = cand.FinishReason

		if cand.Content != nil {
			var textParts []string
			for _, p := range cand.Content.Parts {
				if p.Text != "" {
					textParts = append(textParts, p.Text)
				}
				if p.FunctionCall != nil {
					argsJSON, _ := json.Marshal(p.FunctionCall.Args)
					norm.ToolCalls = append(norm.ToolCalls, provider.ToolCall{
						ID:        p.FunctionCall.Name,
						Type:      "function",
						Name:      p.FunctionCall.Name,
						Arguments: string(argsJSON),
					})
				}
			}
			norm.Text = strings.Join(textParts, "")
		}
	}

	return norm
}

func mapGeminiError(err error) error {
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
