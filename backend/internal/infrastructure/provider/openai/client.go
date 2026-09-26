// Package openai implements the OpenAI provider client.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/provider"
)

const defaultBaseURL = "https://api.openai.com"

// Client talks to the OpenAI API.
type Client struct {
	baseURL   *url.URL
	client    *http.Client
	version   string
	userAgent string
}

// New creates a new OpenAI client.
func New(baseAddr, version, userAgent string) (*Client, error) {
	base := baseAddr
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("openai: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("openai: base url must include scheme and host")
	}
	if version == "" {
		version = "2024-08-01-preview"
	}
	return &Client{
		baseURL:   u,
		client:    &http.Client{Timeout: 120 * time.Second},
		version:   version,
		userAgent: userAgent,
	}, nil
}

// ValidateCredentials verifies the key against GET /v1/models.
func (c *Client) ValidateCredentials(ctx context.Context, apiKey string) error {
	resp, err := c.do(ctx, apiKey, "/v1/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidAPIKey
	default:
		return fmt.Errorf("openai: unexpected status %d validating credentials", resp.StatusCode)
	}
}

// HealthCheck proves connectivity to OpenAI.
func (c *Client) HealthCheck(ctx context.Context, apiKey string) error {
	resp, err := c.do(ctx, apiKey, "/v1/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Any response (including 401) proves connectivity
	return nil
}

// ListModels returns the model IDs from GET /v1/models.
func (c *Client) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	resp, err := c.do(ctx, apiKey, "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: unexpected status %d listing models", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("openai: decode models: %w", err)
	}
	models := make([]provider.Model, 0, len(body.Data))
	for _, m := range body.Data {
		models = append(models, provider.Model{ID: m.ID, Provider: domain.ProviderKindOpenAI})
	}
	return models, nil
}

// CreateMessage sends a non-streaming chat completion request.
func (c *Client) CreateMessage(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, apiKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var respBody ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}

// CreateMessageStream sends a streaming chat completion request.
func (c *Client) CreateMessageStream(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*MessageStream, error) {
	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.parseError(resp)
	}

	return &MessageStream{
		resp:   resp,
		reader: json.NewDecoder(resp.Body),
	}, nil
}

// CountTokens counts tokens for a request (OpenAI doesn't have a separate count_tokens endpoint,
// so we estimate or use the model's tokenizer - for now we return estimated values).
func (c *Client) CountTokens(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*CountTokensResponse, error) {
	// OpenAI doesn't have a public count_tokens endpoint like Anthropic.
	// We can estimate based on the request.
	// For now, return a reasonable estimate.
	return &CountTokensResponse{
		PromptTokens: estimateTokens(req.Messages, req.Model),
	}, nil
}

// MessageStream handles streaming responses from OpenAI.
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

	// Check for error events - OpenAI uses "error" in the object field
	if event.Object == "error" && event.Error != nil {
		s.done = true
		s.err = errors.New(event.Error.Message)
		return nil, ErrProviderError
	}

	if event.Object == "chat.completion.chunk" {
		// Check for finish_reason in choices
		for _, choice := range event.Choices {
			if choice.FinishReason != "" {
				s.done = true
				break
			}
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

// CountTokensResponse represents a token counting response.
type CountTokensResponse struct {
	PromptTokens int `json:"prompt_tokens"`
}

// estimateTokens provides a rough token count estimation.
func estimateTokens(messages []ChatMessage, model string) int {
	total := 0
	for _, msg := range messages {
		switch c := msg.Content.(type) {
		case string:
			total += len(c) / 4
		case []ContentPart:
			for _, part := range c {
				if part.Text != "" {
					total += len(part.Text) / 4
				}
			}
		}
	}
	if total == 0 {
		total = 256 // default minimum
	}
	return total
}

// ---- Helpers -----------------------------------------------------------------

func (c *Client) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenAI-Version", c.version)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
}

func (c *Client) parseError(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return ErrProviderError
	}

	switch resp.StatusCode {
	case http.StatusBadRequest:
		return ErrInvalidRequest
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidAPIKey
	case http.StatusNotFound:
		return ErrModelNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusServiceUnavailable:
		return ErrOverloaded
	case http.StatusInternalServerError:
		return ErrProviderError
	default:
		if errResp.Error.Message != "" {
			return errors.New(errResp.Error.Message)
		}
		return ErrProviderError
	}
}

func (c *Client) do(ctx context.Context, apiKey, path string, body io.Reader) (*http.Response, error) {
	u := *c.baseURL
	u.Path = path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("OpenAI-Version", c.version)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %s %s: %w", req.Method, u.String(), err)
	}
	return resp, nil
}