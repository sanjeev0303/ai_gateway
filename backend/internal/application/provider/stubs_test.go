package provider

// White-box unit-test scaffolding for the provider service: stub repos,
// a stub secret store and a stub provider client wired against a fixed
// clock. Stubs embed their port interface so tests preregister only the
// methods a path exercises; an unregistered call panics, turning an
// unexpected interaction into a loud failure.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"
)

var (
	testUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOrgID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testAccountID = uuid.MustParse("00000000-0000-0000-0000-00000000000a")
	testCredID    = uuid.MustParse("00000000-0000-0000-0000-00000000000b")
	testOtherID   = uuid.MustParse("00000000-0000-0000-0000-00000000000c")
	testNow       = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
)

func notConfigured(method string) {
	panic("provider test: " + method + " called without a stub")
}

// testStubs bundles every stub a Service depends on.
type testStubs struct {
	accounts    *stubProviderAccounts
	credentials *stubProviderCredentials
	models      *stubModels
	health      *stubProviderHealth
	audits      *stubAudits
	secrets     *stubSecrets
	provider    *stubProvider
}

// newTestService wires a Service against the stubs with a fixed clock. The
// Anthropic client (the only M4 provider) is registered by default.
func newTestService() (*Service, *testStubs) {
	st := &testStubs{
		accounts:    &stubProviderAccounts{},
		credentials: &stubProviderCredentials{},
		models:      &stubModels{},
		health:      &stubProviderHealth{},
		audits:      &stubAudits{},
		secrets:     &stubSecrets{values: map[string]string{}},
		provider:    &stubProvider{},
	}
	svc := NewService(ServiceDeps{
		Accounts:    st.accounts,
		Credentials: st.credentials,
		Models:      st.models,
		Health:      st.health,
		Audits:      st.audits,
		Secrets:     st.secrets,
		Providers: map[string]providerclient.Provider{
			domain.ProviderKindAnthropic: st.provider,
		},
		Now: func() time.Time { return testNow },
	})
	return svc, st
}

