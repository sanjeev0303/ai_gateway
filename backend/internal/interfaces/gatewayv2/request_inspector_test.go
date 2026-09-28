// Package gatewayv2 contract tests for request inspection: extraction rules,
// token estimation, and the request body size bound.
package gatewayv2

import (
	"context"
	"strings"
	"testing"
)

// errReader fails every read so Inspect surfaces the read error.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errStub }

func TestInspect_FullRequest(t *testing.T) {
	body := `{
		"model": "claude-sonnet-4-5",
		"messages": [{"role": "user", "content": "abcdefgh"}],
		"stream": true,
		"tools": [{"name": "web_search"}, {"name": "calculator"}],
		"conversation_id": "11111111-1111-4111-8111-111111111111",
		"project_id": "22222222-2222-4222-8222-222222222222",
		"routing_tags": {"env": "prod"},
		"max_tokens": 2048
	}`

	hints, err := NewRequestInspector().Inspect(context.Background(), strings.NewReader(body))
	requireNoErr(t, err, "Inspect")
	requireTrue(t, hints.Model == "claude-sonnet-4-5", "model not extracted")
	requireTrue(t, hints.RoutingTags["model"] == "claude-sonnet-4-5", "model tag missing")
	requireTrue(t, hints.RoutingTags["stream"] == "true", "stream tag missing")
	requireTrue(t, hints.RoutingTags["project_id"] == "22222222-2222-4222-8222-222222222222", "project tag missing")
	requireTrue(t, hints.RoutingTags["env"] == "prod", "custom tag missing")
	requireTrue(t, hints.EstimatedInputTokens == 2, "input token estimate wrong: %d", hints.EstimatedInputTokens)
	requireTrue(t, hints.EstimatedOutputTokens == 2048, "output token estimate wrong: %d", hints.EstimatedOutputTokens)
	requireTrue(t, hints.Stream, "stream flag not extracted")
	requireTrue(t, len(hints.Tools) == 2 && hints.Tools[0] == "web_search" && hints.Tools[1] == "calculator", "tools not extracted")
	requireTrue(t, hints.ConversationID != nil && *hints.ConversationID == mustUUIDs[0], "conversation id not parsed")
	requireTrue(t, hints.ProjectID != nil && *hints.ProjectID == mustUUIDs[1], "project id not parsed")
	requireTrue(t, len(hints.Warnings) == 0, "unexpected warnings")
}

func TestInspect_MissingModelWarns(t *testing.T) {
	hints, err := NewRequestInspector().Inspect(context.Background(), strings.NewReader(`{"messages":[{"content":"abcdefgh"}]}`))
	requireNoErr(t, err, "Inspect")
	requireTrue(t, hints.Model == "", "model should be empty")
	requireTrue(t, len(hints.Warnings) == 1 && hints.Warnings[0] == "model field is missing", "missing model warning not recorded")
	requireTrue(t, hints.EstimatedInputTokens == 2, "tokens still estimated without model")
}

func TestInspect_NoMessagesZeroTokens(t *testing.T) {
	hints, err := NewRequestInspector().Inspect(context.Background(), strings.NewReader(`{"model":"claude-sonnet-4-5"}`))
	requireNoErr(t, err, "Inspect")
	requireTrue(t, hints.EstimatedInputTokens == 0 && hints.EstimatedOutputTokens == 0, "tokens must be zero without messages")
}

func TestInspect_MalformedBodyReturnsEmpty(t *testing.T) {
	hints, err := NewRequestInspector().Inspect(context.Background(), strings.NewReader("this is not json"))
	requireNoErr(t, err, "malformed body must not error")
	requireTrue(t, hints.Model == "", "expected empty hints")
	requireTrue(t, len(hints.RoutingTags) == 0, "expected no routing tags")
}

func TestInspect_ReadError(t *testing.T) {
	_, err := NewRequestInspector().Inspect(context.Background(), errReader{})
	requireErrIs(t, err, errStub)
}

func TestInspect_RejectsOversizedBody(t *testing.T) {
	overLimit := strings.NewReader(strings.Repeat("a", maxRequestBodyBytes+1))
	_, err := NewRequestInspector().Inspect(context.Background(), overLimit)
	requireErrIs(t, err, ErrRequestTooLarge)

	atLimit := strings.NewReader(strings.Repeat("a", maxRequestBodyBytes))
	_, err = NewRequestInspector().Inspect(context.Background(), atLimit)
	requireNoErr(t, err, "body exactly at the limit must be accepted")
}

func TestInspect_InvalidUUIDsIgnored(t *testing.T) {
	hints, err := NewRequestInspector().Inspect(context.Background(), strings.NewReader(`{"model":"x","conversation_id":"nope","project_id":"nope"}`))
	requireNoErr(t, err, "Inspect")
	requireTrue(t, hints.ConversationID == nil, "invalid conversation id must be ignored")
	requireTrue(t, hints.ProjectID == nil, "invalid project id must be ignored")
}

func TestEstimateInputTokens(t *testing.T) {
	single := []any{map[string]any{"content": "abcdefgh"}}
	requireTrue(t, estimateInputTokens(single) == 2, "string content estimate wrong")

	// Multimodal: text blocks sum (4/4 + 8/4 = 3).
	multi := []any{map[string]any{"content": []any{map[string]any{"text": "abcd"}, map[string]any{"text": "abcdefgh"}}}}
	requireTrue(t, estimateInputTokens(multi) == 3, "multimodal estimate wrong")

	emptyMsg := []any{map[string]any{"content": []any{map[string]any{"type": "image"}}}}
	requireTrue(t, estimateInputTokens(emptyMsg) == 256, "default minimum not applied")
}

func TestEstimateOutputTokens(t *testing.T) {
	// JSON-decoded numbers arrive as float64.
	requireTrue(t, estimateOutputTokens(map[string]any{"max_tokens": float64(2048)}) == 2048, "max_tokens not honored")
	requireTrue(t, estimateOutputTokens(map[string]any{}) == 1024, "default output estimate wrong")
}
