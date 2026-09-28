// Package gatewayv2 end-to-end proxying tests: an allowed request must reach the
// upstream provider, and request inspection must never consume the body that is
// forwarded. These use a real StreamProxyImpl and a mock upstream, not stubs.
package gatewayv2

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// capture records what the mock upstream actually received.
type capture struct {
	mu     sync.Mutex
	body   string
	path   string
	header http.Header
}

func (c *capture) set(body, path string, h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.body, c.path, c.header = body, path, h
}

func (c *capture) get() (string, string, http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.body, c.path, c.header
}

// newMockUpstream starts a provider stand-in that echoes a fixed completion and
// records the request it was given.
func newMockUpstream(t *testing.T, cap *capture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		cap.set(string(body), r.URL.Path, r.Header.Clone())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"id":"msg_1","type":"message","content":[{"type":"text","text":"upstream-ok"}],"usage":{"input_tokens":11,"output_tokens":7}}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newGatewayWithRealProxy wires a gateway to a real stream proxy so Forward
// performs a genuine upstream round trip. The trailing slash mirrors the
// normalization ProviderRouterImpl applies, which buildUpstreamRequest relies on.
func newGatewayWithRealProxy(t *testing.T, upstreamURL string) (*Gateway, *gatewayStubs) {
	t.Helper()
	g, stubs := newGatewayForPipeline()
	g.streamProxy = NewStreamProxy(nil, discardLogger())
	stubs.router.url = strings.TrimRight(upstreamURL, "/") + "/"
	stubs.router.headers = http.Header{"X-Upstream-Key": {"k"}}
	return g, stubs
}

// TestServeHTTP_AllowedReachesMockUpstream is the core regression test for the
// bug where an allowed request returned a bare 200 without ever calling the
// provider. It asserts the upstream received the request body and that the
// upstream response body reached the client.
func TestServeHTTP_AllowedReachesMockUpstream(t *testing.T) {
	var cap capture
	upstream := newMockUpstream(t, &cap)
	g, _ := newGatewayWithRealProxy(t, upstream.URL)

	payload := `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hello"}]}`
	req := pipelineRequest("key-1", payload)
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	gotBody, gotPath, gotHeader := cap.get()

	requireTrue(t, gotBody == payload, "upstream received body %q, want %q", gotBody, payload)
	requireTrue(t, gotPath == "/v1/messages", "upstream path %q, want /v1/messages", gotPath)
	requireTrue(t, gotHeader.Get("X-Upstream-Key") == "k", "routed upstream headers were not applied")
	requireTrue(t, rec.Code == http.StatusOK, "want 200 from upstream, got %d", rec.Code)
	requireTrue(t, strings.Contains(rec.Body.String(), "upstream-ok"), "upstream response body never reached the client: %q", rec.Body.String())
	requireTrue(t, rec.Header().Get("x-request-id") != "", "x-request-id not set")
}

// TestServeHTTP_DeniedNeverReachesUpstream guards the other half of the contract:
// a policy denial must not spend an upstream call.
func TestServeHTTP_DeniedNeverReachesUpstream(t *testing.T) {
	var cap capture
	upstream := newMockUpstream(t, &cap)
	g, stubs := newGatewayWithRealProxy(t, upstream.URL)
	stubs.auth.err = errStub

	req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	rec := httptest.NewRecorder()

	g.ServeHTTP(rec, req)

	gotBody, _, _ := cap.get()
	requireTrue(t, gotBody == "", "denied request still reached the upstream provider")
	requireTrue(t, rec.Code == http.StatusUnauthorized, "want 401, got %d", rec.Code)
}

