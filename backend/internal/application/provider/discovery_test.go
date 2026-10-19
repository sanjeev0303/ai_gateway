package provider

import (
	"context"
	"errors"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

// TestDynamicModelDiscovery verifies capabilities discovery, metadata persistence,
// and stale model deprecation during SyncModels.
func TestDynamicModelDiscovery(t *testing.T) {
	ctx := context.Background()

	t.Run("discovers capabilities and marks stale models inactive", func(t *testing.T) {
		svc, st := newTestService()

		// Provider advertises 2 models
		st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
			return []providerclient.Model{
				{
					ID:           "claude-3-5-sonnet",
					DisplayName:  "Claude 3.5 Sonnet",
					Capabilities: []string{"text_generation", "streaming", "tool_calling"},
					ContextLimit: 200000,
				},
				{
					ID:           "claude-3-opus",
					DisplayName:  "Claude 3 Opus",
					Capabilities: []string{"text_generation", "streaming"},
					ContextLimit: 200000,
				},
			}, nil
		}

		var upsertedModels []domain.Model
		st.models.upsertWithMetadata = func(_ context.Context, m domain.Model) (domain.Model, error) {
			upsertedModels = append(upsertedModels, m)
			return m, nil
		}

		var markedInactiveProvider string
		var markedInactiveSlugs []string
		st.models.markInactiveExcept = func(_ context.Context, kind string, activeSlugs []string) error {
			markedInactiveProvider = kind
			markedInactiveSlugs = activeSlugs
			return nil
		}

		synced, err := svc.SyncModels(ctx, userActor(domain.RoleOwner), domain.ProviderKindAnthropic, "sk-ant-test")
		requireNoErr(t, err)
		requireEqual(t, len(synced), 2)

		// Verify upserted metadata
		requireEqual(t, len(upsertedModels), 2)
		requireEqual(t, upsertedModels[0].ModelSlug, "claude-3-5-sonnet")
		requireEqual(t, upsertedModels[0].ContextLimit, 200000)
		requireEqual(t, upsertedModels[0].SupportsStreaming, true)
		requireEqual(t, upsertedModels[0].SupportsTools, true)
		requireEqual(t, upsertedModels[0].Status, "active")
		requireEqual(t, upsertedModels[0].IsActive, true)

		// Verify stale cleanup call
		requireEqual(t, markedInactiveProvider, domain.ProviderKindAnthropic)
		requireEqual(t, len(markedInactiveSlugs), 2)
		requireEqual(t, markedInactiveSlugs[0], "claude-3-5-sonnet")
		requireEqual(t, markedInactiveSlugs[1], "claude-3-opus")
	})

	t.Run("mock provider populates rich metadata across providers", func(t *testing.T) {
		svc, st := newTestService()
		svc.providers[domain.ProviderKindOpenAI] = st.provider

		st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
			return []providerclient.Model{
				{ID: "gpt-4o", DisplayName: "GPT-4o", Capabilities: []string{"text_generation", "streaming", "tool_calling", "multimodal"}, ContextLimit: 128000},
				{ID: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Capabilities: []string{"text_generation", "streaming", "tool_calling"}, ContextLimit: 128000},
				{ID: "o1", DisplayName: "o1", Capabilities: []string{"text_generation", "reasoning"}, ContextLimit: 200000},
			}, nil
		}

		var upserted []domain.Model
		st.models.upsertWithMetadata = func(_ context.Context, m domain.Model) (domain.Model, error) {
			upserted = append(upserted, m)
			return m, nil
		}

		synced, err := svc.SyncModels(ctx, userActor(domain.RoleOwner), domain.ProviderKindOpenAI, "sk-test-key")
		requireNoErr(t, err)
		requireTrue(t, len(synced) >= 3, "expected at least 3 openai models")

		// Verify reasoning capability on o1
		var foundO1 bool
		for _, m := range upserted {
			if m.ModelSlug == "o1" {
				foundO1 = true
				requireEqual(t, m.SupportsReasoning, true)
			}
		}
		requireTrue(t, foundO1, "expected o1 model with reasoning support")
	})

	t.Run("SyncAccountModels loads stored credential and syncs", func(t *testing.T) {
		svc, st := newTestService()
		accID := uuid.New()
		credID := uuid.New()

		st.accounts.get = func(_ context.Context, _ uuid.UUID, id uuid.UUID) (domain.ProviderAccount, error) {
			return domain.ProviderAccount{
				ID:             accID,
				OrganizationID: testOrgID,
				ProviderKind:   domain.ProviderKindAnthropic,
				Status:         domain.ProviderStatusActive,
			}, nil
		}

		st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
			return []domain.ProviderCredential{
				{
					ID:                credID,
					ProviderAccountID: accID,
					SecretRef:         "ref:secret-1",
					Status:            domain.ProviderCredentialStatusActive,
				},
			}, nil
		}
		st.secrets.values["ref:secret-1"] = "sk-ant-valid-secret"

		st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
			requireEqual(t, apiKey, "sk-ant-valid-secret")
			return []providerclient.Model{
				{ID: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet"},
			}, nil
		}

		synced, err := svc.SyncAccountModels(ctx, userActor(domain.RoleOwner), accID)
		requireNoErr(t, err)
		requireEqual(t, len(synced), 1)
		requireEqual(t, synced[0].ModelSlug, "claude-3-5-sonnet")
	})
}

