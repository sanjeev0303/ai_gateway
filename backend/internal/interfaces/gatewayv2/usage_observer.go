package gatewayv2

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// StreamUsage is provider-reported token usage observed while forwarding a
// response body. Reported is true only when the upstream actually sent usage;
// otherwise the gateway falls back to pre-flight estimates.
type StreamUsage struct {
	Provider         string
	InputTokens      int
	OutputTokens     int
	CacheReadTokens  int
	CacheWriteTokens int
	Reported         bool
	FirstTokenAt     time.Time
}

const (
	// maxObservedFrame bounds a single SSE data line. A usage frame is a few
	// hundred bytes; anything larger is a content delta we do not need.
	maxObservedFrame = 1 << 20
	// maxObservedBody bounds the copy kept for a non-streaming JSON body, whose
	// usage sits at the top level. Non-streaming bodies beyond this fall back to
	// estimates rather than growing the proxy's memory without bound.
	maxObservedBody = 1 << 20
)

// usageObserver watches forwarded bytes for provider-reported usage without
// altering them. It is not concurrency-safe: one observer belongs to one
// response, fed by that response's single copy loop.
type usageObserver struct {
	sse     bool
	frame   []byte
	body    []byte
	dropped bool
	usage   StreamUsage
}

// newUsageObserver selects the parsing strategy from the response content type.
func newUsageObserver(contentType string) *usageObserver {
	return &usageObserver{sse: strings.Contains(contentType, "text/event-stream")}
}

// observingReader passes bytes through untouched while feeding the observer.
type observingReader struct {
	src        io.Reader
	obs        *usageObserver
	firstToken time.Time
}

func (r *observingReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		if r.firstToken.IsZero() {
			r.firstToken = time.Now().UTC()
			r.obs.usage.FirstTokenAt = r.firstToken
		}
		r.obs.feed(p[:n])
	}
	return n, err
}

// feed consumes a chunk of the response body.
func (o *usageObserver) feed(chunk []byte) {
	if o.sse {
		o.feedSSE(chunk)
		return
	}
	if len(o.body)+len(chunk) > maxObservedBody {
		o.body = nil
		o.dropped = true
		return
	}
	o.body = append(o.body, chunk...)
}

// feedSSE accumulates the chunk and parses every complete line it produced. A
// frame boundary can fall anywhere in the stream, so partial lines are kept
// until the terminating newline arrives.
func (o *usageObserver) feedSSE(chunk []byte) {
	if o.dropped {
		return
	}
	if len(o.frame)+len(chunk) > maxObservedFrame {
		// Not a usage frame. Drop what we have and stop observing this response
		// rather than re-allocating on every subsequent chunk.
		o.frame = nil
		o.dropped = true
		return
	}
	o.frame = append(o.frame, chunk...)

	for {
		idx := bytes.IndexByte(o.frame, '\n')
		if idx < 0 {
			return
		}
		line := o.frame[:idx]
		o.frame = o.frame[idx+1:]
		o.parseSSELine(line)
	}
}

// parseSSELine handles one complete SSE line.
func (o *usageObserver) parseSSELine(line []byte) {
	line = bytes.TrimSuffix(line, []byte("\r"))
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	payload := bytes.TrimSpace(line[len("data:"):])
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
		return
	}
	o.parseEvent(payload)
}

// finish flushes the buffered state and reports the observed usage. It returns
// nil when the upstream sent no usage, so the caller can fall back.
func (o *usageObserver) finish() *StreamUsage {
	if !o.sse && !o.dropped && len(o.body) > 0 {
		o.parseEvent(o.body)
	}
	if !o.usage.Reported {
		return nil
	}
	usage := o.usage
	return &usage
}

