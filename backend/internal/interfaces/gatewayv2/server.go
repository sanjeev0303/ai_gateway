// Package gatewayv2 implements the latency-sensitive data-plane gateway.
package gatewayv2

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// Deps carries the gateway dependencies; routes register only when
// their dependency is non-nil.
type Deps struct {
	Gateway *Gateway
}

// NewHandler assembles the gateway HTTP handler: health probes, plus the
// provider streaming proxy when wired, wrapped in the shared middleware chain.
func NewHandler(log *slog.Logger, mw *httpx.Middleware, reg *health.Registry, deps Deps) http.Handler {
	mux := http.NewServeMux()
	registerHealthRoutes(mux, reg)
	if deps.Gateway != nil {
		registerGatewayRoutes(mux, deps.Gateway)
	}
	return mw.Chain(mux)
}

func registerHealthRoutes(mux *http.ServeMux, reg *health.Registry) {
	mux.Handle("GET /livez", health.Handler(reg))
	mux.Handle("GET /readyz", health.Handler(reg))
	mux.Handle("GET /healthz", health.Handler(reg))
}

// registerGatewayRoutes wires the gateway routes.
func registerGatewayRoutes(mux *http.ServeMux, gw *Gateway) {
	// Anthropic messages endpoint
	mux.Handle("POST /v1/messages", gw)
	// Token counting endpoint
	mux.Handle("POST /v1/messages/count_tokens", gw)
	// OpenAI-compatible chat and completions endpoints
	mux.Handle("POST /v1/chat/completions", gw)
	mux.Handle("POST /v1/completions", gw)
	mux.Handle("POST /v1/embeddings", gw)
	// Model listing (if needed)
	mux.Handle("GET /v1/models", gw)
	// Admin/debug endpoints
	mux.Handle("GET /v1/gateway/stats", http.HandlerFunc(gw.statsHandler))
	mux.Handle("GET /v1/gateway/health", http.HandlerFunc(gw.healthHandler))
}

// statsHandler returns gateway statistics.
func (g *Gateway) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := g.Stats()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// healthHandler returns gateway health status.
func (g *Gateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := g.HealthCheck(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}