// userActor builds a user-kind actor for the fixed test user and org.
func userActor(role string) domain.Actor {
	return domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         testUserID,
		OrganizationID: testOrgID,
		Role:           role,
		DisplayName:    "Test User",
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireErrIs(t *testing.T, err, want error) {
	t.Helper()
	if err == nil || !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func requireEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func requireAudit(t *testing.T, st *testStubs, action string) {
	t.Helper()
	for _, a := range st.audits.actions {
		if a == action {
			return
		}
	}
	t.Fatalf("audit actions = %v, missing %q", st.audits.actions, action)
}

type stubProviderAccounts struct {
	repositories.ProviderAccountRepository
	create func(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error)
	list   func(context.Context, uuid.UUID) ([]domain.ProviderAccount, error)
}

func (s *stubProviderAccounts) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProviderAccount) (domain.ProviderAccount, error) {
	if s.create == nil {
		notConfigured("stubProviderAccounts.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubProviderAccounts) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
	if s.get == nil {
		notConfigured("stubProviderAccounts.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubProviderAccounts) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
	if s.list == nil {
		notConfigured("stubProviderAccounts.List")
	}
	return s.list(ctx, orgID)
}

type stubProviderCredentials struct {
	repositories.ProviderCredentialRepository
	create        func(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error)
	get           func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error)
	list          func(context.Context, uuid.UUID) ([]domain.ProviderCredential, error)
	listByAccount func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error)
	revoke        func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error)
}

func (s *stubProviderCredentials) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
	if s.create == nil {
		notConfigured("stubProviderCredentials.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubProviderCredentials) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	if s.get == nil {
		notConfigured("stubProviderCredentials.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubProviderCredentials) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
	if s.list == nil {
		notConfigured("stubProviderCredentials.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubProviderCredentials) ListByAccount(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
	if s.listByAccount == nil {
		notConfigured("stubProviderCredentials.ListByAccount")
	}
	return s.listByAccount(ctx, orgID, accountID)
}

func (s *stubProviderCredentials) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	if s.revoke == nil {
		notConfigured("stubProviderCredentials.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

type stubModels struct {
	repositories.ModelRepository
	list func(context.Context) ([]domain.Model, error)
}

func (s *stubModels) List(ctx context.Context) ([]domain.Model, error) {
	if s.list == nil {
		notConfigured("stubModels.List")
	}
	return s.list(ctx)
}

type stubProviderHealth struct {
	repositories.ProviderHealthRepository
	get    func(context.Context, string) (domain.ProviderHealthState, error)
	list   func(context.Context) ([]domain.ProviderHealthState, error)
	upsert func(context.Context, domain.ProviderHealthState) error
	states []domain.ProviderHealthState
}

func (s *stubProviderHealth) Get(ctx context.Context, providerKind string) (domain.ProviderHealthState, error) {
	if s.get == nil {
		notConfigured("stubProviderHealth.Get")
	}
	return s.get(ctx, providerKind)
}

func (s *stubProviderHealth) List(ctx context.Context) ([]domain.ProviderHealthState, error) {
	if s.list != nil {
		return s.list(ctx)
	}
	return append([]domain.ProviderHealthState(nil), s.states...), nil
}

func (s *stubProviderHealth) Upsert(ctx context.Context, state domain.ProviderHealthState) error {
	if s.upsert != nil {
		return s.upsert(ctx, state)
	}
	s.states = append(s.states, state)
	return nil
}

// stubAudits records every append and defaults to success.
type stubAudits struct {
	repositories.AuditLogRepository
	appendFn func(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error)
	actions  []string
}

func (s *stubAudits) Append(ctx context.Context, orgID uuid.UUID, in domain.NewAuditLog) (domain.AuditLog, error) {
	if s.appendFn != nil {
		return s.appendFn(ctx, orgID, in)
	}
	s.actions = append(s.actions, in.Action)
	return domain.AuditLog{}, nil
}

// stubSecrets is an in-memory secrets store. Deleting a missing ref is a
// no-op by contract; NotFound is returned only by Get on a missing ref.
type stubSecrets struct {
	secrets.Store
	values map[string]string
}

func (s *stubSecrets) Get(_ context.Context, ref string) (string, error) {
	v, ok := s.values[ref]
	if !ok {
		return "", secrets.ErrNotFound
	}
	return v, nil
}

func (s *stubSecrets) Put(_ context.Context, ref, value string) error {
	s.values[ref] = value
	return nil
}

func (s *stubSecrets) Delete(_ context.Context, ref string) error {
	delete(s.values, ref)
	return nil
}

// stubProvider is an in-memory provider client; ValidateCredentials and
// HealthCheck pass by default so tests opt into failures.
type stubProvider struct {
	providerclient.Provider
	validateErr error
	healthErr   error
	listModels  func(context.Context, string) ([]providerclient.Model, error)
}

func (s *stubProvider) ValidateCredentials(_ context.Context, _ string) error {
	return s.validateErr
}

func (s *stubProvider) HealthCheck(_ context.Context, _ string) error {
	return s.healthErr
}

func (s *stubProvider) ListModels(ctx context.Context, apiKey string) ([]providerclient.Model, error) {
	if s.listModels == nil {
		notConfigured("stubProvider.ListModels")
	}
	return s.listModels(ctx, apiKey)
}

// Compile-time proof that the stubs satisfy their ports.
var (
	_ repositories.ProviderAccountRepository    = (*stubProviderAccounts)(nil)
	_ repositories.ProviderCredentialRepository = (*stubProviderCredentials)(nil)
	_ repositories.ModelRepository              = (*stubModels)(nil)
	_ repositories.ProviderHealthRepository     = (*stubProviderHealth)(nil)
	_ repositories.AuditLogRepository           = (*stubAudits)(nil)
	_ secrets.Store                             = (*stubSecrets)(nil)
	_ providerclient.Provider                   = (*stubProvider)(nil)
)