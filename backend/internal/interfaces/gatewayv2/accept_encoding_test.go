package gatewayv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildUpstreamRequestDropsAcceptEncoding(t *testing.T) {
	s := &StreamProxyImpl{}
	in := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{}`))
	in.Header.Set("Accept-Encoding", "gzip, deflate, br")

	out, err := s.buildUpstreamRequest(context.Background(), in, "https://upstream.example/v1/", http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Header.Get("Accept-Encoding"); got != "" {
		t.Fatalf("upstream request must not ask for compression, got %q", got)
	}
}
