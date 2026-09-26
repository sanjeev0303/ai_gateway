package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- provider accounts ----------------------------------------------------

func TestCreateProviderAccount(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderAccount) (domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, in.ProviderKind, domain.ProviderKindAnthropic)
		requireEqual(t, in.Name, "prod")
		requireEqual(t, in.BaseURL, "https://api.anthropic.com")
		return testProviderAccount, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"name":          "prod",
		"base_url":      "https://api.anthropic.com",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body providerAccountDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testProvAcct.String())
	requireEqual(t, body.ProviderKind, domain.ProviderKindAnthropic)
	requireEqual(t, body.Name, "Anthropic Prod")
	requireEqual(t, body.Status, domain.ProviderStatusActive)
	requireAudit(t, st, "provider_account.create")
}

func TestCreateProviderAccountEmptyName(t *testing.T) {
	srv, st := newTestServer(t)
	called := false
	st.providerAccounts.create = func(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error) {
		called = true
		return testProviderAccount, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"name":          "  ",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusBadRequest)
	requireEqual(t, code, "invalid_argument")
	requireEqual(t, called, false)
}

func TestCreateProviderAccountUnknownProviderKind(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{
		"provider_kind": "bogus",
		"name":          "prod",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusBadRequest)
	requireEqual(t, code, "invalid_argument")
}

func TestCreateProviderAccountMemberDenied(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"name":          "prod",
	}, bearerHeader(st, domain.RoleMember))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusForbidden)
	requireEqual(t, code, "permission_denied")
}

func TestListProviderAccounts(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderAccount{testProviderAccount}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []providerAccountDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testProvAcct.String())
}

func TestGetProviderAccount(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testProvAcct)
		return testProviderAccount, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts/"+testProvAcct.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body providerAccountDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.BaseURL, "https://api.anthropic.com")
}

func TestGetProviderAccountNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/provider-accounts/"+testProvAcct.String(), nil, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusNotFound)
	requireEqual(t, code, "not_found")
}

// ---- provider credentials -------------------------------------------------

func TestCreateProviderCredential(t *testing.T) {
	srv, st := newTestServer(t)

	var captured domain.NewProviderCredential
	st.providerAccounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		requireEqual(t, id, testProvAcct)
		return testProviderAccount, nil
	}
	st.providerCredentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
		captured = in
		return credentialFromNew(testProvCred, in), nil
	}

	const secret = "sk-ant-prod-1234"
	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "prod key",
		"kind":   domain.CredentialKindAPIKey,
		"secret": secret,
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	requireEqual(t, captured.Name, "prod key")
	requireEqual(t, captured.Kind, domain.CredentialKindAPIKey)
	requireEqual(t, captured.ProviderAccountID, testProvAcct)
	if captured.SecretRef == "" {
		t.Fatal("SecretRef is empty, want a stored-secret reference")
	}
	if len(captured.Tail) != 4 {
		t.Fatalf("Tail = %q, want a 4-character tail", captured.Tail)
	}
	stored, ok := st.providerSecrets.values[captured.SecretRef]
	if !ok || stored != secret {
		t.Fatalf("secret store missing new ref: got %q, want %q", stored, secret)
	}

	var body map[string]any
	decodeBody(t, resp, &body)
	requireEqual(t, body["name"], "prod key")
	if v := body["tail"]; v != captured.Tail {
		t.Fatalf("tail = %v, want %q", v, captured.Tail)
	}
	if _, has := body["secret_ref"]; has {
		t.Fatal("response exposes secret_ref; provider credential DTO must never leak it")
	}
	requireAudit(t, st, "provider_credential.create")
}

func TestCreateProviderCredentialBlankSecret(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "prod key",
		"kind":   domain.CredentialKindAPIKey,
		"secret": "  ",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusBadRequest)
	requireEqual(t, code, "invalid_argument")
}

func TestCreateProviderCredentialAccountNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return domain.ProviderAccount{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "prod key",
		"kind":   domain.CredentialKindAPIKey,
		"secret": "sk-ant-prod-1234",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusNotFound)
	requireEqual(t, code, "not_found")
}

func TestCreateProviderCredentialRejectedByProvider(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return testProviderAccount, nil
	}
	st.providerClient.validateErr = providerclient.ErrInvalidCredentials
	st.providerCredentials.create = func(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error) {
		t.Fatal("repo create must not run when the provider rejects the secret")
		return domain.ProviderCredential{}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "prod key",
		"kind":   domain.CredentialKindAPIKey,
		"secret": "sk-ant-bad",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusBadRequest)
	requireEqual(t, code, "invalid_provider_credential")
	requireEqual(t, len(st.providerSecrets.values), 0)
	requireEqual(t, len(st.audits.actions), 0)
}

func TestCreateProviderCredentialMemberDenied(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "prod key",
		"kind":   domain.CredentialKindAPIKey,
		"secret": "sk-ant-prod-1234",
	}, bearerHeader(st, domain.RoleMember))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusForbidden)
	requireEqual(t, code, "permission_denied")
}

