package gateway

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
)

func testHandler(*testing.T) http.Handler {
	mw := httpx.NewMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), httpx.DefaultOptions())
	return NewHandler(nil, mw, health.NewRegistry(), Deps{})
}

func TestHealthEndpoints(t *testing.T) {
	h := testHandler(t)

	for _, path := range []string{"/livez", "/readyz", "/healthz"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))

		if rr.Code != http.StatusOK {
			t.Errorf("%s status = %d, want 200", path, rr.Code)
		}
		var body health.Status
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s body is not valid JSON: %v", path, err)
		}
		if body.Status != "ok" {
			t.Errorf("%s status field = %q, want ok", path, body.Status)
		}
		if rr.Header().Get("X-Request-Id") == "" {
			t.Errorf("%s missing X-Request-Id header", path)
		}
		if rr.Header().Get("X-Correlation-Id") == "" {
			t.Errorf("%s missing X-Correlation-Id header", path)
		}
	}
}

func TestUnknownRoute404(t *testing.T) {
	h := testHandler(t)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/definitely-not-a-route", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}
