// Package openai implements the OpenAI provider adapter.
package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// UsageCollector captures usage from streaming events.
type UsageCollector struct {
	promptTokens     int
	completionTokens int
	cacheTokens      int
	reasoningTokens  int
	finalUsage       *Usage
	streamComplete   bool
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

	switch event.Object {
	case "chat.completion.chunk":
		for _, choice := range event.Choices {
			if choice.FinishReason != "" {
				c.streamComplete = true
			}
		}
		// OpenAI sends usage in the final chunk only, but may also send incremental usage
		if event.Usage != nil {
			c.promptTokens = event.Usage.PromptTokens
			c.completionTokens += event.Usage.CompletionTokens
			if event.Usage.PromptTokensDetails != nil {
				c.cacheTokens = event.Usage.PromptTokensDetails.CachedTokens
			}
			if event.Usage.CompletionTokensDetails != nil {
				c.reasoningTokens = event.Usage.CompletionTokensDetails.ReasoningTokens
			}
		}
	}
	return nil
}

// FinalUsage returns the final usage after the stream completes.
func (c *UsageCollector) FinalUsage() *Usage {
	if c.finalUsage != nil {
		return c.finalUsage
	}
	return &Usage{
		PromptTokens:     c.promptTokens,
		CompletionTokens: c.completionTokens,
		PromptTokensDetails: &PromptTokensDetails{
			CachedTokens: c.cacheTokens,
		},
		CompletionTokensDetails: &CompletionTokensDetails{
			ReasoningTokens: c.reasoningTokens,
		},
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
	c.promptTokens = 0
	c.completionTokens = 0
	c.cacheTokens = 0
	c.reasoningTokens = 0
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

	eventData := []byte(strings.TrimSpace(data.String()))
	if bytes.Equal(eventData, []byte("[DONE]")) {
		return nil, io.EOF
	}

	var event StreamingEvent
	event.Object = eventType
	if err := json.Unmarshal(eventData, &event); err != nil {
		// If unmarshal fails, try to unmarshal as delta
		event.Delta = eventData
	}

	return &event, nil
}
