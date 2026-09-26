// Contract tests for tenant (upstream) resolution: DB fallback, caching,
// provider/credential status gating and secret lookup.
package gatewayv2

import (
	"context"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/secrets"

	"github.com/google/uuid"
)

const testUpstreamKey = "sk-ant-test-secret"

func makeTenantResolver(t *testing.T, accounts []domain.ProviderAccount, creds []domain.ProviderCredential, store secrets.Store) *TenantResolverImpl {
	t.Helper()
	return NewTenantResolver(
		&stubAccountRepo{listFn: func(context.Context, uuid.UUID) ([]domain.ProviderAccount, error) {
			return accounts, nil
		}},
		&stubCredentialRepo{listByAccountFn: func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error) {
			return creds, nil
		}},
		store,
	)
}

func activeAnthropicAccount() domain.ProviderAccount {
	return domain.ProviderAccount{
		ID:             mustUUIDs[1],
		OrganizationID: mustUUIDs[0],
		ProviderKind:   domain.ProviderKindAnthropic,
		Name:           "primary",
		Status:         domain.ProviderStatusActive,
	}
}

func activeCredential() domain.ProviderCredential {
	return domain.ProviderCredential{
		ID:                mustUUIDs[2],
		OrganizationID:    mustUUIDs[0],
		ProviderAccountID: mustUUIDs[1],
		SecretRef:         "ref-main",
		Status:            domain.ProviderCredentialStatusActive,
	}
}

func TestTenantResolver_HappyPath(t *testing.T) {
	store := &stubSecretStore{values: map[string]string{"ref-main": testUpstreamKey}}
	resolver := makeTenantResolver(t, []domain.ProviderAccount{activeAnthropicAccount()}, []domain.ProviderCredential{activeCredential()}, store)

	upstream, err := resolver.ResolveUpstream(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	requireTrue(t, upstream.ProviderAccount.ID == mustUUIDs[1], "expected account copied into resolved upstream")
	requireTrue(t, upstream.Credential.ID == mustUUIDs[2], "expected credential copied into resolved upstream")
	if upstream.UpstreamKey != testUpstreamKey {
		t.Fatalf("expected secret value as upstream key, got %q", upstream.UpstreamKey)
	}
	if upstream.BaseURL != "https://api.anthropic.com" {
		t.Fatalf("expected default anthropic base URL, got %q", upstream.BaseURL)
	}
}

func TestTenantResolver_CustomBaseURL(t *testing.T) {
	account := activeAnthropicAccount()
	account.BaseURL = "https://eu.api.anthropic.com"
	store := &stubSecretStore{values: map[string]string{"ref-main": testUpstreamKey}}
	resolver := makeTenantResolver(t, []domain.ProviderAccount{account}, []domain.ProviderCredential{activeCredential()}, store)

	upstream, err := resolver.ResolveUpstream(context.Background(), mustUUIDs[0])
	requireNoErr(t, err)

	if upstream.BaseURL != "https://eu.api.anthropic.com" {
		t.Fatalf("expected account base URL override, got %q", upstream.BaseURL)
	}
}

func TestTenantResolver_CachesAndInvalidates(t *testing.T) {
	store := &stubSecretStore{values: map[string]string{"ref-main": testUpstreamKey}}
	accounts := []domain.ProviderAccount{activeAnthropicAccount()}
	creds := []domain.ProviderCredential{activeCredential()}

	listCalls := 0
	tr := NewTenantResolver(
		&stubAccountRepo{listFn: func(context.Context, uuid.UUID) ([]domain.ProviderAccount, error) {
			listCalls++
			return accounts, nil
		}},
		&stubCredentialRepo{listByAccountFn: func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error) {
			return creds, nil
		}},
		store,
	)

	ctx := context.Background()
	orgID := mustUUIDs[0]

	first, err := tr.ResolveUpstream(ctx, orgID)
	requireNoErr(t, err)
	second, err := tr.ResolveUpstream(ctx, orgID)
	requireNoErr(t, err)

	if listCalls != 1 {
		t.Fatalf("expected cache hit on second resolve (1 DB call), got %d calls", listCalls)
	}
	if first.UpstreamKey != second.UpstreamKey {
		t.Fatal("expected identical resolved upstream on cache hit")
	}

	tr.cache.Invalidate(orgID)
	if _, err := tr.ResolveUpstream(ctx, orgID); err != nil {
		t.Fatalf("expected re-resolve after invalidate: %v", err)
	}
	if listCalls != 2 {
		t.Fatalf("expected DB re-hit after invalidate (2 calls), got %d", listCalls)
	}
}

