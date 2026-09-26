package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmptyRegistryIsHealthy(t *testing.T) {
	reg := NewRegistry()
	s := reg.Status(context.Background())
	if s.Status != "ok" {
		t.Errorf("Status = %q, want ok", s.Status)
	}
}

func TestPassingChecksStayOk(t *testing.T) {
	reg := NewRegistry()
	reg.Register("postgres", func(context.Context) error { return nil })
	reg.Register("redis", func(context.Context) error { return nil })

	s := reg.Status(context.Background())
	if s.Status != "ok" {
		t.Errorf("Status = %q, want ok (checks: %v)", s.Status, s.Checks)
	}
	if s.Checks["postgres"] != "ok" || s.Checks["redis"] != "ok" {
		t.Errorf("checks = %v, want all ok", s.Checks)
	}
}

func TestFailingCheckDegrades(t *testing.T) {
	reg := NewRegistry()
	reg.Register("postgres", func(context.Context) error { return nil })
	reg.Register("redis", func(context.Context) error { return errors.New("connection refused") })

	s := reg.Status(context.Background())
	if s.Status != "degraded" {
		t.Errorf("Status = %q, want degraded", s.Status)
	}
	if s.Checks["redis"] != "connection refused" {
		t.Errorf("redis check = %q, want failure message", s.Checks["redis"])
	}
}

func TestHandlerStatusCodes(t *testing.T) {
	okReg := NewRegistry()
	rr := httptest.NewRecorder()
	Handler(okReg).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("healthy status = %d, want 200", rr.Code)
	}

	badReg := NewRegistry()
	badReg.Register("db", func(context.Context) error { return errors.New("down") })
	rr = httptest.NewRecorder()
	Handler(badReg).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("degraded status = %d, want 503", rr.Code)
	}

	var body Status
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body.Status != "degraded" {
		t.Errorf("body status = %q, want degraded", body.Status)
	}
	if body.Checks["db"] == "" {
		t.Error("body checks missing db entry")
	}
}

func TestRegisterIgnoresNil(t *testing.T) {
	reg := NewRegistry()
	reg.Register("nil-check", nil)
	if got := len(reg.checks); got != 0 {
		t.Errorf("checks count = %d, want 0", got)
	}
}
