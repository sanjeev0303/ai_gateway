// Package gemini implements the Gemini provider adapter.
package gemini

import (
	"errors"
	"time"
)

var (
	ErrInvalidAPIKey  = errors.New("gemini: invalid API key")
	ErrRateLimited    = errors.New("gemini: rate limited")
	ErrProviderError  = errors.New("gemini: provider error")
	ErrInvalidRequest = errors.New("gemini: invalid request")
	ErrModelNotFound  = errors.New("gemini: model not found")
	ErrOverloaded     = errors.New("gemini: overloaded")
)

const (
	defaultVersion = "v1beta"
	defaultTimeout = 120 * time.Second
)

// ---- Gemini Request Types ----------------------------------------------------

// GenerateContentRequest represents a request to the Gemini generateContent API.
// See: https://ai.google.dev/gemini-api/docs/text-generation
type GenerateContentRequest struct {
	Model             string            `json:"-"` // model id used to build the URL path, not serialized
	Contents          []Content         `json:"contents"`
	SystemInstruction *Content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
	Tools             []Tool            `json:"tools,omitempty"`
	SafetySettings    []SafetySetting   `json:"safetySettings,omitempty"`
}

// Content represents a conversation turn.
type Content struct {
	Role  string `json:"role,omitempty"` // "user" or "model"
	Parts []Part `json:"parts"`
}

// Part is a single content part within a turn.
type Part struct {
	Text             string            `json:"text,omitempty"`
	InlineData       *Blob             `json:"inlineData,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// Blob represents inline binary data (base64 encoded).
type Blob struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// FunctionCall represents a model-initiated function call.
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// FunctionResponse represents the result of a function call.
type FunctionResponse struct {
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

// GenerationConfig controls model generation behavior.
type GenerationConfig struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	CandidateCount   *int     `json:"candidateCount,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
	ResponseMimeType string   `json:"responseMimeType,omitempty"`
}

// Tool is a tool declaration (function calling).
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration declares a callable function.
type FunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema
}

// SafetySetting configures harm category thresholds.
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// ---- Gemini Response Types ---------------------------------------------------

// GenerateContentResponse represents the Gemini generateContent API response.
type GenerateContentResponse struct {
	Candidates     []Candidate     `json:"candidates,omitempty"`
	PromptFeedback *PromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *UsageMetadata  `json:"usageMetadata,omitempty"`
	ModelVersion   string          `json:"modelVersion,omitempty"`
}

// Candidate is a single generated response candidate.
type Candidate struct {
	Content      *Content `json:"content,omitempty"`
	FinishReason string   `json:"finishReason,omitempty"` // "STOP", "MAX_TOKENS", "SAFETY", "RECITATION", ...
	Index        int      `json:"index,omitempty"`
}

// PromptFeedback reports why a prompt was blocked, if it was.
type PromptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

// UsageMetadata captures token usage for a response.
type UsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount,omitempty"`
}

// StreamingEvent represents a streaming chunk from streamGenerateContent.
type StreamingEvent struct {
	EventType     string         `json:"event_type,omitempty"` // "content.start", "content.delta", "content.stop"
	Index         int            `json:"index,omitempty"`
	Candidates    []Candidate    `json:"candidates,omitempty"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
	Error         *ErrorDetail   `json:"error,omitempty"`
}

// Usage represents normalized token usage information for Gemini.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
	ThinkingTokens   int
	TotalTokens      int
}

// ---- Count Tokens API --------------------------------------------------------

// CountTokensRequest represents a token counting request.
type CountTokensRequest struct {
	Contents []Content `json:"contents,omitempty"`
}

// CountTokensResponse represents a token counting response.
type CountTokensResponse struct {
	TotalTokens int `json:"totalTokens"`
}

// ---- Models API ---------------------------------------------------------------

// ModelsResponse represents the models list response.
type ModelsResponse struct {
	Models []ModelSummary `json:"models"`
}

// ModelSummary describes a model from the models API.
type ModelSummary struct {
	Name                       string   `json:"name"` // "models/gemini-2.5-flash"
	DisplayName                string   `json:"displayName"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	InputTokenLimit            int      `json:"inputTokenLimit"`
	OutputTokenLimit           int      `json:"outputTokenLimit"`
}

// ---- Error --------------------------------------------------------------------

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents the error details.
type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"` // e.g. "INVALID_ARGUMENT", "UNAUTHENTICATED", "NOT_FOUND", "RESOURCE_EXHAUSTED"
}
