// Package huggingface implements the Hugging Face Inference Providers adapter.
package huggingface

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ai-governance-saas/backend/internal/ports/provider"
)

// Client talks to Hugging Face Inference API and the Hugging Face Hub.
type Client struct {
	baseURL    *url.URL
	hubURL     *url.URL
	httpClient *http.Client
	userAgent  string
}

// New creates a new Hugging Face client.
func New(rawBaseURL, rawHubURL, userAgent string) (*Client, error) {
	if rawBaseURL == "" {
		rawBaseURL = defaultBaseURL
	}
	if rawHubURL == "" {
		rawHubURL = defaultHubURL
	}
	if userAgent == "" {
		userAgent = "Sentinel-AI-Gateway"
	}

	bu, err := url.Parse(rawBaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	hu, err := url.Parse(rawHubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid hub URL: %w", err)
	}

	return &Client{
		baseURL: bu,
		hubURL:  hu,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		userAgent: userAgent,
	}, nil
}

// ValidateCredentials verifies the API token against Hugging Face.
func (c *Client) ValidateCredentials(ctx context.Context, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ErrInvalidAPIKey
	}

	// 1. Try Hub whoami-v2 first (standard user/org token check)
	whoamiURL := strings.TrimRight(c.hubURL.String(), "/") + "/api/whoami-v2"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, whoamiURL, nil)
	if err == nil {
		c.setHeaders(req, apiKey)
		resp, err := c.httpClient.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				return provider.ErrInvalidCredentials
			}
		}
	}

	// 2. Fallback check for custom dedicated endpoints
	reqURL := strings.TrimRight(c.baseURL.String(), "/") + "/models"
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}
	c.setHeaders(req2, apiKey)
	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusUnauthorized || resp2.StatusCode == http.StatusForbidden {
		return provider.ErrInvalidCredentials
	}
	if resp2.StatusCode >= 400 && resp2.StatusCode != http.StatusNotFound {
		return fmt.Errorf("huggingface validation failed with status %d", resp2.StatusCode)
	}

	return nil
}

