// Package provider defines the provider-client port (M4). M4 implements
// credential validation, connectivity health checks and the model catalog;
// request streaming lands as a byte-level reverse proxy in the gateway, which
// is transport and therefore not part of this port. Payloads pass through
// provider-native (Claude Code included) — this port never rewrites them.
package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidCredentials marks a provider rejection of an API key (HTTP 401 or 403).
	ErrInvalidCredentials = errors.New("provider: invalid credentials")

	// ErrUnsupportedCapability indicates that a requested model or provider feature is not supported.
	ErrUnsupportedCapability = errors.New("provider: unsupported capability")

	// ErrUnsupportedParameter indicates that a request parameter is not supported by the upstream provider.
	ErrUnsupportedParameter = errors.New("provider: unsupported parameter")

	// ErrModelNotFound indicates that the requested model does not exist or is inaccessible.
	ErrModelNotFound = errors.New("provider: model not found")

	// ErrRateLimitExceeded marks an upstream 429 rate limit or quota exhaustion.
	ErrRateLimitExceeded = errors.New("provider: upstream rate limit exceeded")

	// ErrProviderUnavailable marks an upstream 502/503/504 outage or network failure.
	ErrProviderUnavailable = errors.New("provider: upstream service unavailable")

	// ErrContextLimitExceeded indicates prompt length exceeds model context capacity.
	ErrContextLimitExceeded = errors.New("provider: context limit exceeded")

	// ErrUnknownProvider indicates the provider kind is not registered.
	ErrUnknownProvider = errors.New("provider: unknown provider")
)

// Provider talks to one upstream LLM provider (Anthropic, OpenAI, Gemini, …).
// Implementations must never log API keys or request payloads.
type Provider interface {
	// ValidateCredentials verifies the API key against the provider. It
	// returns ErrInvalidCredentials on 401/403, and a transport/protocol
	// error otherwise.
	ValidateCredentials(ctx context.Context, apiKey string) error
	// HealthCheck proves connectivity to the provider. It succeeds on any
	// provider response (including 401, which proves reachability) and
	// fails only on transport-level errors.
	HealthCheck(ctx context.Context, apiKey string) error
	// ListModels returns the provider's model IDs.
	ListModels(ctx context.Context, apiKey string) ([]Model, error)
}

// Capability represents an inference capability supported by a provider or model.
type Capability string

const (
	CapabilityTextGeneration   Capability = "text_generation"
	CapabilityStreaming        Capability = "streaming"
	CapabilitySystemPrompt     Capability = "system_prompt"
	CapabilityChatHistory      Capability = "chat_history"
	CapabilityToolCalling      Capability = "tool_calling"
	CapabilityStructuredJSON   Capability = "structured_json"
	CapabilityMultimodal       Capability = "multimodal"
	CapabilityEmbeddings       Capability = "embeddings"
	CapabilityReasoning        Capability = "reasoning"
	CapabilityThinking         Capability = "thinking"
	CapabilitySafetySettings   Capability = "safety_settings"
)

// ModelCapabilities encapsulates verified capabilities for a specific model.
type ModelCapabilities struct {
	ModelID            string       `json:"model_id"`
	Provider           string       `json:"provider"`
	ContextLimit       int          `json:"context_limit"`
	MaxOutputTokens    int          `json:"max_output_tokens"`
	Capabilities       []Capability `json:"capabilities"`
	SupportsStreaming  bool         `json:"supports_streaming"`
	SupportsTools      bool         `json:"supports_tools"`
	SupportsVision     bool         `json:"supports_vision"`
	SupportsJSON       bool         `json:"supports_json"`
	SupportsEmbeddings bool         `json:"supports_embeddings"`
	SupportsReasoning  bool         `json:"supports_reasoning"`
}

// HasCapability checks whether a specific capability is present.
func (mc ModelCapabilities) HasCapability(c Capability) bool {
	for _, cap := range mc.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}

