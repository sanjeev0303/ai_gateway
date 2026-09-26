// Package health provides liveness/readiness probes backed by a registry of
// dependency checks. Each process registers the checks for its own
// dependencies; /healthz reports them all, /livez proves the process is up.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
)

// Check reports whether a dependency is healthy; a nil error means healthy.
type Check func(ctx context.Context) error

// Registry holds named dependency checks.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
}

// NewRegistry returns an empty registry. An empty registry is healthy: no
// dependencies means nothing to check.
func NewRegistry() *Registry {
	return &Registry{checks: make(map[string]Check)}
}

// Register adds (or replaces) a named check. A nil check is ignored.
func (r *Registry) Register(name string, c Check) {
	if c == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = c
}

// Status reports the aggregate health across all registered checks.
func (r *Registry) Status(ctx context.Context) Status {
	r.mu.RLock()
	checks := make(map[string]Check, len(r.checks))
	for name, c := range r.checks {
		checks[name] = c
	}
	r.mu.RUnlock()

	out := Status{Checks: make(map[string]string, len(checks))}
	for name, check := range checks {
		if err := check(ctx); err != nil {
			out.Checks[name] = err.Error()
			continue
		}
		out.Checks[name] = "ok"
	}
	if len(out.Checks) > 0 {
		for _, outcome := range out.Checks {
			if outcome != "ok" {
				out.Status = "degraded"
				break
			}
		}
	}
	if out.Status == "" {
		out.Status = "ok"
	}
	return out
}

// Status is the wire shape returned by the health endpoints.
type Status struct {
	Status string            `json:"status"`           // ok | degraded
	Checks map[string]string `json:"checks,omitempty"` // name -> outcome
}

// Handler serves the aggregate health report: 200 when ok, 503 when degraded.
func Handler(reg *Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := reg.Status(r.Context())
		code := http.StatusOK
		if s.Status != "ok" {
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(s)
	}
}