// ---- Comprehensive Phase B Regression Tests ---------------------------------

func TestDiscovery_Successful(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()
	credID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{
				ID:                credID,
				ProviderAccountID: accID,
				SecretRef:         "ref:secret-live",
				Status:            domain.ProviderCredentialStatusActive,
			},
		}, nil
	}
	st.secrets.values["ref:secret-live"] = "sk-ant-live-token"

	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		requireEqual(t, apiKey, "sk-ant-live-token")
		return []providerclient.Model{
			{ID: "claude-3-7-sonnet-20250219", DisplayName: "Claude 3.7 Sonnet", ContextLimit: 200000, Capabilities: []string{"streaming", "tool_calling"}},
			{ID: "claude-3-5-haiku-20241022", DisplayName: "Claude 3.5 Haiku", ContextLimit: 200000, Capabilities: []string{"streaming"}},
		}, nil
	}

	var upserted []domain.Model
	st.models.upsertWithMetadata = func(_ context.Context, m domain.Model) (domain.Model, error) {
		upserted = append(upserted, m)
		return m, nil
	}

	synced, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireNoErr(t, err)
	requireEqual(t, len(synced), 2)
	requireEqual(t, synced[0].ModelSlug, "claude-3-7-sonnet-20250219")
	requireEqual(t, synced[1].ModelSlug, "claude-3-5-haiku-20241022")

	// Verify discovery status is recorded in health table
	health, err := st.health.Get(ctx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	requireEqual(t, health.Status, domain.ProviderHealthHealthy)
	requireEqual(t, health.StatusReason, "discovery_succeeded")
}

func TestDiscovery_EmptyModelLists(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), ProviderAccountID: accID, SecretRef: "ref:secret-empty", Status: domain.ProviderCredentialStatusActive},
		}, nil
	}
	st.secrets.values["ref:secret-empty"] = "sk-ant-empty-token"

	// Provider returns 0 models
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return []providerclient.Model{}, nil
	}

	var markedInactiveProvider string
	var markedInactiveSlugs []string
	st.models.markInactiveExcept = func(_ context.Context, kind string, slugs []string) error {
		markedInactiveProvider = kind
		markedInactiveSlugs = slugs
		return nil
	}

	synced, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireNoErr(t, err)
	// Must return 0 models — NO hardcoded fallback models!
	requireEqual(t, len(synced), 0)
	// MarkInactiveExcept must be called with empty list to retire existing models
	requireEqual(t, markedInactiveProvider, domain.ProviderKindAnthropic)
	requireEqual(t, len(markedInactiveSlugs), 0)
}

func TestDiscovery_InvalidCredentials(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), ProviderAccountID: accID, SecretRef: "ref:secret-bad", Status: domain.ProviderCredentialStatusActive},
		}, nil
	}
	st.secrets.values["ref:secret-bad"] = "sk-ant-invalid-token"

	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return nil, providerclient.ErrInvalidCredentials
	}

	_, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireTrue(t, errors.Is(err, ErrInvalidProviderCredential), "expected ErrInvalidProviderCredential")

	// Verify health recorded as credentials_rejected
	health, err := st.health.Get(ctx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	requireEqual(t, health.Status, domain.ProviderHealthUnhealthy)
	requireEqual(t, health.StatusReason, "credentials_rejected")
}

func TestDiscovery_ProviderTimeouts(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), ProviderAccountID: accID, SecretRef: "ref:secret-timeout", Status: domain.ProviderCredentialStatusActive},
		}, nil
	}
	st.secrets.values["ref:secret-timeout"] = "sk-ant-timeout-token"

	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return nil, context.DeadlineExceeded
	}

	_, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireTrue(t, errors.Is(err, context.DeadlineExceeded), "expected timeout error")

	// Verify health recorded as discovery_failed
	health, err := st.health.Get(ctx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	requireEqual(t, health.Status, domain.ProviderHealthUnhealthy)
	requireEqual(t, health.StatusReason, "discovery_failed")
}

