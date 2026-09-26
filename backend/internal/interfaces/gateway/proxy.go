// Package gateway implements the data-plane HTTP surface: the health probes
// and the provider streaming proxy. The proxy routes Claude Code traffic to
// the calling tenant's active Anthropic upstream, byte-exact, with SSE
// flushing, and returns provider-native errors (Anthropic error envelope).
package gateway

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

const (
	// defaultAnthropicBaseURL is used when neither the provider account nor
	// the process configuration specifies a base URL.
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	// defaultAnthropicVersion is injected when the client omits the header.
	defaultAnthropicVersion = "2023-06-01"
	// proxyBufferSize is the streaming copy chunk size.
	proxyBufferSize = 32 * 1024
)

// ProxyDeps supplies the tenant-resolution dependencies for the streaming
// proxy. Every field except Client/UpstreamBaseURL is required; NewProxy
// validates the set.
type ProxyDeps struct {
	Log                 *slog.Logger
	Hasher              ports.TokenHasher
	Credentials         repositories.APICredentialRepository
	Accounts            repositories.ProviderAccountRepository
	ProviderCredentials repositories.ProviderCredentialRepository
	Secrets             secrets.Store
	// UpstreamBaseURL is the default upstream for accounts without a BaseURL;
	// empty means the Anthropic default. An account BaseURL always wins.
	UpstreamBaseURL string
	// Client is optional; a streaming-safe default (no overall timeout,
	// compression disabled) is used when nil.
	Client *http.Client
}

// Proxy forwards /v1/messages* calls to the tenant's active Anthropic
// upstream. It implements http.Handler; the routes are registered by the
// server wiring when the proxy is present.
type Proxy struct {
	deps    ProxyDeps
	client  *http.Client
	baseURL string
}

// NewProxy validates deps and returns a ready proxy.
func NewProxy(deps ProxyDeps) (*Proxy, error) {
	if deps.Hasher == nil {
		return nil, errors.New("gateway: NewProxy requires Hasher")
	}
	if deps.Credentials == nil {
		return nil, errors.New("gateway: NewProxy requires Credentials")
	}
	if deps.Accounts == nil {
		return nil, errors.New("gateway: NewProxy requires Accounts")
	}
	if deps.ProviderCredentials == nil {
		return nil, errors.New("gateway: NewProxy requires ProviderCredentials")
	}
	if deps.Secrets == nil {
		return nil, errors.New("gateway: NewProxy requires Secrets")
	}
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	if deps.Client == nil {
		deps.Client = defaultProxyClient()
	}
	baseURL := strings.TrimRight(deps.UpstreamBaseURL, "/")
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	return &Proxy{deps: deps, client: deps.Client, baseURL: baseURL}, nil
}

// defaultProxyClient returns the outbound client for upstream calls. It has
// no overall timeout: streaming responses may run for minutes. The request
// context cancels the upstream call when the client disconnects.
func defaultProxyClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			// Forward Accept-Encoding verbatim and never decode upstream
			// bodies: the gateway is a byte-exact pipe.
			DisableCompression: true,
			// Guard against upstreams that accept but never respond.
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

// ServeHTTP authenticates the caller, resolves the tenant's upstream
// credential, and streams the request/response byte-exact.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := httpx.LoggerFrom(r.Context())

	rawKey, ok := extractAPIToken(r)
	if !ok {
		writeAnthropicError(w, http.StatusUnauthorized, "authentication_error", "invalid x-api-key")
		return
	}

	hash, err := p.deps.Hasher.Hash(r.Context(), rawKey)
	if err != nil {
		logger.Error("gateway: hash api key", "error", err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "internal error")
		return
	}

	cred, err := p.deps.Credentials.ResolveByKeyHash(r.Context(), hash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			writeAnthropicError(w, http.StatusUnauthorized, "authentication_error", "invalid x-api-key")
			return
		}
		logger.Error("gateway: resolve api key", "error", err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "internal error")
		return
	}
	// The gateway plane only accepts gateway/all-scoped active keys; a key
	// revoked here or scoped to the control plane is rejected identically.
	if cred.Status != domain.APICredentialStatusActive ||
		(cred.Scope != string(domain.APICredentialScopeGateway) && cred.Scope != string(domain.APICredentialScopeAll)) {
		writeAnthropicError(w, http.StatusUnauthorized, "authentication_error", "invalid x-api-key")
		return
	}

	// resolveUpstream is hot-path tenant resolution: never trust any inbound
	// tenant header; the owning organization comes from the key record.
	upstreamKey, upstreamBase, err := p.resolveUpstream(r.Context(), cred.OrganizationID)
	if err != nil {
		if errors.Is(err, errNoUpstream) {
			writeAnthropicError(w, http.StatusServiceUnavailable, "api_error", "no active Anthropic credential configured")
			return
		}
		logger.Error("gateway: resolve upstream credential", "organization_id", cred.OrganizationID, "error", err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "internal error")
		return
	}

	p.forward(w, r, logger, upstreamBase, upstreamKey)
}

// errNoUpstream marks a tenant with no routable Anthropic configuration.
var errNoUpstream = errors.New("gateway: no active upstream provider credential")

