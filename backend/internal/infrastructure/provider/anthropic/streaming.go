// Package anthropic implements the Anthropic provider adapter.
package anthropic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
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
