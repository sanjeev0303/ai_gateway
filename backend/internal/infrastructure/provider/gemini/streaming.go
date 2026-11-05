// Package gemini implements the Gemini provider adapter.
package gemini

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
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
		// Keep the event type line; unparseable payloads are still passed through.
		return event, nil
	}
	return event, nil
}