// HealthCheck proves reachability to Hugging Face.
func (c *Client) HealthCheck(ctx context.Context, apiKey string) error {
	checkURL := strings.TrimRight(c.baseURL.String(), "/") + "/models"
	if strings.Contains(c.baseURL.Host, "api-inference.huggingface.co") {
		checkURL = strings.TrimRight(c.hubURL.String(), "/") + "/api/models?limit=1"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checkURL, nil)
	if err != nil {
		return err
	}
	if apiKey != "" {
		c.setHeaders(req, apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	// Reachability succeeds on any HTTP status code received (including 401/403)
	return nil
}

// ListModels fetches available text generation models from Hugging Face.
func (c *Client) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	// 1. If using custom dedicated endpoint, query /models or /v1/models
	if !strings.Contains(c.baseURL.Host, "api-inference.huggingface.co") && !strings.Contains(c.baseURL.Host, "huggingface.co") {
		endpointModelsURL := strings.TrimRight(c.baseURL.String(), "/") + "/models"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointModelsURL, nil)
		if err == nil {
			c.setHeaders(req, apiKey)
			if resp, err := c.httpClient.Do(req); err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
					return nil, provider.ErrInvalidCredentials
				}
				if resp.StatusCode == http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					var data struct {
						Data []struct {
							ID string `json:"id"`
						} `json:"data"`
					}
					if err := json.Unmarshal(bodyBytes, &data); err == nil && len(data.Data) > 0 {
						out := make([]provider.Model, 0, len(data.Data))
						for _, m := range data.Data {
							if m.ID != "" {
								out = append(out, provider.Model{
									ID:           m.ID,
									Provider:     "huggingface",
									DisplayName:  m.ID,
									Capabilities: []string{"text-generation", "chat"},
								})
							}
						}
						return out, nil
					}
					var directItems []struct {
						ID string `json:"id"`
					}
					if err := json.Unmarshal(bodyBytes, &directItems); err == nil {
						out := make([]provider.Model, 0, len(directItems))
						for _, m := range directItems {
							if m.ID != "" {
								out = append(out, provider.Model{
									ID:           m.ID,
									Provider:     "huggingface",
									DisplayName:  m.ID,
									Capabilities: []string{"text-generation", "chat"},
								})
							}
						}
						return out, nil
					}
				}
			}
		}
	}

	// 2. Query Hugging Face Hub for popular text generation models
	hubAPIURL := fmt.Sprintf("%s/api/models?pipeline_tag=text-generation&sort=trendingScore&direction=-1&limit=20", strings.TrimRight(c.hubURL.String(), "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hubAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("huggingface: hub models request: %w", err)
	}
	if apiKey != "" {
		c.setHeaders(req, apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("huggingface: hub models query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, provider.ErrInvalidCredentials
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("%w: hub models returned %d", provider.ErrDiscoveryUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("huggingface: hub models returned unexpected status %d", resp.StatusCode)
	}

	var items []HubModelItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("huggingface: decode hub models: %w", err)
	}

	result := make([]provider.Model, 0, len(items))
	for _, it := range items {
		if it.ID == "" {
			continue
		}
		result = append(result, provider.Model{
			ID:           it.ID,
			Provider:     "huggingface",
			DisplayName:  it.ID,
			Capabilities: []string{"text-generation", "chat"},
		})
	}

	return result, nil
}

// CreateMessage sends a non-streaming chat completion request.
func (c *Client) CreateMessage(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	req.Stream = false
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	completionsURL := c.chatCompletionsURL()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, completionsURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode huggingface response: %w", err)
	}

	return &chatResp, nil
}

// CreateMessageStream sends a streaming chat completion request.
func (c *Client) CreateMessageStream(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*MessageStream, error) {
	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	completionsURL := c.chatCompletionsURL()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, completionsURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, c.parseError(resp)
	}

	return &MessageStream{
		resp:    resp,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// CountTokens estimates or counts tokens for a request.
func (c *Client) CountTokens(ctx context.Context, apiKey string, req *ChatCompletionRequest) (*CountTokensResponse, error) {
	totalChars := 0
	for _, m := range req.Messages {
		totalChars += len(m.Content) + len(m.Role)
	}
	// Heuristic 1 token ~= 4 chars with a floor of 1
	tokens := totalChars / 4
	if tokens < 1 && len(req.Messages) > 0 {
		tokens = 1
	}
	return &CountTokensResponse{
		TotalTokens: tokens,
	}, nil
}

func (c *Client) chatCompletionsURL() string {
	base := strings.TrimRight(c.baseURL.String(), "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func (c *Client) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("User-Agent", c.userAgent)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func (c *Client) parseError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return provider.ErrInvalidCredentials
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return ErrOverloaded
	}

	var errResp struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(data, &errResp); err == nil && errResp.Error != nil {
		return fmt.Errorf("huggingface error: %v", errResp.Error)
	}
	return fmt.Errorf("huggingface returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
}

// MessageStream handles SSE streaming responses from Hugging Face Inference API.
type MessageStream struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
	err     error
}

// Next reads and parses the next SSE streaming event.
func (s *MessageStream) Next() (*ChatCompletionStreamResponse, error) {
	if s.done {
		return nil, io.EOF
	}

	for s.scanner.Scan() {
		line := strings.TrimSpace(s.scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			s.done = true
			return nil, io.EOF
		}

		var event ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// If finish reason was present, flag done
		for _, ch := range event.Choices {
			if ch.FinishReason != nil && *ch.FinishReason != "" {
				s.done = true
				break
			}
		}

		return &event, nil
	}

	s.done = true
	if err := s.scanner.Err(); err != nil {
		s.err = err
		return nil, err
	}
	return nil, io.EOF
}

// Close closes the underlying HTTP response body.
func (s *MessageStream) Close() error {
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil
}

// Err returns the last stream error.
func (s *MessageStream) Err() error {
	return s.err
}
