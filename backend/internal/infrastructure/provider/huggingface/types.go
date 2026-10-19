// Package huggingface implements the Hugging Face Inference Providers adapter.
package huggingface

import (
	"errors"
	"time"
)

var (
	ErrInvalidAPIKey  = errors.New("huggingface: invalid API key")
	ErrRateLimited    = errors.New("huggingface: rate limited")
	ErrProviderError  = errors.New("huggingface: provider error")
	ErrInvalidRequest = errors.New("huggingface: invalid request")
	ErrModelNotFound  = errors.New("huggingface: model not found")
	ErrOverloaded     = errors.New("huggingface: overloaded")
)

const (
	defaultBaseURL = "https://api-inference.huggingface.co/v1"
	defaultHubURL  = "https://huggingface.co"
	defaultTimeout = 120 * time.Second
)

// ChatCompletionRequest represents an OpenAI-compatible chat request to Hugging Face Inference API.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
}

// ChatMessage represents a conversation turn.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents the completion response.
type ChatCompletionResponse struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   *UsageInfo   `json:"usage,omitempty"`
}

// ChatChoice is one completion choice.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// UsageInfo represents token usage metadata.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionStreamResponse is one SSE delta event.
type ChatCompletionStreamResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []StreamChatChoice `json:"choices"`
	Usage   *UsageInfo         `json:"usage,omitempty"`
}

// StreamChatChoice is one delta choice.
type StreamChatChoice struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// CountTokensResponse represents a token count response.
type CountTokensResponse struct {
	TotalTokens int `json:"total_tokens"`
}

// HubWhoAmI represents Hugging Face user/org metadata from /api/whoami-v2.
type HubWhoAmI struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Auth  struct {
		Type        string `json:"type"`
		AccessToken struct {
			DisplayName string   `json:"displayName"`
			Role        string   `json:"role"`
			Scopes      []string `json:"scopes"`
		} `json:"accessToken"`
	} `json:"auth"`
}

// HubModelItem represents a model from /api/models.
type HubModelItem struct {
	ID          string   `json:"id"`
	PipelineTag string   `json:"pipeline_tag"`
	Private     bool     `json:"private"`
	Tags        []string `json:"tags"`
}