func TestListProviderCredentials(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerCredentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderCredential{testProviderCredential}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []map[string]any `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0]["name"], "prod key")
	requireEqual(t, body.Items[0]["tail"], "0a08")
	if _, has := body.Items[0]["secret_ref"]; has {
		t.Fatal("list response exposes secret_ref; provider credential DTO must never leak it")
	}
}

func TestGetProviderCredential(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerCredentials.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testProvCred)
		return testProviderCredential, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/provider-credentials/"+testProvCred.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body map[string]any
	decodeBody(t, resp, &body)
	if v := body["fingerprint"]; v != testProviderCredential.Fingerprint {
		t.Fatalf("fingerprint = %v, want %q", v, testProviderCredential.Fingerprint)
	}
	if _, has := body["secret_ref"]; has {
		t.Fatal("get response exposes secret_ref; provider credential DTO must never leak it")
	}
}

func TestRevokeProviderCredential(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerCredentials.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testProvCred)
		revoked := testProviderCredential
		revoked.Status = domain.ProviderCredentialStatusRevoked
		return revoked, nil
	}
	st.providerSecrets.values[testProviderCredential.SecretRef] = "sk-ant-prod-1234"

	resp := doJSON(t, srv, http.MethodDelete, "/v1/provider-credentials/"+testProvCred.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	if _, has := st.providerSecrets.values[testProviderCredential.SecretRef]; has {
		t.Fatal("secret must be deleted from the store after revoke")
	}
	requireAudit(t, st, "provider_credential.revoke")
}

func TestRevokeProviderCredentialMemberDenied(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/provider-credentials/"+testProvCred.String(), nil, bearerHeader(st, domain.RoleMember))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusForbidden)
	requireEqual(t, code, "permission_denied")
}

func TestRotateProviderCredential(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerCredentials.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
		return testProviderCredential, nil
	}
	st.providerAccounts.get = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error) {
		return testProviderAccount, nil
	}
	st.providerCredentials.create = func(_ context.Context, _ uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
		requireEqual(t, in.Name, "prod key (rotated)")
		requireEqual(t, in.ProviderAccountID, testProvAcct)
		return credentialFromNew(uuid.New(), in), nil
	}
	st.providerCredentials.revoke = func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error) {
		revoked := testProviderCredential
		revoked.Status = domain.ProviderCredentialStatusRevoked
		return revoked, nil
	}
	st.providerSecrets.values[testProviderCredential.SecretRef] = "sk-ant-old"

	const newSecret = "sk-ant-new-5678"
	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-credentials/"+testProvCred.String()+"/rotate", map[string]any{
		"secret": newSecret,
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	if _, has := st.providerSecrets.values[testProviderCredential.SecretRef]; has {
		t.Fatal("old secret must be deleted from the store after rotate")
	}
	stored := false
	for ref, value := range st.providerSecrets.values {
		if value == newSecret {
			stored = true
			_ = ref
		}
	}
	if !stored {
		t.Fatalf("new secret not stored; store = %v", st.providerSecrets.values)
	}
	var body map[string]any
	decodeBody(t, resp, &body)
	requireEqual(t, body["name"], "prod key (rotated)")
	requireAudit(t, st, "provider_credential.rotate")
}

func TestRotateProviderCredentialBlankSecret(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-credentials/"+testProvCred.String()+"/rotate", map[string]any{
		"secret": "",
	}, bearerHeader(st, domain.RoleAdmin))

	status, code := decodeError(t, resp)
	requireEqual(t, status, http.StatusBadRequest)
	requireEqual(t, code, "invalid_argument")
}

// ---- models ---------------------------------------------------------------

func TestListModels(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerModels.list = func(_ context.Context) ([]domain.Model, error) {
		return []domain.Model{testModel}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/models", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []modelDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ModelSlug, "claude-sonnet-4-5")
	requireEqual(t, body.Items[0].IsActive, true)
}

// Contract: provider_kind + "/" + model_slug composes the qualified ID the
// resolver emits as ModelHandle.ModelID (application/provider.QualifiedModelID).
func TestListModelsContractQualifiedID(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerModels.list = func(_ context.Context) ([]domain.Model, error) {
		return []domain.Model{
			testModel,
			{
				ID:           uuid.MustParse("00000000-0000-0000-0000-00000000000b"),
				ProviderKind: domain.ProviderKindOpenAI,
				ModelSlug:    "gpt-4o",
				DisplayName:  "GPT-4o",
				IsActive:     true,
				CreatedAt:    testNow,
				UpdatedAt:    testNow,
			},
		}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/models", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []modelDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 2)

	want := map[string]bool{
		"anthropic/claude-sonnet-4-5": false,
		"openai/gpt-4o":               false,
	}
	for _, item := range body.Items {
		qualified := item.ProviderKind + "/" + item.ModelSlug
		if _, ok := want[qualified]; !ok {
			t.Fatalf("models listing returned qualified ID %q, want only %q or %q",
				qualified, "anthropic/claude-sonnet-4-5", "openai/gpt-4o")
		}
		want[qualified] = true
	}
	for qualified, seen := range want {
		if !seen {
			t.Fatalf("models listing missing qualified ID %q", qualified)
		}
	}
}

// credentialFromNew materializes a stored provider credential from the values
// the service wrote, mirroring the repository's row construction.
func credentialFromNew(id uuid.UUID, in domain.NewProviderCredential) domain.ProviderCredential {
	return domain.ProviderCredential{
		ID:                id,
		OrganizationID:    testOrgID,
		ProviderAccountID: in.ProviderAccountID,
		Name:              in.Name,
		Kind:              in.Kind,
		SecretRef:         in.SecretRef,
		Fingerprint:       in.Fingerprint,
		Tail:              in.Tail,
		Status:            domain.ProviderCredentialStatusActive,
		CreatedAt:         testNow,
		UpdatedAt:         testNow,
	}
}