// resolveUpstream finds the tenant's active Anthropic account and its newest
// active credential, returning the upstream API key and the effective base
// URL (account BaseURL wins over the proxy default).
func (p *Proxy) resolveUpstream(ctx context.Context, orgID uuid.UUID) (key, baseURL string, err error) {
	accounts, err := p.deps.Accounts.List(ctx, orgID)
	if err != nil {
		return "", "", err
	}
	var account *domain.ProviderAccount
	for i := range accounts {
		if accounts[i].ProviderKind == domain.ProviderKindAnthropic &&
			accounts[i].Status == domain.ProviderStatusActive {
			account = &accounts[i]
			break
		}
	}
	if account == nil {
		return "", "", errNoUpstream
	}

	creds, err := p.deps.ProviderCredentials.ListByAccount(ctx, orgID, account.ID)
	if err != nil {
		return "", "", err
	}
	// ListByAccount is newest-first, active-only: the first entry is the live
	// rotation candidate.
	if len(creds) == 0 || creds[0].Status != domain.ProviderCredentialStatusActive {
		return "", "", errNoUpstream
	}

	key, err = p.deps.Secrets.Get(ctx, creds[0].SecretRef)
	if err != nil {
		if errors.Is(err, secrets.ErrNotFound) {
			return "", "", errNoUpstream
		}
		return "", "", err
	}

	baseURL = p.baseURL
	if account.BaseURL != "" {
		baseURL = strings.TrimRight(account.BaseURL, "/")
	}
	return key, baseURL, nil
}

// forward streams the inbound request to the upstream, then copies the
// response back byte-exact: status, end-to-end headers, and the body with SSE
// flushing.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, logger *slog.Logger, upstreamBase, upstreamKey string) {
	// RequestURI() preserves the path and query without the host, so the
	// upstream receives the identical /v1/messages* request.
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamBase+r.URL.RequestURI(), r.Body)
	if err != nil {
		logger.Error("gateway: build upstream request", "error", err)
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", "internal error")
		return
	}
	if r.ContentLength >= 0 {
		outReq.ContentLength = r.ContentLength
	}
	CopyForwardHeaders(outReq.Header, r.Header)
	outReq.Header.Set("x-api-key", upstreamKey)
	if outReq.Header.Get("anthropic-version") == "" {
		outReq.Header.Set("anthropic-version", defaultAnthropicVersion)
	}

	resp, err := p.client.Do(outReq)
	if err != nil {
		logger.Error("gateway: upstream request failed", "error", err)
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "upstream request failed")
		return
	}
	defer resp.Body.Close()

	CopyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if err := copyStream(w, resp.Body); err != nil {
		// Headers are committed; a mid-stream failure only truncates the
		// body, which is the correct proxy behavior on a dropped connection.
		logger.Warn("gateway: stream upstream response", "error", err)
	}
}

// extractAPIToken returns the gateway API key from x-api-key, falling back to
// an Authorization: Bearer header (Claude Code sends either).
func extractAPIToken(r *http.Request) (string, bool) {
	if v := strings.TrimSpace(r.Header.Get("x-api-key")); v != "" {
		return v, true
	}
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) > len(prefix) && strings.EqualFold(auth[:len(prefix)], prefix) {
		if v := strings.TrimSpace(auth[len(prefix):]); v != "" {
			return v, true
		}
	}
	return "", false
}

// hopByHopHeaders are the connection-scoped headers (RFC 9110 9.6) that a
// proxy must consume and never forward.
var hopByHopHeaders = []string{
	"connection", "keep-alive", "proxy-authenticate", "proxy-authorization",
	"te", "trailer", "transfer-encoding", "upgrade",
}

func isHopByHop(key string) bool {
	key = strings.ToLower(key)
	for _, h := range hopByHopHeaders {
		if key == h {
			return true
		}
	}
	return false
}

// copyForwardHeaders copies end-to-end headers onto dst, dropping hop-by-hop
// headers and the client's authentication/material headers; the proxy injects
// its own upstream key afterward. Header fields named by a Connection header
// are covered by the hop-by-hop list for the standard cases.
// CopyForwardHeaders copies end-to-end headers from src to dst, dropping hop-by-hop
// and authentication headers. Exported for use by other packages.
func CopyForwardHeaders(dst, src http.Header) {
	for key, values := range src {
		lower := strings.ToLower(key)
		if isHopByHop(lower) || lower == "host" || lower == "content-length" ||
			lower == "authorization" || lower == "x-api-key" {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// copyResponseHeaders copies end-to-end headers onto dst, dropping hop-by-hop
// headers and framing that the Go response writer manages itself.
// CopyResponseHeaders copies end-to-end headers from src to dst, dropping hop-by-hop
// headers and framing that the Go response writer manages itself.
func CopyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		lower := strings.ToLower(key)
		if isHopByHop(lower) || lower == "content-length" || lower == "transfer-encoding" {
			continue
		}
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

// copyStream copies src to w in proxyBufferSize chunks, flushing after every
// chunk when w supports it so SSE and streaming responses stay live.
func copyStream(w http.ResponseWriter, src io.Reader) error {
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, proxyBufferSize)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}
	}
}

// anthropicErrorBody is the provider-native error envelope returned on the
// gateway so Claude Code surfaces failures exactly like Anthropic's API.
type anthropicErrorBody struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeAnthropicError responds with an Anthropic-shaped error body.
func writeAnthropicError(w http.ResponseWriter, status int, etype, message string) {
	body := anthropicErrorBody{Type: "error"}
	body.Error.Type = etype
	body.Error.Message = message
	httpx.WriteJSON(w, status, body)
}
