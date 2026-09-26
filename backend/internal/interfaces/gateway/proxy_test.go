package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/health"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

const (
	testGatewayKey  = "sk-gateway-test"
	testUpstreamKey = "sk-ant-upstream-test"
	testSecretRef   = "ref:acct:live"
)

var (
	testOrgID  = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	testAcctID = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testCredID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
)

type upstreamSnapshot struct {
	headers http.Header
	uri     string
	body    []byte
}

// ---- stubs ---------------------------------------------------------------

type stubHasher struct{ prefix string }

func (s stubHasher) Hash(_ context.Context, secret string) (string, error) {
	return s.prefix + secret, nil
}

type stubCredentialRepo struct {
	byHash map[string]domain.APICredential
}

func (s *stubCredentialRepo) Create(context.Context, uuid.UUID, domain.NewAPICredential) (domain.APICredential, error) {
	panic("unused")
}
func (s *stubCredentialRepo) List(context.Context, uuid.UUID) ([]domain.APICredential, error) {
	panic("unused")
}
func (s *stubCredentialRepo) ResolveByKeyHash(_ context.Context, keyHash string) (domain.APICredential, error) {
	cred, ok := s.byHash[keyHash]
	if !ok {
		return domain.APICredential{}, repositories.ErrNotFound
	}
	return cred, nil
}
func (s *stubCredentialRepo) Revoke(context.Context, uuid.UUID, uuid.UUID) (domain.APICredential, error) {
	panic("unused")
}

type stubAccountsRepo struct{ accounts []domain.ProviderAccount }

func (s *stubAccountsRepo) Create(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error) {
	panic("unused")
}
func (s *stubAccountsRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
	panic("unused")
}
func (s *stubAccountsRepo) List(context.Context, uuid.UUID) ([]domain.ProviderAccount, error) {
	return s.accounts, nil
}

type stubProviderCredRepo struct{ creds []domain.ProviderCredential }

func (s *stubProviderCredRepo) Create(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error) {
	panic("unused")
}
func (s *stubProviderCredRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
	panic("unused")
}
func (s *stubProviderCredRepo) List(context.Context, uuid.UUID) ([]domain.ProviderCredential, error) {
	panic("unused")
}
func (s *stubProviderCredRepo) ListByAccount(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error) {
	return s.creds, nil
}
func (s *stubProviderCredRepo) Revoke(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
	panic("unused")
}

type stubSecretStore struct{ values map[string]string }

func (s *stubSecretStore) Get(_ context.Context, ref string) (string, error) {
	v, ok := s.values[ref]
	if !ok {
		return "", secrets.ErrNotFound
	}
	return v, nil
}
func (s *stubSecretStore) Put(_ context.Context, ref, value string) error {
	s.values[ref] = value
	return nil
}
func (s *stubSecretStore) Delete(_ context.Context, ref string) error {
	delete(s.values, ref)
	return nil
}

// ---- fixtures ------------------------------------------------------------

func defaultAccounts() []domain.ProviderAccount {
	return []domain.ProviderAccount{{
		ID:             testAcctID,
		OrganizationID: testOrgID,
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "main",
		Status:         domain.ProviderStatusActive,
	}}
}

func defaultCreds() []domain.ProviderCredential {
	return []domain.ProviderCredential{{
		ID:                testCredID,
		OrganizationID:    testOrgID,
		ProviderAccountID: testAcctID,
		Name:              "live",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         testSecretRef,
		Status:            domain.ProviderCredentialStatusActive,
	}}
}

func defaultRefs() map[string]string {
	return map[string]string{testSecretRef: testUpstreamKey}
}

func defaultGatewayCreds() map[string]domain.APICredential {
	return map[string]domain.APICredential{
		"h:" + testGatewayKey: {
			ID:             uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			OrganizationID: testOrgID,
			Name:           "claude-code-1",
			KeyPrefix:      "sk-gateway",
			KeyHash:        "h:" + testGatewayKey,
			Scope:          string(domain.APICredentialScopeGateway),
			Status:         domain.APICredentialStatusActive,
		},
	}
}

