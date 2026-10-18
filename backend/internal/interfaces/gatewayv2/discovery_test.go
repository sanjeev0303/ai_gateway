package gatewayv2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	"github.com/google/uuid"
)

var errUpstreamUnavailable = errors.New("upstream unavailable")

// stubMultiOrgTenantResolver implements TenantResolver for multi-tenant testing.
type stubMultiOrgTenantResolver struct {
	orgProviders map[uuid.UUID]map[string]ResolvedUpstream
}

func newStubMultiOrgTenantResolver() *stubMultiOrgTenantResolver {
	return &stubMultiOrgTenantResolver{
		orgProviders: make(map[uuid.UUID]map[string]ResolvedUpstream),
	}
}

func (r *stubMultiOrgTenantResolver) SetUpstream(orgID uuid.UUID, providerKind string, up ResolvedUpstream) {
	if r.orgProviders[orgID] == nil {
		r.orgProviders[orgID] = make(map[string]ResolvedUpstream)
	}
	r.orgProviders[orgID][providerKind] = up
}

func (r *stubMultiOrgTenantResolver) ResolveUpstream(ctx context.Context, orgID uuid.UUID) (ResolvedUpstream, error) {
	providers := r.orgProviders[orgID]
	for _, up := range providers {
		if up.ProviderAccount.Status == domain.ProviderStatusActive && up.Credential.Status == domain.ProviderCredentialStatusActive {
			return up, nil
		}
	}
	return ResolvedUpstream{}, errUpstreamUnavailable
}

func (r *stubMultiOrgTenantResolver) ResolveUpstreamForProvider(ctx context.Context, orgID uuid.UUID, providerKind string) (ResolvedUpstream, error) {
	providers := r.orgProviders[orgID]
	if providers == nil {
		return ResolvedUpstream{}, errUpstreamUnavailable
	}
	up, ok := providers[providerKind]
	if !ok || up.ProviderAccount.Status != domain.ProviderStatusActive || up.Credential.Status != domain.ProviderCredentialStatusActive {
		return ResolvedUpstream{}, errUpstreamUnavailable
	}
	return up, nil
}

func (r *stubMultiOrgTenantResolver) ListActiveProviders(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	providers := r.orgProviders[orgID]
	if providers == nil {
		return nil, nil
	}
	var active []string
	for kind, up := range providers {
		if up.ProviderAccount.Status == domain.ProviderStatusActive && up.Credential.Status == domain.ProviderCredentialStatusActive {
			active = append(active, kind)
		}
	}
	return active, nil
}

func (r *stubMultiOrgTenantResolver) ResolveUpstreamForAccount(ctx context.Context, orgID, accountID uuid.UUID) (ResolvedUpstream, error) {
	providers := r.orgProviders[orgID]
	if providers == nil {
		return ResolvedUpstream{}, ErrNoUpstream
	}
	for _, up := range providers {
		if up.ProviderAccount.ID == accountID {
			return up, nil
		}
	}
	return ResolvedUpstream{}, ErrNoUpstream
}

func (r *stubMultiOrgTenantResolver) Invalidate(orgID uuid.UUID) {}

type stubMultiOrgModelRepo struct {
	models []domain.Model
}

func (m *stubMultiOrgModelRepo) List(ctx context.Context) ([]domain.Model, error) {
	var active []domain.Model
	for _, mod := range m.models {
		if mod.IsActive && mod.Status != "inactive" {
			active = append(active, mod)
		}
	}
	return active, nil
}

func (m *stubMultiOrgModelRepo) ListAll(ctx context.Context) ([]domain.Model, error) {
	return m.models, nil
}

func (m *stubMultiOrgModelRepo) ListByProvider(ctx context.Context, providerKind string) ([]domain.Model, error) {
	var filtered []domain.Model
	for _, mod := range m.models {
		if mod.ProviderKind == providerKind {
			filtered = append(filtered, mod)
		}
	}
	return filtered, nil
}

