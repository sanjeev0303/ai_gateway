package gatewayv2

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPromptTurn(t *testing.T) {
	cases := []struct {
		name, body, first, latest string
	}{
		{"openai multi-turn", `{"messages":[{"role":"system","content":"be brief"},{"role":"user","content":"hi"},{"role":"assistant","content":"hello"},{"role":"user","content":"write a haiku"}]}`, "hi", "write a haiku"},
		{"openai content parts", `{"messages":[{"role":"user","content":[{"type":"text","text":"describe"},{"type":"image_url","image_url":{"url":"x"}}]}]}`, "describe", "describe"},
		{"anthropic", `{"model":"claude","messages":[{"role":"user","content":"ping"}]}`, "ping", "ping"},
		{"gemini", `{"contents":[{"role":"user","parts":[{"text":"a"},{"text":"b"}]}]}`, "ab", "ab"},
		{"no user message", `{"messages":[{"role":"system","content":"x"}]}`, "", ""},
		{"not json", `nope`, "", ""},
	}
	for _, tc := range cases {
		first, latest := promptTurn([]byte(tc.body))
		if first != tc.first || latest != tc.latest {
			t.Errorf("%s: got (%q, %q), want (%q, %q)", tc.name, first, latest, tc.first, tc.latest)
		}
	}
}

func TestConversationIDFor(t *testing.T) {
	cred := uuid.New()
	r := httptest.NewRequest("POST", "/v1/chat/completions", nil)

	a := conversationIDFor(r, InspectorHints{}, cred, "plan a trip")
	b := conversationIDFor(r, InspectorHints{}, cred, "plan a trip")
	if a != b {
		t.Fatal("same credential and opening message must map to the same conversation")
	}
	if conversationIDFor(r, InspectorHints{}, uuid.New(), "plan a trip") == a {
		t.Fatal("another credential must not share the conversation")
	}

	explicit := uuid.New()
	r.Header.Set("x-conversation-id", explicit.String())
	if got := conversationIDFor(r, InspectorHints{}, cred, "plan a trip"); got != explicit {
		t.Fatalf("header id ignored: got %s", got)
	}
	body := uuid.New()
	if got := conversationIDFor(r, InspectorHints{ConversationID: &body}, cred, "x"); got != body {
		t.Fatalf("body conversation_id must win: got %s", got)
	}
}

func TestObserverCollectsReplyText(t *testing.T) {
	cases := []struct {
		name, contentType, body, want string
	}{
		{"openai json", "application/json", `{"choices":[{"message":{"content":"Hello there"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`, "Hello there"},
		{"openai sse", "text/event-stream", "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\ndata: [DONE]\n\n", "Hello"},
		{"anthropic json", "application/json", `{"content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":1,"output_tokens":1}}`, "Hi"},
		{"anthropic sse", "text/event-stream", "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Yo\"}}\n\n", "Yo"},
		{"gemini sse", "text/event-stream", "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Ok\"}]}}]}\n\n", "Ok"},
	}
	for _, tc := range cases {
		obs := newUsageObserver(tc.contentType)
		obs.usage.StatusCode = 200
		obs.feed([]byte(tc.body))
		if got := obs.finish(); got == nil || got.Text != tc.want {
			t.Errorf("%s: got %+v, want text %q", tc.name, got, tc.want)
		}
	}

	big := newUsageObserver("text/event-stream")
	big.usage.StatusCode = 200
	chunk := "data: {\"choices\":[{\"delta\":{\"content\":\"" + strings.Repeat("x", 1000) + "\"}}]}\n\n"
	for i := 0; i < 100; i++ {
		big.feed([]byte(chunk))
	}
	if got := big.finish(); len(got.Text) != maxObservedText {
		t.Fatalf("reply text must be capped at %d bytes, got %d", maxObservedText, len(got.Text))
	}
}