// ValidateRequest ensures a normalized request does not demand unsupported capabilities.
func (mc ModelCapabilities) ValidateRequest(req NormalizedRequest) error {
	if req.Stream && !mc.SupportsStreaming && !mc.HasCapability(CapabilityStreaming) {
		return fmt.Errorf("%w: model %q does not support streaming", ErrUnsupportedCapability, mc.ModelID)
	}
	if len(req.Tools) > 0 && !mc.SupportsTools && !mc.HasCapability(CapabilityToolCalling) {
		return fmt.Errorf("%w: model %q does not support tool calling", ErrUnsupportedCapability, mc.ModelID)
	}
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" && !mc.SupportsJSON && !mc.HasCapability(CapabilityStructuredJSON) {
		return fmt.Errorf("%w: model %q does not support structured JSON responses", ErrUnsupportedCapability, mc.ModelID)
	}

	// Verify multimodal parts
	hasImage := false
	for _, msg := range req.Messages {
		for _, part := range msg.Parts {
			if part.Type == ContentTypeImage {
				hasImage = true
				break
			}
		}
		if hasImage {
			break
		}
	}
	if hasImage && !mc.SupportsVision && !mc.HasCapability(CapabilityMultimodal) {
		return fmt.Errorf("%w: model %q does not support image or multimodal inputs", ErrUnsupportedCapability, mc.ModelID)
	}

	// Verify reasoning capability if requested
	if req.CustomParameters != nil {
		if _, ok := req.CustomParameters["reasoning_effort"]; ok && !mc.SupportsReasoning && !mc.HasCapability(CapabilityReasoning) {
			return fmt.Errorf("%w: model %q does not support reasoning_effort", ErrUnsupportedCapability, mc.ModelID)
		}
	}

	return nil
}

// ValidateSupportedParameters verifies custom parameters against a provider's supported parameter list.
func ValidateSupportedParameters(providerKind string, allowed []string, custom map[string]any) error {
	if len(custom) == 0 {
		return nil
	}
	allowedMap := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedMap[strings.ToLower(a)] = true
	}

	for k := range custom {
		lowerK := strings.ToLower(k)
		if !allowedMap[lowerK] {
			return fmt.Errorf("%w: parameter %q is not supported by %s", ErrUnsupportedParameter, k, providerKind)
		}
	}
	return nil
}

// ProviderMetadata specifies identification, capabilities, and connection formats for a provider.
type ProviderMetadata struct {
	Kind                string       `json:"kind"`
	DisplayName         string       `json:"display_name"`
	DefaultBaseURL      string       `json:"default_base_url"`
	DocURL              string       `json:"doc_url"`
	SupportedFeatures   []Capability `json:"supported_features"`
	SupportedParameters []string     `json:"supported_parameters,omitempty"`
	AuthHeaderFormat    string       `json:"auth_header_format"`
}

// Role represents a conversational role.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ContentPartType distinguishes text from binary or multimodal message parts.
type ContentPartType string

const (
	ContentTypeText  ContentPartType = "text"
	ContentTypeImage ContentPartType = "image"
)

// ContentPart represents a granular message fragment.
type ContentPart struct {
	Type     ContentPartType `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL string          `json:"image_url,omitempty"`
	MimeType string          `json:"mime_type,omitempty"`
	Data     []byte          `json:"data,omitempty"`
}

// ToolCall represents an invoked tool call.
type ToolCall struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Message represents a unified conversational entry.
type Message struct {
	Role       Role          `json:"role"`
	Content    string        `json:"content,omitempty"`
	Parts      []ContentPart `json:"parts,omitempty"`
	Name       string        `json:"name,omitempty"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

// ToolDefinition describes a function/tool callable by the model.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ResponseFormat enforces structured generation schemas.
type ResponseFormat struct {
	Type   string         `json:"type"` // "text", "json_object", "json_schema"
	Schema map[string]any `json:"schema,omitempty"`
}

// NormalizedRequest standardizes multi-provider parameters.
type NormalizedRequest struct {
	Model            string           `json:"model"`
	Messages         []Message        `json:"messages"`
	SystemPrompt     string           `json:"system_prompt,omitempty"`
	Temperature      *float64         `json:"temperature,omitempty"`
	TopP             *float64         `json:"top_p,omitempty"`
	MaxTokens        int              `json:"max_tokens,omitempty"`
	Stream           bool             `json:"stream,omitempty"`
	Tools            []ToolDefinition `json:"tools,omitempty"`
	ToolChoice       any              `json:"tool_choice,omitempty"`
	ResponseFormat   *ResponseFormat  `json:"response_format,omitempty"`
	StopSequences    []string         `json:"stop_sequences,omitempty"`
	CustomParameters map[string]any   `json:"custom_parameters,omitempty"`
}

// TokenUsage tracks exact or estimated token consumption.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
	CachedTokens     int `json:"cached_tokens,omitempty"`
	ReasoningTokens  int `json:"reasoning_tokens,omitempty"`
}

