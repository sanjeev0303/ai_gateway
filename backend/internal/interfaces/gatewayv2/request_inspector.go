// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"context"
	"encoding/json"
	"io"

	"github.com/google/uuid"
)

// RequestInspectorImpl validates and extracts routing hints from requests.
type RequestInspectorImpl struct {
	// No state needed for basic inspection
}

// NewRequestInspector creates a new request inspector.
func NewRequestInspector() *RequestInspectorImpl {
	return &RequestInspectorImpl{}
}

// Inspect reads and parses the request body to extract routing hints.
// The body is consumed but can be re-read if needed by the caller.
func (i *RequestInspectorImpl) Inspect(ctx context.Context, body io.Reader) (InspectorHints, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return InspectorHints{}, err
	}

	var req map[string]any
	if err := json.Unmarshal(data, &req); err != nil {
		// Not JSON, return empty hints
		return InspectorHints{}, nil
	}

	hints := InspectorHints{
		RoutingTags: make(map[string]string),
	}

	// Extract model
	if model, ok := req["model"].(string); ok {
		hints.Model = model
		hints.RoutingTags["model"] = model
	}

	// Extract messages for token estimation
	if messages, ok := req["messages"].([]any); ok {
		hints.EstimatedInputTokens = estimateInputTokens(messages)
		hints.EstimatedOutputTokens = estimateOutputTokens(req)
	}

	// Extract stream flag
	if stream, ok := req["stream"].(bool); ok {
		hints.Stream = stream
		hints.RoutingTags["stream"] = boolToString(stream)
	}

	// Extract tools
	if tools, ok := req["tools"].([]any); ok {
		for _, t := range tools {
			if toolMap, ok := t.(map[string]any); ok {
				if name, ok := toolMap["name"].(string); ok {
					hints.Tools = append(hints.Tools, name)
				}
			}
		}
	}

	// Extract conversation/project metadata from custom fields
	if convID, ok := req["conversation_id"].(string); ok {
		if id, err := uuid.Parse(convID); err == nil {
			hints.ConversationID = &id
		}
	}

	if projID, ok := req["project_id"].(string); ok {
		if id, err := uuid.Parse(projID); err == nil {
			hints.ProjectID = &id
			hints.RoutingTags["project_id"] = projID
		}
	}

	// Extract custom routing tags
	if tags, ok := req["routing_tags"].(map[string]any); ok {
		for k, v := range tags {
			if str, ok := v.(string); ok {
				hints.RoutingTags[k] = str
			}
		}
	}

	// Validate required fields
	if hints.Model == "" {
		hints.Warnings = append(hints.Warnings, "model field is missing")
	}

	return hints, nil
}

// estimateInputTokens provides a rough estimate of input tokens from messages.
func estimateInputTokens(messages []any) int {
	total := 0
	for _, msg := range messages {
		if msgMap, ok := msg.(map[string]any); ok {
			if content, ok := msgMap["content"].(string); ok {
				total += len(content) / 4 // rough: 4 chars per token
			}
			// Handle content as array (multimodal)
			if contentArr, ok := msgMap["content"].([]any); ok {
				for _, c := range contentArr {
					if cMap, ok := c.(map[string]any); ok {
						if text, ok := cMap["text"].(string); ok {
							total += len(text) / 4
						}
					}
				}
			}
		}
	}
	if total == 0 {
		total = 256 // default minimum
	}
	return total
}

// estimateOutputTokens estimates output tokens from request parameters.
func estimateOutputTokens(req map[string]any) int {
	// Check max_tokens
	if maxTokens, ok := req["max_tokens"].(float64); ok {
		return int(maxTokens)
	}
	// Default based on model
	return 1024
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// StreamingInspector wraps a reader to allow inspection while streaming.
// This allows the body to be consumed once while still being available for forwarding.
type StreamingInspector struct {
	reader    io.Reader
	buffer    []byte
	inspected bool
	hints     InspectorHints
	err       error
}

func NewStreamingInspector(r io.Reader) *StreamingInspector {
	return &StreamingInspector{reader: r}
}

// Read implements io.Reader, buffering data for inspection.
func (s *StreamingInspector) Read(p []byte) (int, error) {
	n, err := s.reader.Read(p)
	if n > 0 {
		s.buffer = append(s.buffer, p[:n]...)
	}

	// Try to inspect once we have enough data
	if !s.inspected && len(s.buffer) > 1024 {
		s.inspect()
	}

	return n, err
}

// Hints returns the inspection hints (blocking until inspected).
func (s *StreamingInspector) Hints(ctx context.Context) (InspectorHints, error) {
	if !s.inspected {
		s.inspect()
	}
	return s.hints, s.err
}

// inspect parses the buffered data for hints.
func (s *StreamingInspector) inspect() {
	s.inspected = true
	if len(s.buffer) == 0 {
		return
	}

	var req map[string]any
	if err := json.Unmarshal(s.buffer, &req); err != nil {
		s.err = err
		return
	}

	hints := InspectorHints{
		RoutingTags: make(map[string]string),
	}

	if model, ok := req["model"].(string); ok {
		hints.Model = model
		hints.RoutingTags["model"] = model
	}

	if messages, ok := req["messages"].([]any); ok {
		hints.EstimatedInputTokens = estimateInputTokens(messages)
		hints.EstimatedOutputTokens = estimateOutputTokens(req)
	}

	if stream, ok := req["stream"].(bool); ok {
		hints.Stream = stream
		hints.RoutingTags["stream"] = boolToString(stream)
	}

	if tools, ok := req["tools"].([]any); ok {
		for _, t := range tools {
			if toolMap, ok := t.(map[string]any); ok {
				if name, ok := toolMap["name"].(string); ok {
					hints.Tools = append(hints.Tools, name)
				}
			}
		}
	}

	if convID, ok := req["conversation_id"].(string); ok {
		if id, err := uuid.Parse(convID); err == nil {
			hints.ConversationID = &id
		}
	}

	if projID, ok := req["project_id"].(string); ok {
		if id, err := uuid.Parse(projID); err == nil {
			hints.ProjectID = &id
			hints.RoutingTags["project_id"] = projID
		}
	}

	if tags, ok := req["routing_tags"].(map[string]any); ok {
		for k, v := range tags {
			if str, ok := v.(string); ok {
				hints.RoutingTags[k] = str
			}
		}
	}

	if hints.Model == "" {
		hints.Warnings = append(hints.Warnings, "model field is missing")
	}

	s.hints = hints
}

// PeekReader returns a reader that allows peeking at the stream.
// The original body is replaced with a tee reader.
func PeekReader(body io.ReadCloser) (*StreamingInspector, io.ReadCloser) {
	inspector := NewStreamingInspector(body)
	return inspector, &peekReadCloser{Reader: inspector, closer: body}
}

type peekReadCloser struct {
	io.Reader
	closer io.Closer
}

func (p *peekReadCloser) Close() error {
	return p.closer.Close()
}