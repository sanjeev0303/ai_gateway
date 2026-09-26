// Package anthropic implements the Anthropic provider adapter.
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UsageCollector captures usage from streaming events.
type UsageCollector struct {
	inputTokens              int
	outputTokens             int
	cacheCreationInputTokens int
	cacheReadInputTokens     int
	finalUsage               *Usage
	streamComplete           bool
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

	switch event.Type {
	case "message_start":
		if event.Message != nil {
			c.inputTokens = event.Message.Usage.InputTokens
			c.outputTokens = event.Message.Usage.OutputTokens
			c.cacheCreationInputTokens = event.Message.Usage.CacheCreationInputTokens
			c.cacheReadInputTokens = event.Message.Usage.CacheReadInputTokens
		}
	case "message_delta":
		if event.Usage != nil {
			// Delta usage adds to existing
			c.outputTokens += event.Usage.OutputTokens
		}
	case "message_stop":
		c.streamComplete = true
	case "content_block_delta":
		// For tool use deltas, we might track output tokens incrementally
		// But Anthropic typically sends usage in message_delta
	}
	return nil
}

// FinalUsage returns the final usage after the stream completes.
func (c *UsageCollector) FinalUsage() *Usage {
	if c.finalUsage != nil {
		return c.finalUsage
	}
	return &Usage{
		InputTokens:              c.inputTokens,
		OutputTokens:             c.outputTokens,
		CacheCreationInputTokens: c.cacheCreationInputTokens,
		CacheReadInputTokens:     c.cacheReadInputTokens,
	}
}

// SetFinalUsage sets the final usage (for testing or manual override).
func (c *UsageCollector) SetFinalUsage(usage *Usage) {
	c.finalUsage = usage
}

// IsComplete returns true if the stream has completed.
func (c *UsageCollector) IsComplete() bool {
	return c.streamComplete
}

// Reset resets the collector for reuse.
func (c *UsageCollector) Reset() {
	c.inputTokens = 0
	c.outputTokens = 0
	c.cacheCreationInputTokens = 0
	c.cacheReadInputTokens = 0
	c.finalUsage = nil
	c.streamComplete = false
}

// ---- Streaming Proxy with Usage Capture ------------------------------------

// StreamingProxy forwards requests to Anthropic and captures usage from the stream.
type StreamingProxy struct {
	client     *http.Client
	baseURL    string
	version    string
	logger     interface {
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
				DisableCompression: true,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
		baseURL: baseURL,
		version: defaultAnthropicVersion,
	}
	for _, opt := range opts {
		opt(sp)
	}
	return sp
}

// Forward forwards a streaming request to Anthropic and captures usage.
func (sp *StreamingProxy) Forward(ctx context.Context, w http.ResponseWriter, r *http.Request, apiKey string, providerCredentialID string) error {
	// Read and parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Parse the request to extract model and other fields for logging
	var req MessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// If we can't parse, still forward
		req = MessageRequest{}
	}

	// Build the upstream request
	upstreamReq, err := http.NewRequestWithContext(ctx, r.Method, sp.baseURL+"/v1/messages", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create upstream request: %w", err)
	}

	// Copy headers (excluding hop-by-hop and auth)
	copyForwardHeaders(upstreamReq.Header, r.Header)
	upstreamReq.Header.Set("x-api-key", apiKey)
	upstreamReq.Header.Set("anthropic-version", defaultAnthropicVersion)
	upstreamReq.Header.Set("anthropic-beta", "output-128k-2025-02-19")
	upstreamReq.Header.Set("Accept", "text/event-stream")

	// Make the upstream request
	resp, err := http.DefaultClient.Do(upstreamReq)
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
	if usageCollector != nil && sp.usageCallback != nil {
		finalUsage := usageCollector.FinalUsage()
		if finalUsage != nil {
			sp.usageCallback(finalUsage, providerCredentialID)
		}
	}

	return usageErr
}

// streamWithUsageCapture streams the SSE response and captures usage.
func (sp *StreamingProxy) streamWithUsageCapture(ctx context.Context, w http.ResponseWriter, body io.Reader, collector *UsageCollector) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("ResponseWriter does not support flushing")
	}

	decoder := NewSSEDecoder(body)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		event, err := decoder.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Process event for usage
		if err := collector.ProcessEvent(event); err != nil {
			// Log but continue streaming
			fmt.Printf("Warning: failed to process event for usage: %v\n", err)
		}

		// Write event to client
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}

		// Write SSE format
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
			return err
		}

		if flusher != nil {
			flusher.Flush()
		}

		if event.Type == "message_stop" || event.Type == "error" {
			return nil
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
	var msgResp MessageResponse
	if err := json.Unmarshal(respBody, &msgResp); err == nil {
		usage := &msgResp.Usage
		collector.SetFinalUsage(usage)
	}

	// Write to client
	_, err2 := w.Write(respBody)
	return err2
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
	var eventData []byte

	for {
		line, err := d.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF && data.Len() == 0 {
				return nil, io.EOF
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
		} else if bytes.HasPrefix(line, []byte("id:")) || bytes.HasPrefix(line, []byte("retry:")) {
			// Ignore id and retry fields
		}
	}

	if data.Len() == 0 {
		return nil, io.EOF
	}

	eventData = []byte(data.String())

	var event StreamingEvent
	event.Type = eventType
	if err := json.Unmarshal(eventData, &event); err != nil {
		// If unmarshal fails, try to unmarshal as delta
		event.Delta = eventData
	}

	return &event, nil
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
		"x-api-key":           true,
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

// copyResponseHeaders copies end-to-end headers from src to dst, dropping hop-by-hop
// headers and framing that the Go response writer manages itself.
func copyResponseHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"connection":        true,
		"keep-alive":        true,
		"proxy-authenticate": true,
		"proxy-authorization": true,
		"te":                true,
		"trailer":           true,
		"transfer-encoding": true,
		"upgrade":           true,
		"content-length":    true,
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