func (m *stubMultiOrgModelRepo) GetBySlug(ctx context.Context, providerKind, modelSlug string) (domain.Model, error) {
	for _, mod := range m.models {
		if (providerKind == "" || mod.ProviderKind == providerKind) && mod.ModelSlug == modelSlug {
			return mod, nil
		}
	}
	return domain.Model{}, errors.New("model not found")
}

func (m *stubMultiOrgModelRepo) Upsert(ctx context.Context, provider, slug, display string) (domain.Model, error) {
	mod := domain.Model{ID: uuid.New(), ProviderKind: provider, ModelSlug: slug, DisplayName: display, IsActive: true, Status: "active"}
	m.models = append(m.models, mod)
	return mod, nil
}

func (m *stubMultiOrgModelRepo) UpsertWithMetadata(ctx context.Context, mod domain.Model) (domain.Model, error) {
	m.models = append(m.models, mod)
	return mod, nil
}

func (m *stubMultiOrgModelRepo) MarkInactiveExcept(ctx context.Context, provider string, activeSlugs []string) error {
	activeSet := make(map[string]bool)
	for _, s := range activeSlugs {
		activeSet[s] = true
	}
	for i := range m.models {
		if m.models[i].ProviderKind == provider && !activeSet[m.models[i].ModelSlug] {
			m.models[i].IsActive = false
			m.models[i].Status = "inactive"
		}
	}
	return nil
}

