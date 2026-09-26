package httpx

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	platformerrors "ai-governance-saas/backend/internal/platform/errors"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestMiddleware(opts Options) *Middleware {
	return NewMiddleware(discardLogger(), opts)
}

func TestRequestIDGeneratedAndEchoed(t *testing.T) {
	mw := newTestMiddleware(DefaultOptions())
	var gotCtx string
	h := mw.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = RequestIDFrom(r.Context())
		if gotCtx == "" {
			t.Error("request ID missing from context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	echoed := rr.Header().Get(defaultRequestIDHeader)
	if echoed == "" || echoed != gotCtx {
		t.Errorf("header %q = %q, ctx = %q; must match and be non-empty", defaultRequestIDHeader, echoed, gotCtx)
	}
}

func TestInboundRequestIDHonored(t *testing.T) {
	mw := newTestMiddleware(DefaultOptions())
	var gotCtx string
	h := mw.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtx = RequestIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(defaultRequestIDHeader, "upstream-id-123")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if gotCtx != "upstream-id-123" {
		t.Errorf("ctx request ID = %q, want upstream-id-123", gotCtx)
	}
}

func TestCorrelationIDDistinct(t *testing.T) {
	mw := newTestMiddleware(DefaultOptions())
	var reqID, corrID string
	h := mw.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID = RequestIDFrom(r.Context())
		corrID = CorrelationIDFrom(r.Context())
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	if corrID == "" || corrID == reqID {
		t.Errorf("correlation ID %q must be set and differ from request ID %q", corrID, reqID)
	}
}

func TestRecoveryReturnsStructured500(t *testing.T) {
	mw := newTestMiddleware(DefaultOptions())
	h := mw.Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("kaboom")
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/boom", nil))

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rr.Code)
	}
	var body ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not the JSON error envelope: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", body.Error.Code)
	}
	if rr.Header().Get(defaultRequestIDHeader) != body.Error.RequestID {
		t.Error("request_id must echo the X-Request-Id header")
	}
}

func TestTimeoutReports504(t *testing.T) {
	opts := DefaultOptions()
	opts.DefaultRequestTimeout = 50 * time.Millisecond
	mw := newTestMiddleware(opts)

	h := mw.Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/slow", nil))

	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", rr.Code)
	}
	var body ErrorBody
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body.Error.Code != "request_timeout" {
		t.Errorf("code = %q, want request_timeout", body.Error.Code)
	}
}

func TestTimeoutDisabledWhenZero(t *testing.T) {
	h := newTestMiddleware(DefaultOptions()).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); ok {
			t.Error("context deadline set despite zero timeout")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
}

func TestWriteErrorEnvelope(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusBadRequest, "req-1", stderrors.New("plain failure"))

	var body ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("envelope parse failed: %v", err)
	}
	if body.Error.Code != "internal" {
		t.Errorf("plain error code = %q, want internal", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Error("message must not be empty")
	}
	if body.Error.RequestID != "req-1" {
		t.Errorf("request_id = %q, want req-1", body.Error.RequestID)
	}
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestWriteErrorWithPlatformError(t *testing.T) {
	rr := httptest.NewRecorder()
	WriteError(rr, http.StatusNotFound, "req-2", platformerrors.New(
		platformerrors.KindNotFound, "org_not_found", "organization not found",
	))

	var body ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("envelope parse failed: %v", err)
	}
	if body.Error.Code != "org_not_found" {
		t.Errorf("code = %q, want org_not_found", body.Error.Code)
	}
	if body.Error.Message != "organization not found" {
		t.Errorf("message = %q, want organization not found", body.Error.Message)
	}
}

func TestStatusFor(t *testing.T) {
	cases := []struct {
		kind platformerrors.Kind
		want int
	}{
		{platformerrors.KindInvalidArgument, http.StatusBadRequest},
		{platformerrors.KindUnauthenticated, http.StatusUnauthorized},
		{platformerrors.KindPermissionDenied, http.StatusForbidden},
		{platformerrors.KindNotFound, http.StatusNotFound},
		{platformerrors.KindConflict, http.StatusConflict},
		{platformerrors.KindUnavailable, http.StatusServiceUnavailable},
		{platformerrors.KindInternal, http.StatusInternalServerError},
		{platformerrors.Kind(999), http.StatusInternalServerError},
	}
	for _, tt := range cases {
		if got := StatusFor(tt.kind); got != tt.want {
			t.Errorf("StatusFor(%v) = %d, want %d", tt.kind, got, tt.want)
		}
	}
}

func TestSecurityHeadersApplied(t *testing.T) {
	h := newTestMiddleware(DefaultOptions()).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/x", nil))

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	}
	for k, v := range want {
		if got := rr.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
}

func TestCORSAllowedOriginEchoed(t *testing.T) {
	opts := DefaultOptions()
	opts.CORSAllowedOrigins = []string{"https://app.example.test"}
	h := newTestMiddleware(opts).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.test")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.test" {
		t.Errorf("ACAO = %q, want echoed origin", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("credentials = %q, want true", got)
	}
	if !slices.Contains(rr.Header().Values("Vary"), "Origin") {
		t.Error("Vary must include Origin")
	}
}

func TestCORSDeniesDisallowedOrigin(t *testing.T) {
	opts := DefaultOptions()
	opts.CORSAllowedOrigins = []string{"https://app.example.test"}
	h := newTestMiddleware(opts).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.test")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty for disallowed origin", got)
	}
}

func TestCORSPreflightShortCircuits(t *testing.T) {
	opts := DefaultOptions()
	opts.CORSAllowedOrigins = []string{"https://app.example.test"}
	called := false
	h := newTestMiddleware(opts).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/v1/projects", nil)
	req.Header.Set("Origin", "https://app.example.test")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rr.Code)
	}
	if called {
		t.Error("preflight must not reach the handler")
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Errorf("allowed methods = %q, want POST included", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "X-CSRF-Token") {
		t.Errorf("allowed headers = %q, want X-CSRF-Token included", got)
	}
}

func TestCORSDisabledWhenNoOrigins(t *testing.T) {
	h := newTestMiddleware(DefaultOptions()).Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.test")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("ACAO = %q, want empty when CORS disabled", got)
	}
}
