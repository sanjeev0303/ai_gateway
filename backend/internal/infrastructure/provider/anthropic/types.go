// Package anthropic implements the Anthropic provider adapter.
package anthropic

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var (
	ErrInvalidAPIKey  = errors.New("anthropic: invalid API key")
	ErrRateLimited    = errors.New("anthropic: rate limited")
	ErrProviderError  = errors.New("anthropic: provider error")
	ErrInvalidRequest = errors.New("anthropic: invalid request")
	ErrModelNotFound  = errors.New("anthropic: model not found")
	ErrOverloaded     = errors.New("anthropic: overloaded")
)

const (
	defaultAnthropicVersion = "2023-06-01"
	defaultTimeout          = 120 * time.Second
)

// ---- Anthropic Request Types -------------------------------------------------

// MessageRequest represents the Anthropic messages API request.
// See: https://docs.anthropic.com/en/api/messages
type MessageRequest struct {
	Model         string          `json:"model"`
	Messages      []Message       `json:"messages"`
	System        any             `json:"system,omitempty"` // string or []ContentBlock
	MaxTokens     int             `json:"max_tokens"`
	Metadata      *Metadata       `json:"metadata,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	TopK          *int            `json:"top_k,omitempty"`
	Tools         []Tool          `json:"tools,omitempty"`
	ToolChoice    *ToolChoice     `json:"tool_choice,omitempty"`
	Thinking      *ThinkingConfig `json:"thinking,omitempty"`
}

// Message represents a single message in the conversation.
type Message struct {
	Role    string          `json:"role"`    // "user" or "assistant"
	Content json.RawMessage `json:"content"` // string or []ContentBlock
}

// ContentBlock represents a content block in a message.
type ContentBlock struct {
	Type      string          `json:"type"` // "text", "image", "tool_use", "tool_result", "thinking"
	Text      string          `json:"text,omitempty"`
	Source    *ImageSource    `json:"source,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   *bool           `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"`
}

// ImageSource represents an image source for vision.
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data"`       // base64 encoded
}

// Tool represents a tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolChoice represents the tool choice configuration.
type ToolChoice struct {
	Type                   string `json:"type"`           // "auto", "any", "tool", "none"
	Name                   string `json:"name,omitempty"` // for "tool" type
	DisableParallelToolUse *bool  `json:"disable_parallel_tool_use,omitempty"`
}

// ThinkingConfig represents the thinking configuration.
type ThinkingConfig struct {
	Type         string `json:"type"` // "enabled", "disabled"
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// Metadata represents request metadata.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// ---- Anthropic Response Types ------------------------------------------------

// MessageResponse represents the Anthropic messages API response.
type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"` // "message"
	Role         string         `json:"role"` // "assistant"
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason,omitempty"` // "end_turn", "max_tokens", "tool_use", "stop_sequence"
	StopSequence string         `json:"stop_sequence,omitempty"`
	Usage        Usage          `json:"usage"`
}

// StreamingEvent represents a server-sent event from the streaming API.
type StreamingEvent struct {
	Type    string           `json:"type"` // "message_start", "content_block_start", "content_block_delta", "content_block_stop", "message_delta", "message_stop", "ping", "error"
	Index   *int             `json:"index,omitempty"`
	Delta   json.RawMessage  `json:"delta,omitempty"`
	Usage   *Usage           `json:"usage,omitempty"`
	Message *MessageResponse `json:"message,omitempty"`
	Error   *ErrorResponse   `json:"error,omitempty"`
}

// ErrorResponse represents an error response from the API.
type ErrorResponse struct {
	Type  string `json:"type"` // "error"
	Error Error  `json:"error"`
}

// Error represents the error details.
type Error struct {
	Type    string `json:"type"` // "invalid_request_error", "authentication_error", "permission_error", "not_found_error", "rate_limit_error", "api_error", "overloaded_error"
	Message string `json:"message"`
}

// Usage represents token usage information.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// CountTokensRequest represents a token counting request.
type CountTokensRequest struct {
	Model    string          `json:"model"`
	Messages []Message       `json:"messages"`
	System   any             `json:"system,omitempty"`
	Tools    []Tool          `json:"tools,omitempty"`
	Thinking *ThinkingConfig `json:"thinking,omitempty"`
}

// CountTokensResponse represents a token counting response.
type CountTokensResponse struct {
	InputTokens int `json:"input_tokens"`
}

// Model represents a model from the models API.
type Model struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Type        string   `json:"type"`
	CreatedAt   string   `json:"created_at"`
	Permissions []string `json:"permissions,omitempty"`
}

// ModelsResponse represents the models list response.
type ModelsResponse struct {
	Data    []Model `json:"data"`
	HasMore bool    `json:"has_more"`
	FirstID string  `json:"first_id"`
	LastID  string  `json:"last_id"`
}

// MessageStream handles streaming responses from Anthropic.
type MessageStream struct {
	resp   *http.Response
	reader *json.Decoder
	done   bool
	err    error
}

// Next returns the next streaming event.
func (s *MessageStream) Next() (*StreamingEvent, error) {
	if s.done {
		return nil, io.EOF
	}

	var event StreamingEvent
	if err := s.reader.Decode(&event); err != nil {
		s.done = true
		if err == io.EOF {
			return nil, io.EOF
		}
		s.err = err
		return nil, err
	}

	if event.Type == "message_stop" || event.Type == "error" {
		s.done = true
		if event.Type == "error" && event.Error != nil {
			s.err = errors.New(event.Error.Error.Message)
			return nil, ErrProviderError
		}
	}

	return &event, nil
}

// Close closes the stream.
func (s *MessageStream) Close() error {
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}

// Err returns any error that occurred during streaming.
func (s *MessageStream) Err() error {
	return s.err
}
