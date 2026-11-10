package runtimestats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_RequiresToken(t *testing.T) {
	h := Handler("s3cret", nil)

	for _, auth := range []string{"", "Bearer wrong"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/internal/stats", nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("auth %q: got %d, want 404", auth, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/stats", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("right token: got %d, want 200", rec.Code)
	}
	var s Stats
	if err := json.NewDecoder(rec.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if s.Goroutines == 0 || s.HeapBytes == 0 || s.RSSBytes == 0 {
		t.Fatalf("expected live process figures, got %+v", s)
	}
}

func TestHandler_EmptyTokenDisables(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/stats", nil)
	req.Header.Set("Authorization", "Bearer ")
	Handler("", nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("empty token: got %d, want 404", rec.Code)
	}
}