// newProxyHandler wires a Proxy with stubbed deps into the full middleware
// chain and returns the resulting http.Handler. The hasher prefix matches the
// stub credential key hashes.
func newProxyHandler(t *testing.T, upstreamURL string, accounts []domain.ProviderAccount,
	creds []domain.ProviderCredential, credsByHash map[string]domain.APICredential,
	refs map[string]string) http.Handler {
	t.Helper()

	proxy, err := NewProxy(ProxyDeps{
		Log:                 slog.New(slog.NewTextHandler(io.Discard, nil)),
		Hasher:              stubHasher{prefix: "h:"},
		Credentials:         &stubCredentialRepo{byHash: credsByHash},
		Accounts:            &stubAccountsRepo{accounts: accounts},
		ProviderCredentials: &stubProviderCredRepo{creds: creds},
		Secrets:             &stubSecretStore{values: refs},
		UpstreamBaseURL:     upstreamURL,
	})
	if err != nil {
		t.Fatalf("NewProxy: %v", err)
	}
	// Gateway-style chain: no per-request deadline (streaming).
	mw := httpx.NewMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), httpx.Options{})
	return NewHandler(nil, mw, health.NewRegistry(), Deps{Proxy: proxy})
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (r *flushRecorder) Flush() { r.flushes++ }

type errorEnvelope struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("error body is not an Anthropic envelope: %v\n%s", err, body)
	}
	if env.Type != "error" {
		t.Errorf("envelope type = %q, want error", env.Type)
	}
	return env
}

// ---- tests ---------------------------------------------------------------

func TestProxyForwardsRequestByteExact(t *testing.T) {
	var got upstreamSnapshot
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.headers = r.Header.Clone()
		got.uri = r.URL.RequestURI()
		got.body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Upstream-Trace", "abc")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"msg_01","type":"message"}`))
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	body := `{"model":"claude-sonnet-4-5","max_tokens":128}`
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
	req.Header.Set("x-api-key", testGatewayKey)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("X-Custom", "keep-me")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != `{"id":"msg_01","type":"message"}` {
		t.Errorf("response body = %q, want byte-exact upstream body", rr.Body.String())
	}
	if rr.Header().Get("Content-Type") != "application/json" {
		t.Errorf("response Content-Type = %q, want application/json", rr.Header().Get("Content-Type"))
	}
	if rr.Header().Get("X-Upstream-Trace") != "abc" {
		t.Errorf("response X-Upstream-Trace = %q, want abc", rr.Header().Get("X-Upstream-Trace"))
	}

	if got.headers.Get("x-api-key") != testUpstreamKey {
		t.Errorf("upstream x-api-key = %q, want injected %q", got.headers.Get("x-api-key"), testUpstreamKey)
	}
	if got.headers.Get("anthropic-version") != defaultAnthropicVersion {
		t.Errorf("upstream anthropic-version = %q, want default %q", got.headers.Get("anthropic-version"), defaultAnthropicVersion)
	}
	if got.uri != "/v1/messages" {
		t.Errorf("upstream URI = %q, want /v1/messages", got.uri)
	}
	if string(got.body) != body {
		t.Errorf("upstream body = %q, want %q", got.body, body)
	}
	if got.headers.Get("Connection") != "" {
		t.Errorf("hop-by-hop Connection header forwarded: %q", got.headers.Get("Connection"))
	}
	if got.headers.Get("X-Custom") != "keep-me" {
		t.Errorf("end-to-end X-Custom header not forwarded: %q", got.headers.Get("X-Custom"))
	}
}

func TestProxyAuthBearerFallback(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+testGatewayKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (Bearer token); body: %s", rr.Code, rr.Body.String())
	}
}