// TestMultiOrgModelDiscoveryAndIsolation validates that two organizations with different
// connected provider credentials only see and can only route to models for their connected providers.
func TestMultiOrgModelDiscoveryAndIsolation(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	orgC := uuid.New() // No providers connected

	tenantResolver := newStubMultiOrgTenantResolver()

	// Org A: Connected Anthropic only
	tenantResolver.SetUpstream(orgA, domain.ProviderKindAnthropic, ResolvedUpstream{
		ProviderAccount: ProviderAccount{
			ID:             uuid.New(),
			OrganizationID: orgA,
			ProviderKind:   domain.ProviderKindAnthropic,
			Name:           "Org A Anthropic",
			Status:         domain.ProviderStatusActive,
		},
		Credential: ProviderCredential{
			ID:             uuid.New(),
			OrganizationID: orgA,
			Status:         domain.ProviderCredentialStatusActive,
		},
		UpstreamKey: "sk-ant-orgA",
		BaseURL:     "https://api.anthropic.com",
	})

	// Org B: Connected OpenAI only
	tenantResolver.SetUpstream(orgB, domain.ProviderKindOpenAI, ResolvedUpstream{
		ProviderAccount: ProviderAccount{
			ID:             uuid.New(),
			OrganizationID: orgB,
			ProviderKind:   domain.ProviderKindOpenAI,
			Name:           "Org B OpenAI",
			Status:         domain.ProviderStatusActive,
		},
		Credential: ProviderCredential{
			ID:             uuid.New(),
			OrganizationID: orgB,
			Status:         domain.ProviderCredentialStatusActive,
		},
		UpstreamKey: "sk-openai-orgB",
		BaseURL:     "https://api.openai.com",
	})

	modelRepo := &stubMultiOrgModelRepo{
		models: []domain.Model{
			{
				ID:                uuid.New(),
				ProviderKind:      domain.ProviderKindAnthropic,
				ModelSlug:         "claude-3-5-sonnet",
				DisplayName:       "Claude 3.5 Sonnet",
				Capabilities:      []string{"text_generation", "streaming", "tool_calling"},
				ContextLimit:      200000,
				SupportsStreaming: true,
				SupportsTools:     true,
				Status:            "active",
				IsActive:          true,
			},
			{
				ID:                uuid.New(),
				ProviderKind:      domain.ProviderKindAnthropic,
				ModelSlug:         "claude-3-opus",
				DisplayName:       "Claude 3 Opus",
				Capabilities:      []string{"text_generation", "streaming"},
				ContextLimit:      200000,
				SupportsStreaming: true,
				Status:            "active",
				IsActive:          true,
			},
			{
				ID:                uuid.New(),
				ProviderKind:      domain.ProviderKindOpenAI,
				ModelSlug:         "gpt-4o",
				DisplayName:       "GPT-4o",
				Capabilities:      []string{"text_generation", "streaming", "tool_calling"},
				ContextLimit:      128000,
				SupportsStreaming: true,
				SupportsTools:     true,
				Status:            "active",
				IsActive:          true,
			},
			{
				ID:                uuid.New(),
				ProviderKind:      domain.ProviderKindOpenAI,
				ModelSlug:         "o1",
				DisplayName:       "o1",
				Capabilities:      []string{"text_generation", "reasoning"},
				ContextLimit:      200000,
				SupportsReasoning: true,
				Status:            "active",
				IsActive:          true,
			},
			{
				ID:           uuid.New(),
				ProviderKind: domain.ProviderKindAnthropic,
				ModelSlug:    "claude-2.1",
				DisplayName:  "Claude 2.1 (Deprecated)",
				Status:       "inactive",
				IsActive:     false,
			},
		},
	}

	authStub := &stubGatewayAuthenticator{}
	policyStub := &stubGatewayPolicyResolver{
		snapshot: PolicySnapshot{},
	}

	g := &Gateway{
		config:           DefaultGatewayConfig(),
		logger:           discardLogger(),
		authenticator:    authStub,
		tenantResolver:   tenantResolver,
		policyResolver:   policyStub,
		rateLimiter:      &stubGatewayRateLimiter{result: RateLimitResult{Allowed: true, Limit: 1000, Remaining: 999}},
		budgetGuard:      &stubGatewayBudgetGuard{result: BudgetResult{Allowed: true, Limit: 1000}},
		requestInspector: NewRequestInspector(),
		providerRouter:   &stubGatewayProviderRouter{url: "https://api.test/v1/messages", headers: http.Header{}},
		streamProxy:      &stubGatewayStreamProxy{},
		telemetry:        &stubGatewayTelemetry{},
		modelRepo:        modelRepo,
	}

	t.Run("Org A sees only Anthropic models, never OpenAI or inactive models", func(t *testing.T) {
		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgA,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgA}

		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("x-api-key", "aigw_orgA_key")
		rec := httptest.NewRecorder()

		g.ServeHTTP(rec, req)
		requireTrue(t, rec.Code == http.StatusOK, "want 200 OK")

		var resp struct {
			Data []struct {
				ID          string `json:"id"`
				Provider    string `json:"provider"`
				DisplayName string `json:"display_name"`
			} `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		requireNoErr(t, err, "unmarshal response")

		requireTrue(t, len(resp.Data) == 2, "Org A must see exactly 2 active Anthropic models")
		for _, m := range resp.Data {
			requireTrue(t, m.Provider == domain.ProviderKindAnthropic, "Org A must only see anthropic models")
			requireTrue(t, m.ID != "claude-2.1", "inactive model claude-2.1 must not be listed")
			requireTrue(t, m.ID != "gpt-4o" && m.ID != "o1", "OpenAI models must never appear for Org A")
		}
	})

	t.Run("Org B sees only OpenAI models, never Anthropic models", func(t *testing.T) {
		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgB,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgB}

		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("x-api-key", "aigw_orgB_key")
		rec := httptest.NewRecorder()

		g.ServeHTTP(rec, req)
		requireTrue(t, rec.Code == http.StatusOK, "want 200 OK")

		var resp struct {
			Data []struct {
				ID       string `json:"id"`
				Provider string `json:"provider"`
			} `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		requireNoErr(t, err, "unmarshal response")

		requireTrue(t, len(resp.Data) == 2, "Org B must see exactly 2 active OpenAI models")
		for _, m := range resp.Data {
			requireTrue(t, m.Provider == domain.ProviderKindOpenAI, "Org B must only see openai models")
			requireTrue(t, m.ID != "claude-3-5-sonnet", "Anthropic models must never appear for Org B")
		}
	})

	t.Run("Org C with no connected providers sees empty list (NEVER hardcoded models)", func(t *testing.T) {
		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgC,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgC}

		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("x-api-key", "aigw_orgC_key")
		rec := httptest.NewRecorder()

		g.ServeHTTP(rec, req)
		requireTrue(t, rec.Code == http.StatusOK, "want 200 OK")

		var resp struct {
			Data []any `json:"data"`
		}
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		requireNoErr(t, err, "unmarshal response")
		requireTrue(t, len(resp.Data) == 0, "Org C with no providers must receive an empty catalog")
	})

	t.Run("Org A is denied routing to OpenAI models (upstream_unavailable)", func(t *testing.T) {
		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgA,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgA}

		req := pipelineRequest("aigw_orgA_key", `{"model":"gpt-4o"}`)
		res := g.executePipeline(context.Background(), req, "req-cross-prov")

		requireTrue(t, !res.Allowed, "Org A must not be allowed to execute OpenAI model")
		requireTrue(t, res.StatusCode == http.StatusServiceUnavailable, "want 503, got %d", res.StatusCode)
		requireTrue(t, res.ErrorType == "upstream_unavailable", "want upstream_unavailable, got %s", res.ErrorType)
	})

	t.Run("Inactive model claude-2.1 is rejected as unsupported_model", func(t *testing.T) {
		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgA,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgA}

		req := pipelineRequest("aigw_orgA_key", `{"model":"claude-2.1"}`)
		res := g.executePipeline(context.Background(), req, "req-inactive-mod")

		requireTrue(t, !res.Allowed, "inactive model must be rejected")
		requireTrue(t, res.StatusCode == http.StatusBadRequest, "want 400, got %d", res.StatusCode)
		requireTrue(t, res.ErrorType == "unsupported_model", "want unsupported_model, got %s", res.ErrorType)
	})

	t.Run("Revoking provider credential immediately cuts off model listing and routing", func(t *testing.T) {
		// Revoke Org B's OpenAI credential
		tenantResolver.SetUpstream(orgB, domain.ProviderKindOpenAI, ResolvedUpstream{
			ProviderAccount: ProviderAccount{
				ID:             uuid.New(),
				OrganizationID: orgB,
				ProviderKind:   domain.ProviderKindOpenAI,
				Status:         domain.ProviderStatusActive,
			},
			Credential: ProviderCredential{
				ID:             uuid.New(),
				OrganizationID: orgB,
				Status:         domain.ProviderCredentialStatusRevoked, // Revoked!
			},
			UpstreamKey: "sk-openai-orgB",
			BaseURL:     "https://api.openai.com",
		})

		authStub.cred = GatewayCredential{
			ID:             uuid.New(),
			OrganizationID: orgB,
			Status:         "active",
			Scopes:         []string{"gateway"},
		}
		policyStub.snapshot = PolicySnapshot{OrganizationID: orgB}

		// Listing models should now be empty
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("x-api-key", "aigw_orgB_key")
		rec := httptest.NewRecorder()
		g.ServeHTTP(rec, req)

		var resp struct {
			Data []any `json:"data"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		requireTrue(t, len(resp.Data) == 0, "revoked credential must result in empty model catalog")

		// Routing request should fail with upstream_unavailable
		req2 := pipelineRequest("aigw_orgB_key", `{"model":"gpt-4o"}`)
		res := g.executePipeline(context.Background(), req2, "req-revoked-cred")
		requireTrue(t, !res.Allowed, "request on revoked provider credential must be rejected")
		requireTrue(t, res.StatusCode == http.StatusServiceUnavailable, "want 503, got %d", res.StatusCode)
		requireTrue(t, res.ErrorType == "upstream_unavailable", "want upstream_unavailable, got %s", res.ErrorType)
	})
}