// TestPipelineBodyPreservedAfterInspection drives the real inspector and then
// re-reads the request body, proving the forwarded payload is byte-identical to
// what the client sent. With the previous body-consuming inspector this read
// returned empty and the upstream would have received EOF.
func TestPipelineBodyPreservedAfterInspection(t *testing.T) {
	cases := []struct {
		name    string
		payload string
	}{
		{"small", `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`},
		{"streaming", `{"model":"claude-sonnet-4-5","stream":true,"messages":[{"role":"user","content":"hi"}]}`},
		{"tool request", `{"model":"claude-sonnet-4-5","tools":[{"name":"get_weather","input_schema":{}}],"messages":[{"role":"user","content":"weather?"}]}`},
		{"tool result", `{"model":"claude-sonnet-4-5","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"get_weather"}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"sunny"}]}]}`},
		{"multimodal", `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":[{"type":"text","text":"look"},{"type":"image","source":{"type":"base64","media_type":"image/png","data":"iVBORw0KGgo="}}]}]}`},
		{"thinking", `{"model":"claude-sonnet-4-5","thinking":{"type":"enabled","budget_tokens":1024},"messages":[{"role":"user","content":"think"}]}`},
		{"malformed json", `{"model":`},
		{"not json at all", `plain text payload`},
		{"provider-native unknown fields", `{"model":"claude-sonnet-4-5","anthropic_only":{"beta":true},"future_field":[1,2,3]}`},
		{"unicode and escapes", `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"héllo \u00e9 \n <script>"}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := newGatewayForPipeline()
			g.requestInspector = NewRequestInspector()
			req := pipelineRequest("key-1", tc.payload)

			g.executePipeline(context.Background(), req, "req-1")

			got, err := io.ReadAll(req.Body)
			requireNoErr(t, err, "read body after inspection")
			requireTrue(t, string(got) == tc.payload, "body was altered by inspection:\n got %q\nwant %q", string(got), tc.payload)
		})
	}
}

// TestPipelineInspectionExtractsHintsFromIntactBody proves hints are still
// extracted while the body is left intact for the forward.
func TestPipelineInspectionExtractsHintsFromIntactBody(t *testing.T) {
	g, _ := newGatewayForPipeline()
	g.requestInspector = NewRequestInspector()
	payload := `{"model":"claude-sonnet-4-5","stream":true,"max_tokens":64,"messages":[{"role":"user","content":"hello world"}]}`
	req := pipelineRequest("key-1", payload)

	result := g.executePipeline(context.Background(), req, "req-1")

	requireTrue(t, result.Allowed, "request should be allowed, got %+v", result)
	requireTrue(t, result.Context.InspectorHints.Model == "claude-sonnet-4-5", "model hint not extracted: %q", result.Context.InspectorHints.Model)
	requireTrue(t, result.Context.InspectorHints.Stream, "stream hint not extracted")
	requireTrue(t, result.Context.InspectorHints.EstimatedInputTokens > 0, "input token hint not estimated")

	got, _ := io.ReadAll(req.Body)
	requireTrue(t, string(got) == payload, "body changed during hint extraction")
}

// TestPipelineRejectsOversizedBody verifies the buffering bound is enforced
// before an oversized payload is forwarded or fully allocated.
func TestPipelineRejectsOversizedBody(t *testing.T) {
	g, _ := newGatewayForPipeline()
	g.requestInspector = NewRequestInspector()
	oversized := `{"model":"claude-sonnet-4-5","pad":"` + strings.Repeat("a", maxRequestBodyBytes) + `"}`
	req := pipelineRequest("key-1", oversized)

	result := g.executePipeline(context.Background(), req, "req-1")

	requireTrue(t, !result.Allowed, "oversized request was allowed")
	requireTrue(t, result.StatusCode == http.StatusRequestEntityTooLarge, "want 413, got %d", result.StatusCode)
	requireTrue(t, result.ErrorType == "request_too_large", "want request_too_large, got %q", result.ErrorType)
}

// TestPipelineAcceptsBodyAtLimit verifies the bound does not reject a request
// that fits exactly.
func TestPipelineAcceptsBodyAtLimit(t *testing.T) {
	g, _ := newGatewayForPipeline()
	g.requestInspector = NewRequestInspector()
	prefix := `{"model":"claude-sonnet-4-5","pad":"`
	suffix := `"}`
	payload := prefix + strings.Repeat("a", maxRequestBodyBytes-len(prefix)-len(suffix)) + suffix
	if len(payload) != maxRequestBodyBytes {
		t.Fatalf("test payload is %d bytes, want exactly %d", len(payload), maxRequestBodyBytes)
	}
	req := pipelineRequest("key-1", payload)

	result := g.executePipeline(context.Background(), req, "req-1")

	requireTrue(t, result.Allowed, "request at the size limit was rejected: %+v", result)
	got, _ := io.ReadAll(req.Body)
	requireTrue(t, len(got) == maxRequestBodyBytes, "body at limit was truncated to %d bytes", len(got))
}

// TestPipelinePreservesContentLength pins the forwarding length so a request
// that arrived without one is not silently re-chunked upstream.
func TestPipelinePreservesContentLength(t *testing.T) {
	g, _ := newGatewayForPipeline()
	g.requestInspector = NewRequestInspector()
	payload := `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`
	req := pipelineRequest("key-1", payload)
	req.ContentLength = -1 // simulate a streamed/chunked upload

	g.executePipeline(context.Background(), req, "req-1")

	requireTrue(t, req.ContentLength == int64(len(payload)), "want ContentLength %d, got %d", len(payload), req.ContentLength)
}

// TestStreamProxyForwardDoesNotRequireCloseNotifier pins the fix for the
// unchecked http.CloseNotifier assertion, which panicked on any writer that is
// not a bare HTTP/1 ResponseWriter.
func TestStreamProxyForwardDoesNotRequireCloseNotifier(t *testing.T) {
	var cap capture
	upstream := newMockUpstream(t, &cap)
	proxy := NewStreamProxy(nil, discardLogger())

	req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
	rec := httptest.NewRecorder()

	if _, err := proxy.Forward(context.Background(), rec, req, upstream.URL+"/", nil); err != nil {
		t.Fatalf("Forward returned an error on a non-CloseNotifier writer: %v", err)
	}

	requireTrue(t, rec.Code == http.StatusOK, "want 200, got %d", rec.Code)
	requireTrue(t, strings.Contains(rec.Body.String(), "upstream-ok"), "upstream body missing from client response")
}

// TestStreamProxyReleasesStreamOnCompletion verifies the active-stream counter
// is decremented so a completed forward does not leak a stream slot.
func TestStreamProxyReleasesStreamOnCompletion(t *testing.T) {
	var cap capture
	upstream := newMockUpstream(t, &cap)
	proxy := NewStreamProxy(nil, discardLogger())

	for i := 0; i < 3; i++ {
		req := pipelineRequest("key-1", `{"model":"claude-sonnet-4-5"}`)
		rec := httptest.NewRecorder()
		if _, err := proxy.Forward(context.Background(), rec, req, upstream.URL+"/", nil); err != nil {
			t.Fatalf("forward %d failed: %v", i, err)
		}
	}

	if got := proxy.ActiveStreamCount(); got != 0 {
		t.Fatalf("active streams leaked: want 0, got %d", got)
	}
}
