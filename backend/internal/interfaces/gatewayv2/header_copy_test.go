package gatewayv2

import (
	"net/http"
	"testing"
)

func TestCopyForwardHeaders_DropsCookies(t *testing.T) {
	src := http.Header{}
	src.Set("Cookie", "session=abc; session_refresh=def")
	src.Set("Anthropic-Version", "2023-06-01")

	dst := http.Header{}
	copyForwardHeaders(dst, src)

	if dst.Get("Cookie") != "" {
		t.Fatalf("Cookie forwarded upstream: %q", dst.Get("Cookie"))
	}
	if dst.Get("Anthropic-Version") == "" {
		t.Fatal("ordinary header was dropped")
	}
}

func TestCopyResponseHeaders_DropsSetCookie(t *testing.T) {
	src := http.Header{}
	src.Set("Set-Cookie", "session=attacker; Path=/")
	src.Set("Content-Type", "text/event-stream")

	dst := http.Header{}
	copyResponseHeaders(dst, src)

	if dst.Get("Set-Cookie") != "" {
		t.Fatalf("upstream Set-Cookie passed to client: %q", dst.Get("Set-Cookie"))
	}
	if dst.Get("Content-Type") == "" {
		t.Fatal("ordinary header was dropped")
	}
}