// wireUsage spans the token fields of all three supported providers. The field
// names do not collide, so one tolerant shape serves Anthropic, OpenAI and
// Gemini in both their streaming and non-streaming forms.
type wireUsage struct {
	// Anthropic.
	InputTokens      *int64 `json:"input_tokens"`
	OutputTokens     *int64 `json:"output_tokens"`
	CacheReadTokens  *int64 `json:"cache_read_input_tokens"`
	CacheWriteTokens *int64 `json:"cache_creation_input_tokens"`
	// OpenAI.
	PromptTokens     *int64 `json:"prompt_tokens"`
	CompletionTokens *int64 `json:"completion_tokens"`
	// OpenAI nests cached tokens one level down.
	PromptTokensDetails *struct {
		CachedTokens *int64 `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	// Gemini.
	PromptTokenCount     *int64 `json:"promptTokenCount"`
	CandidatesTokenCount *int64 `json:"candidatesTokenCount"`
	CachedContentTokens  *int64 `json:"cachedContentTokenCount"`
}

// wireEvent is the envelope each provider places its usage inside.
type wireEvent struct {
	// Anthropic puts the opening usage under message.usage.
	Message *struct {
		Usage *wireUsage `json:"usage"`
	} `json:"message"`
	// OpenAI and Anthropic non-streaming responses use a top-level usage.
	Usage *wireUsage `json:"usage"`
	// Gemini uses usageMetadata.
	UsageMetadata *wireUsage `json:"usageMetadata"`
}

// parseEvent extracts usage from one provider payload, ignoring anything it
// cannot understand. A malformed frame must never break forwarding.
func (o *usageObserver) parseEvent(payload []byte) {
	var event wireEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return
	}

	switch {
	case event.Message != nil && event.Message.Usage != nil:
		o.merge(event.Message.Usage, "anthropic")
	case event.Usage != nil:
		o.merge(event.Usage, "")
	case event.UsageMetadata != nil:
		o.merge(event.UsageMetadata, "gemini")
	}
}

// merge records the latest counts. Providers report usage cumulatively, so the
// last observed value of each field is the final one.
func (o *usageObserver) merge(u *wireUsage, provider string) {
	seen := false

	if v := firstInt64(u.InputTokens, u.PromptTokens, u.PromptTokenCount); v != nil {
		o.usage.InputTokens = int(*v)
		o.setProvider(firstNonEmpty(provider, providerForInput(u)))
		seen = true
	}
	if v := firstInt64(u.OutputTokens, u.CompletionTokens, u.CandidatesTokenCount); v != nil {
		o.usage.OutputTokens = int(*v)
		o.setProvider(firstNonEmpty(provider, providerForOutput(u)))
		seen = true
	}
	var openAICached *int64
	if u.PromptTokensDetails != nil {
		openAICached = u.PromptTokensDetails.CachedTokens
	}
	if v := firstInt64(u.CacheReadTokens, openAICached, u.CachedContentTokens); v != nil {
		o.usage.CacheReadTokens = int(*v)
		seen = true
	}
	if u.CacheWriteTokens != nil {
		o.usage.CacheWriteTokens = int(*u.CacheWriteTokens)
		seen = true
	}

	if seen {
		o.usage.Reported = true
	}
}

// setProvider records the provider shape once; a response comes from one
// provider, so the first detection wins.
func (o *usageObserver) setProvider(name string) {
	if name == "" || o.usage.Provider != "" {
		return
	}
	o.usage.Provider = name
}

func providerForInput(u *wireUsage) string {
	switch {
	case u.InputTokens != nil:
		return "anthropic"
	case u.PromptTokens != nil:
		return "openai"
	case u.PromptTokenCount != nil:
		return "gemini"
	}
	return ""
}

func providerForOutput(u *wireUsage) string {
	switch {
	case u.OutputTokens != nil:
		return "anthropic"
	case u.CompletionTokens != nil:
		return "openai"
	case u.CandidatesTokenCount != nil:
		return "gemini"
	}
	return ""
}

func firstInt64(values ...*int64) *int64 {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