// NormalizedResponse standardizes non-streaming completions across providers.
type NormalizedResponse struct {
	ID           string     `json:"id"`
	Model        string     `json:"model"`
	Provider     string     `json:"provider"`
	Text         string     `json:"text"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string     `json:"finish_reason"`
	Usage        TokenUsage `json:"usage"`
}

// StreamChunk standardizes streaming SSE increments across providers.
type StreamChunk struct {
	ID           string      `json:"id"`
	DeltaText    string      `json:"delta_text"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// NormalizedStream defines an iterator over incoming stream chunks.
type NormalizedStream interface {
	// Next returns the subsequent chunk or io.EOF when stream closes.
	Next() (*StreamChunk, error)
	// Close releases underlying connection resources.
	Close() error
}

// InferenceProvider extends Provider with normalized inference execution, capability resolution, and token measurement.
type InferenceProvider interface {
	Provider

	// Name returns the canonical provider name (e.g. "Google Gemini", "Anthropic Claude").
	Name() string

	// Kind returns the provider identifier string (e.g. "openai", "anthropic", "gemini", "huggingface").
	Kind() string

	// Metadata exposes provider metadata, default URLs, and supported feature set.
	Metadata() ProviderMetadata

	// GetModelCapabilities returns model-specific constraints and feature availability.
	GetModelCapabilities(ctx context.Context, modelID string) (ModelCapabilities, error)

	// Execute converts NormalizedRequest into provider-native payload and executes non-streaming inference.
	Execute(ctx context.Context, apiKey string, req NormalizedRequest) (*NormalizedResponse, error)

	// Stream converts NormalizedRequest into provider-native payload and initiates streaming inference.
	Stream(ctx context.Context, apiKey string, req NormalizedRequest) (NormalizedStream, error)

	// EstimateTokens calculates or retrieves token metrics for a normalized request.
	EstimateTokens(ctx context.Context, apiKey string, req NormalizedRequest) (*TokenUsage, error)
}

// Model is one model the provider exposes, with the metadata the
// provider's catalog API advertises. Fields a provider does not expose
// are left at their zero value; the model registry layers
// governance-owned fields (status, pricing_version, activation flags)
// on top when seeding the DB catalog. Providers must never fabricate
// metadata the upstream API does not return.
type Model struct {
	// ID is the provider's model identifier as used on the wire (e.g.
	// "claude-sonnet-4-5", "gpt-4o", "gemini-2.5-flash").
	ID string `json:"id"`

	// Provider is the provider kind this model belongs to, one of the
	// domain.ProviderKind* values ("openai", "anthropic", "gemini", …).
	Provider string `json:"provider,omitempty"`

	// DisplayName is the human-readable name the provider advertises
	// (e.g. "Gemini 2.5 Flash"); empty when the provider lists only IDs.
	DisplayName string `json:"display_name,omitempty"`

	// Capabilities are the provider-advertised capabilities, e.g.
	// Gemini's supportedGenerationMethods ("generateContent",
	// "countTokens"). Nil when the provider does not advertise them.
	Capabilities []string `json:"capabilities,omitempty"`

	// ContextLimit is the provider-advertised input context window in
	// tokens; 0 when the provider does not advertise one.
	ContextLimit int `json:"context_limit,omitempty"`
}