func TestProxyClientVersionPreserved(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("anthropic-version")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("x-api-key", testGatewayKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got != "2023-06-01" {
		t.Errorf("upstream anthropic-version = %q, want client value preserved", got)
	}
}

func TestProxyQueryPreserved(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages?beta=true", strings.NewReader(`{}`))
	req.Header.Set("x-api-key", testGatewayKey)
	h.ServeHTTP(httptest.NewRecorder(), req)

	if got != "/v1/messages?beta=true" {
		t.Errorf("upstream URI = %q, want /v1/messages?beta=true", got)
	}
}

func TestProxyCountTokensRoute(t *testing.T) {
	var got string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"input_tokens":42}`))
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", strings.NewReader(`{"model":"claude-sonnet-4-5"}`))
	req.Header.Set("x-api-key", testGatewayKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got != "/v1/messages/count_tokens" {
		t.Errorf("upstream URI = %q, want /v1/messages/count_tokens", got)
	}
	if rr.Body.String() != `{"input_tokens":42}` {
		t.Errorf("body = %q, want byte-exact upstream body", rr.Body.String())
	}
}

func TestProxyUnauthorized(t *testing.T) {
	cases := []struct {
		name        string
		credsByHash map[string]domain.APICredential
		headerSet   func(*http.Request)
	}{
		{
			name:        "missing key",
			credsByHash: defaultGatewayCreds(),
			headerSet:   func(*http.Request) {},
		},
		{
			name:        "unknown key",
			credsByHash: defaultGatewayCreds(),
			headerSet: func(r *http.Request) {
				r.Header.Set("x-api-key", "sk-wrong")
			},
		},
		{
			name: "revoked key",
			credsByHash: map[string]domain.APICredential{"h:" + testGatewayKey: {ID: uuid.MustParse("55555555-5555-5555-5555-555555555555"), OrganizationID: testOrgID, Scope: string(domain.APICredentialScopeGateway), Status: domain.APICredentialStatusRevoked}},
			headerSet: func(r *http.Request) {
				r.Header.Set("x-api-key", testGatewayKey)
			},
		},
		{
			name: "control-scoped key",
			credsByHash: map[string]domain.APICredential{"h:" + testGatewayKey: {ID: uuid.MustParse("66666666-6666-6666-6666-666666666666"), OrganizationID: testOrgID, Scope: string(domain.APICredentialScopeControl), Status: domain.APICredentialStatusActive}},
			headerSet: func(r *http.Request) {
				r.Header.Set("x-api-key", testGatewayKey)
			},
		},
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream must not be called for unauthorized requests")
	}))
	defer upstream.Close()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), tc.credsByHash, defaultRefs())
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
			tc.headerSet(req)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401; body: %s", rr.Code, rr.Body.String())
			}
			env := decodeError(t, rr.Body.Bytes())
			if env.Error.Type != "authentication_error" {
				t.Errorf("error type = %q, want authentication_error", env.Error.Type)
			}
		})
	}
}

func TestProxyServiceUnavailableNoUpstream(t *testing.T) {
	cases := []struct {
		name     string
		accounts []domain.ProviderAccount
		creds    []domain.ProviderCredential
		refs     map[string]string
	}{
		{
			name:     "no account",
			accounts: nil,
			creds:    defaultCreds(),
			refs:     defaultRefs(),
		},
		{
			name:     "wrong provider kind",
			accounts: []domain.ProviderAccount{{ID: testAcctID, OrganizationID: testOrgID, ProviderKind: domain.ProviderKindOpenAI, Status: domain.ProviderStatusActive}},
			creds:    defaultCreds(),
			refs:     defaultRefs(),
		},
		{
			name:     "disabled account",
			accounts: []domain.ProviderAccount{{ID: testAcctID, OrganizationID: testOrgID, ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderStatusDisabled}},
			creds:    defaultCreds(),
			refs:     defaultRefs(),
		},
		{
			name:     "no credentials",
			accounts: defaultAccounts(),
			creds:    nil,
			refs:     defaultRefs(),
		},
		{
			name:     "revoked credential",
			accounts: defaultAccounts(),
			creds:    []domain.ProviderCredential{{ID: testCredID, OrganizationID: testOrgID, ProviderAccountID: testAcctID, SecretRef: testSecretRef, Status: domain.ProviderCredentialStatusRevoked}},
			refs:     defaultRefs(),
		},
		{
			name:     "missing secret",
			accounts: defaultAccounts(),
			creds:    defaultCreds(),
			refs:     map[string]string{},
		},
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream must not be called when no credential resolves")
	}))
	defer upstream.Close()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newProxyHandler(t, upstream.URL, tc.accounts, tc.creds, defaultGatewayCreds(), tc.refs)
			req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
			req.Header.Set("x-api-key", testGatewayKey)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503; body: %s", rr.Code, rr.Body.String())
			}
			env := decodeError(t, rr.Body.Bytes())
			if env.Error.Type != "api_error" {
				t.Errorf("error type = %q, want api_error", env.Error.Type)
			}
		})
	}
}

func TestProxyAccountBaseURLOverride(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request must not reach the default upstream when the account sets BaseURL")
	}))
	defer first.Close()

	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"from":"override"}`))
	}))
	defer second.Close()

	accounts := []domain.ProviderAccount{{
		ID:             testAcctID,
		OrganizationID: testOrgID,
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "gateway-owned",
		BaseURL:        second.URL,
		Status:         domain.ProviderStatusActive,
	}}

	h := newProxyHandler(t, first.URL, accounts, defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("x-api-key", testGatewayKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != `{"from":"override"}` {
		t.Errorf("body = %q, want the account-BaseURL upstream response", rr.Body.String())
	}
}

func TestProxyStreamingFlushesChunks(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Error("upstream recorder is not a flusher")
			return
		}
		for _, chunk := range []string{"data: hello\n\n", "data: world\n\n"} {
			if _, err := io.WriteString(w, chunk); err != nil {
				t.Errorf("upstream write: %v", err)
				return
			}
			fl.Flush()
		}
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"stream":true}`))
	req.Header.Set("x-api-key", testGatewayKey)
	fr := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	h.ServeHTTP(fr, req)

	if fr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", fr.Code)
	}
	if fr.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", fr.Header().Get("Content-Type"))
	}
	if fr.Body.String() != "data: hello\n\ndata: world\n\n" {
		t.Errorf("streamed body = %q, want concatenated chunks", fr.Body.String())
	}
	if fr.flushes < 2 {
		t.Errorf("flushes = %d, want >= 2 (one per upstream chunk)", fr.flushes)
	}
}

func TestProxyUpstreamErrorPassthrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"over limit"}}`))
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("x-api-key", testGatewayKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want upstream 429 passthrough", rr.Code)
	}
	if rr.Body.String() != `{"type":"error","error":{"type":"rate_limit_error","message":"over limit"}}` {
		t.Errorf("body = %q, want byte-exact upstream error", rr.Body.String())
	}
	if rr.Header().Get("Retry-After") != "5" {
		t.Errorf("Retry-After = %q, want 5", rr.Header().Get("Retry-After"))
	}
}

