// Package httpx contains the shared standard-library HTTP plumbing for every
// AI Governance process: the middleware chain (request ID, correlation ID,
// panic recovery, access logging, per-request deadlines), JSON response
// conventions and the structured error envelope.
//
// The package is deliberately free of third-party imports: the processes
// serve the hot path and the standard library is proven sufficient here.
package httpx

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/platform/ids"
)

const (
	defaultRequestIDHeader = "X-Request-Id"
	correlationIDHeader    = "X-Correlation-Id"
	maxHeaderValueLength   = 128
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyCorrelationID
	ctxKeyLogger
)

// RequestIDFrom returns the request ID stored in ctx, if any.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// CorrelationIDFrom returns the correlation ID stored in ctx, if any.
func CorrelationIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyCorrelationID).(string)
	return v
}

// LoggerFrom returns the request-scoped logger from ctx, falling back to the
// default logger.
func LoggerFrom(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Options tunes the shared middleware chain.
type Options struct {
	// RequestIDHeader names the inbound header honored as an upstream request
	// ID (default X-Request-Id); the value is sanitized and truncated.
	RequestIDHeader string
	// DefaultRequestTimeout is the per-request deadline applied by the
	// timeout middleware; 0 disables it (the gateway uses 0 once streaming).
	DefaultRequestTimeout time.Duration
	// CORSAllowedOrigins is the exact-match allowlist for cross-origin
	// requests. When empty, CORS is disabled entirely: an Origin header is
	// never echoed and preflights fall through to normal routing. Wildcards
	// are not supported; credentials (cookies) are only echoed for an origin
	// on this list.
	CORSAllowedOrigins []string
}

// DefaultOptions returns the sane defaults for the middleware chain.
func DefaultOptions() Options {
	return Options{RequestIDHeader: defaultRequestIDHeader}
}

// Middleware builds the shared request pipeline for a process.
type Middleware struct {
	opts Options
	log  *slog.Logger
}

// NewMiddleware creates a Middleware; a nil logger falls back to the default.
func NewMiddleware(log *slog.Logger, opts Options) *Middleware {
	if log == nil {
		log = slog.Default()
	}
	if opts.RequestIDHeader == "" {
		opts.RequestIDHeader = defaultRequestIDHeader
	}
	return &Middleware{opts: opts, log: log}
}

// Chain wraps h in the request pipeline. Execution order (outermost first):
// request ID -> correlation ID -> security headers -> CORS -> panic recovery
// -> access log -> deadline.
func (m *Middleware) Chain(h http.Handler) http.Handler {
	return withRequestID(m.opts.RequestIDHeader,
		withCorrelationID(
			withSecurityHeaders(
				withCORS(m.opts.CORSAllowedOrigins,
					withRecovery(m.log,
						withAccessLog(m.log,
							withTimeout(m.opts.DefaultRequestTimeout, h)))))))
}

func withRequestID(header string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := sanitizeHeader(r.Header.Get(header))
		if reqID == "" {
			reqID = ids.New()
		}
		w.Header().Set(defaultRequestIDHeader, reqID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyRequestID, reqID)))
	})
}

// withCorrelationID always issues a fresh correlation ID per request.
func withCorrelationID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corrID := ids.NewCorrelationID()
		w.Header().Set(correlationIDHeader, corrID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyCorrelationID, corrID)))
	})
}

// withRecovery converts panics into a structured 500 and logs the stack — a
// panic must never crash the process. It records the response itself so an
// already-committed response is never overwritten.
func withRecovery(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w}
		defer func() {
			if p := recover(); p != nil {
				reqID := RequestIDFrom(r.Context())
				log.Error("panic recovered",
					"panic", fmt.Sprint(p),
					"stack", string(debug.Stack()),
					"request_id", reqID,
					"path", r.URL.Path,
				)
				if !rec.wroteHeader {
					WriteError(rec, http.StatusInternalServerError, reqID,
						errors.New(errors.KindInternal, "internal_error", "internal error"))
				}
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// withTimeout imposes a per-request deadline; after it fires it reports a 504
// only if the handler has not already written a response.
func withTimeout(d time.Duration, next http.Handler) http.Handler {
	if d <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r.WithContext(ctx))

		if ctx.Err() == context.DeadlineExceeded && !rec.wroteHeader {
			WriteError(w, http.StatusGatewayTimeout, RequestIDFrom(ctx),
				errors.New(errors.KindTimeout, "request_timeout", "request deadline exceeded"))
		}
	})
}

// withAccessLog records one log line per request with its outcome.
func withAccessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		log.LogAttrs(r.Context(), slog.LevelInfo, "http request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", RequestIDFrom(r.Context())),
			slog.String("correlation_id", CorrelationIDFrom(r.Context())),
		)
	})
}

// statusRecorder captures the outcome of a response while preserving
// streaming (Flush) and HTTP/2 response control (Unwrap).
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

// Flush forwards to the underlying writer so SSE/streaming responses keep
// flowing through the middleware chain.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func sanitizeHeader(v string) string {
	return strings.TrimSpace(truncate(v, maxHeaderValueLength))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// withSecurityHeaders applies the always-on response headers that lock down
// the browser-facing surface: no MIME sniffing, no framing, no referrer
// leakage and no inline content.
func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

const (
	corsAllowMethods = "GET, POST, PATCH, PUT, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization, X-Request-Id, X-CSRF-Token, X-API-Key"
	corsMaxAge       = "600"
)

// withCORS implements an exact-match, credentials-aware CORS policy. A
// request whose Origin is on the allowlist gets its origin echoed back with
// Allow-Credentials: true; any other origin receives no ACAO header, so the
// browser blocks the cross-origin read. Preflights from allowed origins
// short-circuit with a 204 before reaching the handler.
func withCORS(allowed []string, next http.Handler) http.Handler {
	if len(allowed) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || !slices.Contains(allowed, origin) {
			next.ServeHTTP(w, r)
			return
		}
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Add("Vary", "Origin")
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			h.Set("Access-Control-Allow-Methods", corsAllowMethods)
			h.Set("Access-Control-Allow-Headers", corsAllowHeaders)
			h.Set("Access-Control-Max-Age", corsMaxAge)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
