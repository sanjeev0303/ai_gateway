// Package gemini implements the Gemini provider client.
package gemini

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

const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Client talks to the Gemini API.
type Client struct {
	baseURL   *url.URL
	client    *http.Client
	version   string
	userAgent string
}

// New creates a new Gemini client.
func New(baseAddr, version, userAgent string) (*Client, error) {
	base := baseAddr
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("gemini: parse base url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("gemini: base url must include scheme and host")
	}
	if version == "" {
		version = defaultVersion
	}
	return &Client{
		baseURL:   u,
		client:    &http.Client{Timeout: 120 * time.Second},
		version:   version,
		userAgent: userAgent,
	}, nil
}

// modelPath returns the API path for a model action, e.g.
// /v1beta/models/gemini-2.5-flash:generateContent.
func (c *Client) modelPath(model, action string) string {
	return fmt.Sprintf("/%s/models/%s:%s", c.version, url.PathEscape(model), action)
}

// ValidateCredentials verifies the key against GET /v1beta/models.
func (c *Client) ValidateCredentials(ctx context.Context, apiKey string) error {
	resp, err := c.do(ctx, apiKey, "/"+c.version+"/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrInvalidAPIKey
	default:
		// Gemini returns 400 for bad keys with status UNAUTHENTICATED/PERMISSION_DENIED.
		if status, ok := c.errorStatus(resp); ok &&
			(status == "UNAUTHENTICATED" || status == "PERMISSION_DENIED") {
			return ErrInvalidAPIKey
		}
		return fmt.Errorf("gemini: unexpected status %d validating credentials", resp.StatusCode)
	}
}

// HealthCheck proves connectivity to Gemini.
func (c *Client) HealthCheck(ctx context.Context, apiKey string) error {
	resp, err := c.do(ctx, apiKey, "/"+c.version+"/models", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Any response (including 401) proves connectivity
	return nil
}

// ListModels returns the model IDs from GET /v1beta/models.
func (c *Client) ListModels(ctx context.Context, apiKey string) ([]provider.Model, error) {
	resp, err := c.do(ctx, apiKey, "/"+c.version+"/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrInvalidAPIKey
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini: unexpected status %d listing models", resp.StatusCode)
	}
	var body ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("gemini: decode models: %w", err)
	}
	models := make([]provider.Model, 0, len(body.Models))
	for _, m := range body.Models {
		models = append(models, provider.Model{
			ID:           strings.TrimPrefix(m.Name, "models/"),
			Provider:     domain.ProviderKindGemini,
			DisplayName:  m.DisplayName,
			Capabilities: m.SupportedGenerationMethods,
			ContextLimit: m.InputTokenLimit,
		})
	}
	return models, nil
}

// CreateMessage sends a non-streaming generateContent request.
func (c *Client) CreateMessage(ctx context.Context, apiKey string, req *GenerateContentRequest) (*GenerateContentResponse, error) {
	if req.Model == "" {
		return nil, ErrInvalidRequest
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+c.modelPath(req.Model, "generateContent"), strings.NewReader(string(body)))
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

	var respBody GenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}
	return &respBody, nil
}

// CreateMessageStream sends a streaming generateContent request.
func (c *Client) CreateMessageStream(ctx context.Context, apiKey string, req *GenerateContentRequest) (*MessageStream, error) {
	if req.Model == "" {
		return nil, ErrInvalidRequest
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+c.modelPath(req.Model, "streamGenerateContent")+"?alt=sse", strings.NewReader(string(body)))
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
		resp:    resp,
		decoder: NewSSEDecoder(resp.Body),
	}, nil
}

// CountTokens counts tokens for a request against the countTokens endpoint.
func (c *Client) CountTokens(ctx context.Context, apiKey string, model string, req *CountTokensRequest) (*CountTokensResponse, error) {
	if model == "" {
		return nil, ErrInvalidRequest
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL.String()+c.modelPath(model, "countTokens"), strings.NewReader(string(body)))
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

// MessageStream handles streaming responses from Gemini.
type MessageStream struct {
	resp    *http.Response
	decoder *SSEDecoder
	done    bool
	err     error
}

// Next returns the next streaming event.
func (s *MessageStream) Next() (*StreamingEvent, error) {
	if s.done {
		return nil, io.EOF
	}
	event, err := s.decoder.Next()
	if err != nil {
		s.done = true
		if err == io.EOF {
			return nil, io.EOF
		}
		s.err = err
		return nil, err
	}

	// Check for error events - Gemini embeds errors in the chunk payload
	if event.Error != nil {
		s.done = true
		s.err = errors.New(event.Error.Message)
		return nil, ErrProviderError
	}

	return event, nil
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

// ---- Helpers -----------------------------------------------------------------

func (c *Client) setHeaders(req *http.Request, apiKey string) {
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
}

// errorStatus decodes the Gemini error envelope status field.
func (c *Client) errorStatus(resp *http.Response) (string, bool) {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return "", false
	}
	return errResp.Error.Status, errResp.Error.Status != ""
}

func (c *Client) parseError(resp *http.Response) error {
	var errResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return ErrProviderError
	}

	// Gemini maps most auth failures to 400 with a status field in the body.
	if errResp.Error.Status == "UNAUTHENTICATED" || errResp.Error.Status == "PERMISSION_DENIED" {
		return ErrInvalidAPIKey
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
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("x-goog-api-key", apiKey)
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: %s %s: %w", req.Method, u.String(), err)
	}
	return resp, nil
}