func TestProxyUpstreamUnreachable(t *testing.T) {
	dead := httptest.NewServer(http.NewServeMux())
	dead.Close()

	h := newProxyHandler(t, dead.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("x-api-key", testGatewayKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body: %s", rr.Code, rr.Body.String())
	}
	env := decodeError(t, rr.Body.Bytes())
	if env.Error.Type != "api_error" {
		t.Errorf("error type = %q, want api_error", env.Error.Type)
	}
}

func TestProxyMethodNotAllowed(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream must not be called for non-POST")
	}))
	defer upstream.Close()

	h := newProxyHandler(t, upstream.URL, defaultAccounts(), defaultCreds(), defaultGatewayCreds(), defaultRefs())

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/v1/messages", nil))

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestProxyWithoutProxyDepsIsHealthOnly(t *testing.T) {
	mw := httpx.NewMiddleware(slog.New(slog.NewTextHandler(io.Discard, nil)), httpx.Options{})
	h := NewHandler(nil, mw, health.NewRegistry(), Deps{})

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{}`)))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when the proxy is not wired", rr.Code)
	}
}

func TestNewProxyRequiresDeps(t *testing.T) {
	for name, deps := range map[string]ProxyDeps{
		"no hasher":               {Credentials: &stubCredentialRepo{}, Accounts: &stubAccountsRepo{}, ProviderCredentials: &stubProviderCredRepo{}, Secrets: &stubSecretStore{}},
		"no credentials":          {Hasher: stubHasher{prefix: "h:"}, Accounts: &stubAccountsRepo{}, ProviderCredentials: &stubProviderCredRepo{}, Secrets: &stubSecretStore{}},
		"no accounts":             {Hasher: stubHasher{prefix: "h:"}, Credentials: &stubCredentialRepo{}, ProviderCredentials: &stubProviderCredRepo{}, Secrets: &stubSecretStore{}},
		"no provider credentials": {Hasher: stubHasher{prefix: "h:"}, Credentials: &stubCredentialRepo{}, Accounts: &stubAccountsRepo{}, Secrets: &stubSecretStore{}},
		"no secrets":              {Hasher: stubHasher{prefix: "h:"}, Credentials: &stubCredentialRepo{}, Accounts: &stubAccountsRepo{}, ProviderCredentials: &stubProviderCredRepo{}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewProxy(deps); err == nil {
				t.Error("NewProxy must reject missing deps")
			}
		})
	}
}
