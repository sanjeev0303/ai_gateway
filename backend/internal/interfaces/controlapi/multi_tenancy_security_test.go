package controlapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

var (
	orgTenantA = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	orgTenantB = uuid.MustParse("22222222-2222-2222-2222-222222222222")

	userTenantA = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	userTenantB = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	projectTenantB = uuid.MustParse("99999999-9999-9999-9999-999999999999")
	teamTenantB    = uuid.MustParse("88888888-8888-8888-8888-888888888888")
	saTenantB      = uuid.MustParse("77777777-7777-7777-7777-777777777777")
	credTenantB    = uuid.MustParse("66666666-6666-6666-6666-666666666666")
	provAcctTenantB = uuid.MustParse("55555555-5555-5555-5555-555555555555")
)

func tenantBearer(st *controlStubs, orgID, userID uuid.UUID, role string) map[string]string {
	st.verifier.verify = func(_ context.Context, _ string) (domain.TokenClaims, error) {
		return domain.TokenClaims{
			Subject:        userID.String(),
			DisplayName:    "Tenant User",
			OrganizationID: orgID,
			Role:           role,
			Kind:           domain.TokenKindUser,
			IssuedAt:       testNow,
			ExpiresAt:      testNow.Add(time.Hour),
		}, nil
	}
	return map[string]string{"Authorization": "Bearer tenant-token"}
}

