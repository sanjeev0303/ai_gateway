// Package anthropic implements the provider port for Anthropic's Messages
// API (M4 scope: validation, health, model catalog). Streaming is exercised
// by the gateway's byte-level reverse proxy, not here.
package anthropic

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
)

const defaultBaseURL = "https://api.anthropic.com"

// Client talks to the Anthropic API.
type Client struct {
	baseURL   *url.URL
	client    *http.Client
	apiKey    string
	version   string
	userAgent string
}

// New returns a Client for baseAddr ("" => https://api.anthropic.com).
// version is the anthropic-version header; userAgent identifies the gateway
// to Anthropic support.
func New(baseAddr, version, userAgent string) (*Client, error) {
	base := baseAddr
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("anthropic: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("anthropic: base url must include scheme and host")
	}
	if version == "" {
		version = "2023-06-01"
	}
	return &Client{
		baseURL:   u,
		client:    &http.Client{Timeout: 30 * time.Second},
		version:   version,
		userAgent: userAgent,
	}, nil
}

// ValidateCredentials verifies the key against GET /v1/models. Any defined
// per-key response (200, 401, 403) proves the credentials are answered
// distinctly; only 401/403 means invalid. Other statuses are surfaced as
// transport errors.
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
		return fmt.Errorf("anthropic: unexpected status %d validating credentials", resp.StatusCode)
	}
}

// HealthCheck succeeds on any HTTP response — 401 proves the endpoint is
// reachable and routing correctly. Only transport-level failures are
// unhealthy.
func (c *Client) HealthCheck(ctx context.Context, apiKey string) error {
	resp, err := c.do(ctx, apiKey, "/v1/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Partial read so an unusable body still surfaces.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return nil
}

// ListModels returns the model IDs from GET /v1/models.
func (c *Client) ListModels(ctx context.Context, apiKey string) ([]Model, error) {
	resp, err := c.do(ctx, apiKey, "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidAPIKey
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNotImplemented {
		return nil, fmt.Errorf("%w: endpoint returned %d", ErrDiscoveryUnavailable, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic: unexpected status %d listing models", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Type        string `json:"type"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("anthropic: decode models: %w", err)
	}
	models := make([]Model, 0, len(body.Data))
	for _, m := range body.Data {
		models = append(models, Model{
			ID:          m.ID,
			DisplayName: m.DisplayName,
			Type:        m.Type,
			CreatedAt:   m.CreatedAt,
		})
	}
	return models, nil
}

func (c *Client) do(ctx context.Context, apiKey, path string, body io.Reader) (*http.Response, error) {
	u := *c.baseURL
	u.Path = path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", c.version)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: %s %s: %w", req.Method, u.String(), err)
	}
	return resp, nil
}

// CreateMessage sends a non-streaming message request.
func (c *Client) CreateMessage(ctx context.Context, apiKey string, req *MessageRequest) (*MessageResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+"/v1/messages", strings.NewReader(string(body)))
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

	var respBody MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}

// CreateMessageStream sends a streaming message request.
func (c *Client) CreateMessageStream(ctx context.Context, apiKey string, req *MessageRequest) (*MessageStream, error) {
	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+"/v1/messages", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	c.setHeaders(httpReq, apiKey)
	// Request streaming
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

// CountTokens counts tokens for a request.
func (c *Client) CountTokens(ctx context.Context, apiKey string, req *CountTokensRequest) (*CountTokensResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+"/v1/messages/count_tokens", strings.NewReader(string(body)))
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

	var respBody CountTokensResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}

// setHeaders sets the required headers for Anthropic API requests.
func (c *Client) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", c.version)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Include beta headers for thinking and other beta features
	req.Header.Set("anthropic-beta", "output-128k-2025-02-19")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
}

// parseError parses an error response from the Anthropic API.
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
