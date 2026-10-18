package provider

import (
	"context"
	"testing"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
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

	t.Run("dummy key populates rich metadata across providers", func(t *testing.T) {
		svc, st := newTestService()
		svc.providers[domain.ProviderKindOpenAI] = st.provider

		var upserted []domain.Model
		st.models.upsertWithMetadata = func(_ context.Context, m domain.Model) (domain.Model, error) {
			upserted = append(upserted, m)
			return m, nil
		}

		synced, err := svc.SyncModels(ctx, userActor(domain.RoleOwner), domain.ProviderKindOpenAI, "sk-dummy-key")
		requireNoErr(t, err)
		requireTrue(t, len(synced) >= 3, "expected at least 3 openai dummy models")

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
