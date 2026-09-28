// Package openai implements the OpenAI provider adapter.
package openai

import (
	"encoding/json"
	"errors"
	"io"
	"time"
)

var (
	ErrInvalidAPIKey  = errors.New("openai: invalid API key")
	ErrRateLimited    = errors.New("openai: rate limited")
	ErrProviderError  = errors.New("openai: provider error")
	ErrInvalidRequest = errors.New("openai: invalid request")
	ErrModelNotFound  = errors.New("openai: model not found")
	ErrOverloaded     = errors.New("openai: overloaded")
)

const (
	defaultOpenAIVersion = "2024-08-01-preview"
	defaultTimeout       = 120 * time.Second
)

// ---- OpenAI Request Types ----------------------------------------------------

// ChatCompletionRequest represents the OpenAI chat completions API request.
// See: https://platform.openai.com/docs/api-reference/chat/create
type ChatCompletionRequest struct {
	Model               string            `json:"model"`
	Messages            []ChatMessage     `json:"messages"`
	MaxTokens           *int              `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int              `json:"max_completion_tokens,omitempty"`
	Temperature         *float64          `json:"temperature,omitempty"`
	TopP                *float64          `json:"top_p,omitempty"`
	N                   *int              `json:"n,omitempty"`
	Stream              bool              `json:"stream,omitempty"`
	Stop                []string          `json:"stop,omitempty"`
	PresencePenalty     *float64          `json:"presence_penalty,omitempty"`
	FrequencyPenalty    *float64          `json:"frequency_penalty,omitempty"`
	LogitBias           map[string]int    `json:"logit_bias,omitempty"`
	User                string            `json:"user,omitempty"`
	Tools               []Tool            `json:"tools,omitempty"`
	ToolChoice          interface{}       `json:"tool_choice,omitempty"` // string or ToolChoiceObject
	ParallelToolCalls   *bool             `json:"parallel_tool_calls,omitempty"`
	ResponseFormat      *ResponseFormat   `json:"response_format,omitempty"`
	Seed                *int              `json:"seed,omitempty"`
	ServiceTier         string            `json:"service_tier,omitempty"`     // "auto", "flex", "default"
	ReasoningEffort     string            `json:"reasoning_effort,omitempty"` // "low", "medium", "high"
	Metadata            map[string]string `json:"metadata,omitempty"`
	Store               *bool             `json:"store,omitempty"`
}

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role       string      `json:"role"`    // "system", "user", "assistant", "tool", "developer"
	Content    interface{} `json:"content"` // string or []ContentPart
	Name       string      `json:"name,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	Refusal    string      `json:"refusal,omitempty"`
	Audio      *AudioData  `json:"audio,omitempty"`
}

// ContentPart represents a content part in a message (for multimodal).
type ContentPart struct {
	Type       string      `json:"type"` // "text", "image_url", "input_audio"
	Text       string      `json:"text,omitempty"`
	ImageURL   *ImageURL   `json:"image_url,omitempty"`
	InputAudio *InputAudio `json:"input_audio,omitempty"`
}

// ImageURL represents an image URL for vision.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// InputAudio represents audio input.
type InputAudio struct {
	Data   string `json:"data"`   // base64 encoded
	Format string `json:"format"` // "wav", "mp3"
}

// AudioData represents audio output.
type AudioData struct {
	ID         string `json:"id"`
	ExpiresAt  int64  `json:"expires_at"`
	Data       string `json:"data"` // base64 encoded
	Transcript string `json:"transcript,omitempty"`
}

// ToolCall represents a tool call in a message.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call.
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool represents a tool definition.
type Tool struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition.
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
	Strict      *bool                  `json:"strict,omitempty"`
}

// ToolChoiceObject represents a specific tool choice.
type ToolChoiceObject struct {
	Type     string       `json:"type"` // "function"
	Function FunctionName `json:"function"`
}

// FunctionName represents a function name for tool choice.
type FunctionName struct {
	Name string `json:"name"`
}

// ResponseFormat represents the response format configuration.
type ResponseFormat struct {
	Type       string                    `json:"type"` // "text", "json_object", "json_schema"
	JSONSchema *ResponseFormatJSONSchema `json:"json_schema,omitempty"`
}

// ResponseFormatJSONSchema represents a JSON schema for response format.
type ResponseFormatJSONSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"`
	Strict      *bool                  `json:"strict,omitempty"`
}

// ---- OpenAI Response Types ---------------------------------------------------