func TestDiscovery_UnsupportedDiscoveryEndpoints(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), ProviderAccountID: accID, SecretRef: "ref:secret-404", Status: domain.ProviderCredentialStatusActive},
		}, nil
	}
	st.secrets.values["ref:secret-404"] = "sk-ant-endpoint-404"

	// Provider returns 404 / unsupported discovery
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return nil, providerclient.ErrDiscoveryUnavailable
	}

	_, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireTrue(t, errors.Is(err, ErrDiscoveryUnavailable), "expected ErrDiscoveryUnavailable")

	health, err := st.health.Get(ctx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	requireEqual(t, health.Status, domain.ProviderHealthUnhealthy)
	requireEqual(t, health.StatusReason, "discovery_unavailable")
}

func TestDiscovery_RevokedCredentials(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusActive,
		}, nil
	}
	// Account only has revoked credentials
	st.credentials.listByAccount = func(_ context.Context, _ uuid.UUID, id uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), ProviderAccountID: accID, SecretRef: "ref:revoked", Status: domain.ProviderCredentialStatusRevoked},
		}, nil
	}

	var upstreamCalled bool
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		upstreamCalled = true
		return []providerclient.Model{}, nil
	}

	_, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireTrue(t, errors.Is(err, ErrRevokedCredential), "expected ErrRevokedCredential")
	requireTrue(t, !upstreamCalled, "upstream must not be called when credential is revoked")
}

func TestDiscovery_DisabledAccount(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accID := uuid.New()

	// Provider account is disabled
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{
			ID:             accID,
			OrganizationID: orgID,
			ProviderKind:   domain.ProviderKindAnthropic,
			Status:         domain.ProviderStatusDisabled,
		}, nil
	}

	var upstreamCalled bool
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		upstreamCalled = true
		return []providerclient.Model{}, nil
	}

	_, err := svc.SyncAccountModels(ctx, userActor(domain.RoleAdmin), accID)
	requireTrue(t, errors.Is(err, ErrDisabledAccount), "expected ErrDisabledAccount")
	requireTrue(t, !upstreamCalled, "upstream must not be called when account is disabled")
}

func TestDiscovery_OrganizationIsolation(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	accB := uuid.New()
	orgB := uuid.New()

	// Account belongs to Org B
	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		if orgID != orgB {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}
		return domain.ProviderAccount{ID: accB, OrganizationID: orgB, ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderStatusActive}, nil
	}

	// Actor from Org A attempts to sync Org B's account
	actorA := userActor(domain.RoleAdmin)
	actorA.OrganizationID = uuid.New() // Org A != Org B

	_, err := svc.SyncAccountModels(ctx, actorA, accB)
	requireTrue(t, errors.Is(err, repositories.ErrNotFound), "expected ErrNotFound for cross-tenant discovery")
}

func TestDiscovery_StaleModels(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()

	// Live provider only reports claude-3-5-sonnet
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return []providerclient.Model{
			{ID: "claude-3-5-sonnet", DisplayName: "Claude 3.5 Sonnet"},
		}, nil
	}

	var markedInactiveProvider string
	var markedInactiveSlugs []string
	st.models.markInactiveExcept = func(_ context.Context, kind string, slugs []string) error {
		markedInactiveProvider = kind
		markedInactiveSlugs = slugs
		return nil
	}

	synced, err := svc.SyncModels(ctx, userActor(domain.RoleAdmin), domain.ProviderKindAnthropic, "sk-ant-live")
	requireNoErr(t, err)
	requireEqual(t, len(synced), 1)

	// Verify that MarkInactiveExcept was called with only active models, marking older models stale
	requireEqual(t, markedInactiveProvider, domain.ProviderKindAnthropic)
	requireEqual(t, len(markedInactiveSlugs), 1)
	requireEqual(t, markedInactiveSlugs[0], "claude-3-5-sonnet")
}

func TestDiscovery_CatalogSuggestionNotTreatedAsConfirmedAvailable(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	svc.providers[domain.ProviderKindOpenAI] = st.provider

	// Provider credential only has access to a specific small model
	st.provider.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		return []providerclient.Model{
			{ID: "gpt-3.5-turbo", DisplayName: "GPT-3.5 Turbo"},
		}, nil
	}

	var retiredSlugs []string
	st.models.markInactiveExcept = func(_ context.Context, kind string, activeSlugs []string) error {
		retiredSlugs = activeSlugs
		return nil
	}

	synced, err := svc.SyncModels(ctx, userActor(domain.RoleAdmin), domain.ProviderKindOpenAI, "sk-test-limited-key")
	requireNoErr(t, err)
	requireEqual(t, len(synced), 1)
	requireEqual(t, synced[0].ModelSlug, "gpt-3.5-turbo")

	// Verify that catalog models (like gpt-4o) not returned by this credential are not in activeSlugs
	requireEqual(t, len(retiredSlugs), 1)
	requireEqual(t, retiredSlugs[0], "gpt-3.5-turbo")
}