func TestTenantResolver_NoActiveAnthropicAccount(t *testing.T) {
	cases := []struct {
		name     string
		accounts []domain.ProviderAccount
	}{
		{"empty", nil},
		{"openai_only", []domain.ProviderAccount{{ID: mustUUIDs[1], ProviderKind: domain.ProviderKindOpenAI, Status: domain.ProviderStatusActive}}},
		{"disabled_anthropic", []domain.ProviderAccount{{ID: mustUUIDs[1], ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderStatusDisabled}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &stubSecretStore{values: map[string]string{}}
			resolver := makeTenantResolver(t, tc.accounts, nil, store)

			_, err := resolver.ResolveUpstream(context.Background(), mustUUIDs[0])
			requireErrIs(t, err, ErrNoUpstream)
		})
	}
}

func TestTenantResolver_NoUsableCredentials(t *testing.T) {
	cases := []struct {
		name  string
		creds []domain.ProviderCredential
	}{
		{"empty", nil},
		{"revoked", []domain.ProviderCredential{{ID: mustUUIDs[2], ProviderAccountID: mustUUIDs[1], SecretRef: "ref-main", Status: domain.ProviderCredentialStatusRevoked}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &stubSecretStore{values: map[string]string{"ref-main": testUpstreamKey}}
			resolver := makeTenantResolver(t, []domain.ProviderAccount{activeAnthropicAccount()}, tc.creds, store)

			_, err := resolver.ResolveUpstream(context.Background(), mustUUIDs[0])
			requireErrIs(t, err, ErrNoUpstream)
		})
	}
}

func TestTenantResolver_MissingSecret(t *testing.T) {
	store := &stubSecretStore{values: map[string]string{}}
	resolver := makeTenantResolver(t, []domain.ProviderAccount{activeAnthropicAccount()}, []domain.ProviderCredential{activeCredential()}, store)

	_, err := resolver.ResolveUpstream(context.Background(), mustUUIDs[0])
	requireErrIs(t, err, ErrNoUpstream)
}

func TestTenantResolver_RepoErrorPropagates(t *testing.T) {
	tr := NewTenantResolver(
		&stubAccountRepo{listFn: func(context.Context, uuid.UUID) ([]domain.ProviderAccount, error) {
			return nil, errStub
		}},
		&stubCredentialRepo{listByAccountFn: func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error) {
			return nil, nil
		}},
		&stubSecretStore{values: map[string]string{}},
	)

	_, err := tr.ResolveUpstream(context.Background(), mustUUIDs[0])
	requireErrIs(t, err, errStub)
}

func TestUpstreamCache_TTLExpiry(t *testing.T) {
	now := time.Now()
	cache := NewUpstreamCache(10, time.Minute)
	cache.clock = func() time.Time { return now }

	cache.Set(mustUUIDs[0], ResolvedUpstream{UpstreamKey: testUpstreamKey, BaseURL: "https://api.anthropic.com"})

	if got := cache.Get(mustUUIDs[0]); got == nil {
		t.Fatal("expected cache hit before TTL")
	}

	now = now.Add(61 * time.Second)
	if got := cache.Get(mustUUIDs[0]); got != nil {
		t.Fatal("expected cache miss after TTL")
	}
}

func TestUpstreamCache_EvictsWhenFull(t *testing.T) {
	cache := NewUpstreamCache(1, time.Minute)
	first := mustUUIDs[0]
	second := mustUUIDs[1]

	cache.Set(first, ResolvedUpstream{UpstreamKey: "a"})
	cache.Set(second, ResolvedUpstream{UpstreamKey: "b"})

	requireTrue(t, cache.Get(second) != nil, "expected most recent entry to survive eviction")
	evicted := cache.Get(first)
	if evicted != nil {
		t.Fatal("expected oldest entry to be evicted when the cache is full")
	}
}

func TestUpstreamCache_Invalidate(t *testing.T) {
	cache := NewUpstreamCache(10, time.Minute)
	cache.Set(mustUUIDs[0], ResolvedUpstream{UpstreamKey: testUpstreamKey})
	cache.Invalidate(mustUUIDs[0])

	if got := cache.Get(mustUUIDs[0]); got != nil {
		t.Fatal("expected nil after invalidate")
	}
}

func TestUpstreamCache_GetUnknownKey(t *testing.T) {
	cache := NewUpstreamCache(10, time.Minute)
	if got := cache.Get(mustUUIDs[2]); got != nil {
		t.Fatal("expected nil for unknown key")
	}
}