// ChatCompletionResponse represents the OpenAI chat completions API response.
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"` // "chat.completion", "chat.completion.chunk"
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
	ServiceTier       string   `json:"service_tier,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int          `json:"index"`
	Message      ChatMessage  `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`         // for streaming
	FinishReason string       `json:"finish_reason,omitempty"` // "stop", "length", "tool_calls", "content_filter", "function_call"
	Logprobs     *Logprobs    `json:"logprobs,omitempty"`
}

// Logprobs represents log probabilities.
type Logprobs struct {
	Content []LogprobContent `json:"content,omitempty"`
}

// LogprobContent represents logprob for a token.
type LogprobContent struct {
	Token       string       `json:"token"`
	Logprob     float64      `json:"logprob"`
	Bytes       []byte       `json:"bytes,omitempty"`
	TopLogprobs []TopLogprob `json:"top_logprobs,omitempty"`
}

// TopLogprob represents a top logprob alternative.
type TopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
	Bytes   []byte  `json:"bytes,omitempty"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// PromptTokensDetails represents prompt token breakdown.
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
	AudioTokens  int `json:"audio_tokens,omitempty"`
	ImageTokens  int `json:"image_tokens,omitempty"`
}

// CompletionTokensDetails represents completion token breakdown.
type CompletionTokensDetails struct {
	ReasoningTokens          int `json:"reasoning_tokens,omitempty"`
	AudioTokens              int `json:"audio_tokens,omitempty"`
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// StreamingEvent represents a server-sent event from the streaming API.
type StreamingEvent struct {
	ID                string          `json:"id,omitempty"`
	Object            string          `json:"object"` // "chat.completion.chunk"
	Created           int64           `json:"created"`
	Model             string          `json:"model"`
	Choices           []StreamChoice  `json:"choices"`
	Usage             *Usage          `json:"usage,omitempty"` // only in final chunk
	SystemFingerprint string          `json:"system_fingerprint,omitempty"`
	Delta             json.RawMessage `json:"delta,omitempty"` // raw delta data when unmarshal fails
	Error             *ErrorDetail    `json:"error,omitempty"` // error details for error events
}

// StreamChoice represents a streaming choice.
type StreamChoice struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Logprobs     *Logprobs   `json:"logprobs,omitempty"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents the error details.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"` // "invalid_request_error", "authentication_error", "permission_error", "not_found_error", "rate_limit_error", "api_error", "overloaded_error"
	Code    string `json:"code,omitempty"`
	Param   string `json:"param,omitempty"`
}

// ---- Models API --------------------------------------------------------------

// Model represents a model from the models API.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"` // "model"
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents the models list response.
type ModelsResponse struct {
	Object string  `json:"object"` // "list"
	Data   []Model `json:"data"`
}

// ---- Embeddings API ----------------------------------------------------------

// EmbeddingRequest represents an embeddings request.
type EmbeddingRequest struct {
	Input          interface{} `json:"input"` // string or []string
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"` // "float", "base64"
	Dimensions     *int        `json:"dimensions,omitempty"`
	User           string      `json:"user,omitempty"`
}

// EmbeddingResponse represents an embeddings response.
type EmbeddingResponse struct {
	Object string      `json:"object"` // "list"
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

// Embedding represents a single embedding.
type Embedding struct {
	Object    string    `json:"object"` // "embedding"
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// ---- Moderation API ----------------------------------------------------------

// ModerationRequest represents a moderation request.
type ModerationRequest struct {
	Input interface{} `json:"input"` // string or []string
	Model string      `json:"model,omitempty"`
}

// ModerationResponse represents a moderation response.
type ModerationResponse struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Results []ModerationResult `json:"results"`
}

// ModerationResult represents a single moderation result.
type ModerationResult struct {
	Flagged        bool               `json:"flagged"`
	Categories     map[string]bool    `json:"categories"`
	CategoryScores map[string]float64 `json:"category_scores"`
}

// ---- Audio API (Transcriptions/Translations) ---------------------------------

// AudioTranscriptionRequest represents a transcription request.
type AudioTranscriptionRequest struct {
	File           io.Reader `json:"-"`
	Filename       string
	Model          string
	Language       string   `json:"language,omitempty"`
	Prompt         string   `json:"prompt,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"` // "json", "text", "srt", "verbose_json", "vtt"
	Temperature    *float64 `json:"temperature,omitempty"`
}

// AudioTranslationRequest represents a translation request.
type AudioTranslationRequest struct {
	File           io.Reader `json:"-"`
	Filename       string
	Model          string
	Prompt         string   `json:"prompt,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
}

// AudioResponse represents audio transcription/translation response.
type AudioResponse struct {
	Text     string    `json:"text"`
	Language string    `json:"language,omitempty"`
	Duration float64   `json:"duration,omitempty"`
	Words    []Word    `json:"words,omitempty"`
	Segments []Segment `json:"segments,omitempty"`
}

type Word struct {
	Word        string  `json:"word"`
	Start       float64 `json:"start"`
	End         float64 `json:"end"`
	Probability float64 `json:"probability,omitempty"`
}

type Segment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}
