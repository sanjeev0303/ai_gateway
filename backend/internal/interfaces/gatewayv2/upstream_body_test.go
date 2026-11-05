package gatewayv2

import (
	"strings"
	"testing"
)

func TestPrepareUpstreamBody(t *testing.T) {
	plain := []byte(`{"model":"gemini-3.5-flash","messages":[{"role":"user","content":"hi"}]}`)
	if got := prepareUpstreamBody(plain, nil); got != nil {
		t.Fatalf("a body with nothing to change was rewritten: %s", got)
	}

	withIDs := []byte(`{"model":"m","conversation_id":"c1","project_id":"p1","seed":12345678901234567,"messages":[{"role":"user","content":"hi"}]}`)
	got := string(prepareUpstreamBody(withIDs, nil))
	if strings.Contains(got, "conversation_id") || strings.Contains(got, "project_id") {
		t.Fatalf("gateway fields were forwarded: %s", got)
	}
	if !strings.Contains(got, "12345678901234567") {
		t.Fatalf("large integer lost precision: %s", got)
	}

	mask := func(s string) string { return strings.ReplaceAll(s, "secret", "[REDACTED]") }
	parts := []byte(`{"model":"secret-model","messages":[{"role":"user","content":[{"type":"text","text":"my secret <plan>"}]}]}`)
	got = string(prepareUpstreamBody(parts, mask))
	if !strings.Contains(got, `"text":"my [REDACTED] <plan>"`) {
		t.Fatalf("content part not redacted: %s", got)
	}
	if !strings.Contains(got, `"model":"secret-model"`) {
		t.Fatalf("model id must never be redacted: %s", got)
	}
}
