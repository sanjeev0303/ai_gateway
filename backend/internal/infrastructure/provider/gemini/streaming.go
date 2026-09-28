// Package gemini implements the Gemini provider adapter.
package gemini

import (
	"bufio"
	"bytes"
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

// UsageCollector captures usage from streaming events.
type UsageCollector struct {
	finalUsage     *Usage
	streamComplete bool
	hasUsage       bool
}

// NewUsageCollector creates a new usage collector.
func NewUsageCollector() *UsageCollector {
	return &UsageCollector{}
}

// ProcessEvent processes a streaming event and extracts usage.
func (c *UsageCollector) ProcessEvent(event *StreamingEvent) error {
	if c.streamComplete {
		return nil
	}

	// Gemini sends cumulative usageMetadata on the final chunk of the stream.
	if event.UsageMetadata != nil && event.UsageMetadata.TotalTokenCount > 0 {
		c.finalUsage = &Usage{
			PromptTokens:     event.UsageMetadata.PromptTokenCount,
			CompletionTokens: event.UsageMetadata.CandidatesTokenCount,
			CachedTokens:     event.UsageMetadata.CachedContentTokenCount,
			ThinkingTokens:   event.UsageMetadata.ThoughtsTokenCount,
			TotalTokens:      event.UsageMetadata.TotalTokenCount,
		}
		c.hasUsage = true
		c.streamComplete = true
	}
	return nil
}

// FinalUsage returns the final usage after the stream completes.
func (c *UsageCollector) FinalUsage() *Usage {
	if c.finalUsage != nil {
		return c.finalUsage
	}
	return &Usage{}
}

// SetFinalUsage sets the final usage (for testing or manual override).
func (c *UsageCollector) SetFinalUsage(usage *Usage) {
	c.finalUsage = usage
	if usage != nil {
		c.hasUsage = true
	}
}

// HasUsage reports whether any usage was captured.
func (c *UsageCollector) HasUsage() bool {
	return c.hasUsage
}

// IsComplete returns true if the stream has completed.
func (c *UsageCollector) IsComplete() bool {
	return c.streamComplete
}

// Reset resets the collector for reuse.
func (c *UsageCollector) Reset() {
	c.finalUsage = nil
	c.streamComplete = false
	c.hasUsage = false
}

// ---- Streaming Proxy with Usage Capture ------------------------------------

// StreamingProxy forwards requests to Gemini and captures usage from the stream.
type StreamingProxy struct {
	client  *http.Client
	baseURL string
	version string
	logger  interface {
		Debug(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
	}
	usageCallback func(usage *Usage, providerCredentialID string)
}

// StreamingProxyOption configures the streaming proxy.
type StreamingProxyOption func(*StreamingProxy)

// WithUsageCallback sets a callback to be called when usage is captured.
func WithUsageCallback(cb func(usage *Usage, providerCredentialID string)) func(*StreamingProxy) {
	return func(sp *StreamingProxy) {
		sp.usageCallback = cb
	}
}

// NewStreamingProxy creates a new streaming proxy for the given base URL.
// An empty baseURL falls back to the provider default.
func NewStreamingProxy(baseURL string, opts ...func(*StreamingProxy)) *StreamingProxy {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	sp := &StreamingProxy{
		client: &http.Client{
			Transport: &http.Transport{
				DisableCompression:    true,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
		baseURL: baseURL,
		version: defaultVersion,
	}
	for _, opt := range opts {
		opt(sp)
	}
	return sp
}

// Forward forwards a streaming request to Gemini and captures usage.
func (sp *StreamingProxy) Forward(ctx context.Context, w http.ResponseWriter, r *http.Request, apiKey string, providerCredentialID string) error {
	// Read and parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Gemini routes by model in the URL path, e.g.
	// /v1beta/models/gemini-2.5-flash:streamGenerateContent. Preserve the
	// provider-native path so the upstream receives the same shape. When the
	// incoming path is not Gemini-shaped, fall back to the model carried in the
	// request body.
	path := r.URL.Path
	if !strings.Contains(path, "/models/") {
		var bodyReq struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(body, &bodyReq) == nil && bodyReq.Model != "" {
			path = sp.modelPath(bodyReq.Model, "streamGenerateContent")
		} else {
			return fmt.Errorf("gemini: cannot determine model from request path %q", r.URL.Path)
		}
	}

	// Ensure SSE output mode for streaming.
	upstreamURL := sp.baseURL + path
	if !strings.Contains(upstreamURL, "alt=sse") {
		sep := "?"
		if strings.Contains(upstreamURL, "?") {
			sep = "&"
		}
		upstreamURL += sep + "alt=sse"
	}

	upstreamReq, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create upstream request: %w", err)
	}

	// Copy headers (excluding hop-by-hop and auth headers)
	copyForwardHeaders(upstreamReq.Header, r.Header)
	upstreamReq.Header.Set("x-goog-api-key", apiKey)
	upstreamReq.Header.Set("Accept", "text/event-stream")

	// Make the upstream request
	resp, err := sp.client.Do(upstreamReq)
	if err != nil {
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	// Copy response headers
	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	// Check if this is a streaming response
	contentType := resp.Header.Get("Content-Type")
	isStream := strings.Contains(contentType, "text/event-stream")

	usageCollector := NewUsageCollector()

	// Stream the response
	var usageErr error
	if isStream {
		usageErr = sp.streamWithUsageCapture(ctx, w, resp.Body, usageCollector)
	} else {
		// Non-streaming: copy body and extract usage from response
		usageErr = sp.copyNonStreamingBody(w, resp.Body, usageCollector)
	}

	// Call usage callback if set
	if sp.usageCallback != nil && usageCollector.HasUsage() {
		sp.usageCallback(usageCollector.FinalUsage(), providerCredentialID)
	}

	return usageErr
}

// streamWithUsageCapture streams the SSE response byte-exact and captures usage.
func (sp *StreamingProxy) streamWithUsageCapture(ctx context.Context, w http.ResponseWriter, body io.Reader, collector *UsageCollector) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("ResponseWriter does not support flushing")
	}

	reader := bufio.NewReader(body)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			// Forward raw bytes verbatim (provider-native passthrough).
			if _, werr := w.Write(line); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}

			// Parse SSE data lines for usage without altering the payload.
			trimmed := bytes.TrimSpace(line)
			if bytes.HasPrefix(trimmed, []byte("data:")) {
				payload := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
				if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
					continue
				}
				var event StreamingEvent
				if err := json.Unmarshal(payload, &event); err == nil {
					if perr := collector.ProcessEvent(&event); perr != nil {
						fmt.Printf("Warning: failed to process event for usage: %v\n", perr)
					}
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// copyNonStreamingBody copies a non-streaming response body and extracts usage.
func (sp *StreamingProxy) copyNonStreamingBody(w http.ResponseWriter, body io.Reader, collector *UsageCollector) error {
	respBody, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	// Extract usage from response
	var genResp GenerateContentResponse
	if err := json.Unmarshal(respBody, &genResp); err == nil && genResp.UsageMetadata != nil {
		collector.SetFinalUsage(&Usage{
			PromptTokens:     genResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: genResp.UsageMetadata.CandidatesTokenCount,
			CachedTokens:     genResp.UsageMetadata.CachedContentTokenCount,
			ThinkingTokens:   genResp.UsageMetadata.ThoughtsTokenCount,
			TotalTokens:      genResp.UsageMetadata.TotalTokenCount,
		})
	}

	// Write to client
	_, err = w.Write(respBody)
	return err
}

// ---- SSE Decoder -------------------------------------------------------------

// SSEDecoder decodes server-sent events.
type SSEDecoder struct {
	reader *bufio.Reader
}

// NewSSEDecoder creates a new SSE decoder.
func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{
		reader: bufio.NewReader(r),
	}
}

// Next returns the next SSE event.
func (d *SSEDecoder) Next() (*StreamingEvent, error) {
	var eventType string
	var data strings.Builder

	for {
		line, err := d.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && data.Len() == 0 {
				return nil, io.EOF
			}
			if err == io.EOF && data.Len() > 0 {
				break
			}
			return nil, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			// Empty line = end of event
			if data.Len() > 0 {
				break
			}
			continue
		}

		if bytes.HasPrefix(line, []byte("event:")) {
			eventType = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("event:"))))
		} else if bytes.HasPrefix(line, []byte("data:")) {
			data.Write(bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:"))))
			data.WriteByte('\n')
		}
	}

	if data.Len() == 0 {
		return nil, io.EOF
	}

	eventData := []byte(strings.TrimSpace(data.String()))
	if bytes.Equal(eventData, []byte("[DONE]")) {
		return nil, io.EOF
	}

	event := &StreamingEvent{}
	if eventType != "" {
		event.EventType = eventType
	}
	if err := json.Unmarshal(eventData, event); err != nil {
		// Keep the event type line; unparseable payloads are still passed
		// through byte-exact by the proxy.
		return event, nil
	}
	return event, nil
}

// copyForwardHeaders copies end-to-end headers from src to dst, dropping hop-by-hop
// and authentication headers.
func copyForwardHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"host":                true,
		"content-length":      true,
		"authorization":       true,
		"x-goog-api-key":      true,
	}

	for key, values := range src {
		if hopByHop[strings.ToLower(key)] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// copyResponseHeaders copies end-to-end headers from upstream to client.
func copyResponseHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailer":             true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"content-length":      true,
	}

	for key, values := range src {
		if hopByHop[strings.ToLower(key)] {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

func (sp *StreamingProxy) modelPath(model, action string) string {
	return fmt.Sprintf("/%s/models/%s:%s", sp.version, url.PathEscape(model), action)
}