// TestMultiTenantSecurity_CrossTenantReadBlocked verifies that a user from Tenant A
// cannot read or enumerate resources owned by Tenant B.
func TestMultiTenantSecurity_CrossTenantReadBlocked(t *testing.T) {
	srv, st := newTestServer(t)

	// Configure stubs so Tenant B resources exist in the repository, but reject
	// lookups that pass Tenant A's organization ID.
	st.projects.get = func(_ context.Context, orgID, projID uuid.UUID) (domain.Project, error) {
		if orgID != orgTenantB && projID == projectTenantB {
			return domain.Project{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && projID == projectTenantB {
			return domain.Project{ID: projectTenantB, OrganizationID: orgTenantB, Name: "Secret Project B"}, nil
		}
		return domain.Project{}, repositories.ErrNotFound
	}

	st.teams.get = func(_ context.Context, orgID, teamID uuid.UUID) (domain.Team, error) {
		if orgID != orgTenantB && teamID == teamTenantB {
			return domain.Team{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && teamID == teamTenantB {
			return domain.Team{ID: teamTenantB, OrganizationID: orgTenantB, Name: "Team B"}, nil
		}
		return domain.Team{}, repositories.ErrNotFound
	}

	st.gatewayCredentials.get = func(_ context.Context, orgID, credID uuid.UUID) (domain.GatewayCredential, error) {
		if orgID != orgTenantB && credID == credTenantB {
			return domain.GatewayCredential{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && credID == credTenantB {
			return domain.GatewayCredential{ID: credTenantB, OrganizationID: orgTenantB, Name: "GW Cred B"}, nil
		}
		return domain.GatewayCredential{}, repositories.ErrNotFound
	}

	st.providerAccounts.get = func(_ context.Context, orgID, acctID uuid.UUID) (domain.ProviderAccount, error) {
		if orgID != orgTenantB && acctID == provAcctTenantB {
			return domain.ProviderAccount{}, repositories.ErrNotFound
		}
		if orgID == orgTenantB && acctID == provAcctTenantB {
			return domain.ProviderAccount{ID: provAcctTenantB, OrganizationID: orgTenantB, Name: "Provider B"}, nil
		}
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA, domain.RoleOwner)

	tests := []struct {
		name string
		path string
	}{
		{"CrossTenantProject", "/v1/projects/" + projectTenantB.String()},
		{"CrossTenantTeam", "/v1/teams/" + teamTenantB.String()},
		{"CrossTenantGatewayCredential", "/v1/gateway-credentials/" + credTenantB.String()},
		{"CrossTenantProviderAccount", "/v1/provider-accounts/" + provAcctTenantB.String()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doJSON(t, srv, http.MethodGet, tt.path, nil, hdrA)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusNotFound {
				t.Errorf("expected 404 Not Found for cross-tenant resource, got %d", resp.StatusCode)
			}
		})
	}
}

// TestMultiTenantSecurity_CrossTenantDeleteBlocked verifies that Tenant A cannot delete Tenant B resources.
func TestMultiTenantSecurity_CrossTenantDeleteBlocked(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.revoke = func(_ context.Context, orgID, saID uuid.UUID) (domain.ServiceAccount, error) {
		if orgID != orgTenantB {
			return domain.ServiceAccount{}, repositories.ErrNotFound
		}
		return domain.ServiceAccount{ID: saID, Status: domain.ServiceAccountStatusRevoked}, nil
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA, domain.RoleOwner)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/service-accounts/"+saTenantB.String(), nil, hdrA)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found for cross-tenant deletion, got %d", resp.StatusCode)
	}
}

// TestMultiTenantSecurity_SecretScrubbing ensures responses never leak raw API keys,
// secret references, password hashes, or full tokens in listing responses.
func TestMultiTenantSecurity_SecretScrubbing(t *testing.T) {
	srv, st := newTestServer(t)

	st.gatewayCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
		return []domain.GatewayCredential{
			{
				ID:             testOtherID,
				OrganizationID: orgTenantA,
				Name:           "gateway-key",
				KeyPrefix:      "aigw_live_test",
				KeyHash:        "super-secret-key-hash-never-expose",
				Scopes:         []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
				Status:         domain.GatewayCredentialStatusActive,
				CreatedAt:      testNow,
				UpdatedAt:      testNow,
			},
		}, nil
	}

	st.providerCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{
				ID:                testProvCred,
				OrganizationID:    orgTenantA,
				ProviderAccountID: testProvAcct,
				Name:              "Anthropic Key",
				Kind:              domain.CredentialKindAPIKey,
				SecretRef:         "vault/secrets/anthropic-raw-key",
				Fingerprint:       "sha256:12345678",
				Tail:              "1234",
				Status:            domain.ProviderCredentialStatusActive,
				CreatedAt:         testNow,
				UpdatedAt:         testNow,
			},
		}, nil
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA, domain.RoleOwner)

	t.Run("GatewayCredentialsScrubbed", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/gateway-credentials", nil, hdrA)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		rawBytes, _ := json.Marshal(body)
		rawStr := string(rawBytes)

		if containsSubstring(rawStr, "super-secret-key-hash-never-expose") {
			t.Errorf("leaked secret key hash in response: %s", rawStr)
		}
	})

	t.Run("ProviderCredentialsScrubbed", func(t *testing.T) {
		resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials", nil, hdrA)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Decode: %v", err)
		}

		rawBytes, _ := json.Marshal(body)
		rawStr := string(rawBytes)

		if containsSubstring(rawStr, "vault/secrets/anthropic-raw-key") {
			t.Errorf("leaked provider SecretRef in response: %s", rawStr)
		}
	})
}

// TestMultiTenantSecurity_RevokedCredentialImmediateRejection tests that revoking a credential
// sets status to revoked and returns 204 No Content.
func TestMultiTenantSecurity_RevokedCredentialImmediateRejection(t *testing.T) {
	srv, st := newTestServer(t)

	revoked := false
	st.gatewayCredentials.revoke = func(_ context.Context, orgID, credID uuid.UUID) (domain.GatewayCredential, error) {
		revoked = true
		return domain.GatewayCredential{
			ID:             credID,
			OrganizationID: orgID,
			Status:         domain.GatewayCredentialStatusRevoked,
			UpdatedAt:      time.Now(),
		}, nil
	}

	hdrA := tenantBearer(st, orgTenantA, userTenantA, domain.RoleOwner)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/gateway-credentials/"+testOtherID.String(), nil, hdrA)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", resp.StatusCode)
	}

	if !revoked {
		t.Fatal("expected credential to be revoked in repository")
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && len(sub) > 0 && stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
