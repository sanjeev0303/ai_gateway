package controlapi

import (
	"context"
	"net/http"
	"strings"
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

	st.providerAccounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		return []domain.ProviderAccount{
			{ID: testProvAcct, ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderStatusActive},
		}, nil
	}
	st.providerCredentials.listByAccount = func(_ context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: testProvCred, Status: domain.ProviderCredentialStatusActive},
		}, nil
	}

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

	st.providerAccounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		return []domain.ProviderAccount{
			{ID: testProvAcct, ProviderKind: domain.ProviderKindAnthropic, Status: domain.ProviderStatusActive},
			{ID: uuid.New(), ProviderKind: domain.ProviderKindOpenAI, Status: domain.ProviderStatusActive},
		}, nil
	}
	st.providerCredentials.listByAccount = func(_ context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: uuid.New(), Status: domain.ProviderCredentialStatusActive},
		}, nil
	}

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

// TestProviderOnboardingEndToEnd walks the full provider path a real customer
// follows: create the account, store a credential, prove the secret is
// accepted, probe connectivity, sync the live model catalog, and read it back.
func TestProviderOnboardingEndToEnd(t *testing.T) {
	srv, st := newTestServer(t)
	hdr := bearerHeader(st, domain.RoleAdmin)

	st.providerAccounts.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderAccount) (domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, in.ProviderKind, domain.ProviderKindAnthropic)
		return testProviderAccount, nil
	}
	st.providerAccounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testProvAcct)
		return testProviderAccount, nil
	}
	st.providerAccounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderAccount{testProviderAccount}, nil
	}
	st.providerCredentials.listByAccount = func(_ context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		return []domain.ProviderCredential{testProviderCredential}, nil
	}

	var stored domain.NewProviderCredential
	st.providerCredentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
		requireEqual(t, orgID, testOrgID)
		stored = in
		// Echo back what the service derived so the DTO reflects real
		// service behavior rather than stub-supplied constants.
		return domain.ProviderCredential{
			ID: testProvCred, OrganizationID: testOrgID, ProviderAccountID: testProvAcct,
			Kind: in.Kind, SecretRef: in.SecretRef, Fingerprint: in.Fingerprint,
			Tail: in.Tail, Status: domain.ProviderCredentialStatusActive,
		}, nil
	}

	st.providerClient.validateErr = nil
	st.providerClient.healthErr = nil
	st.providerClient.listModels = func(_ context.Context, apiKey string) ([]providerclient.Model, error) {
		requireEqual(t, apiKey, "sk-ant-e2e")
		return []providerclient.Model{{ID: "claude-sonnet-4-5"}, {ID: "claude-haiku-4-5"}}, nil
	}
	// upsertedIDs records every row the service persists, and the stub returns a
	// fresh non-zero ID per call. Asserting on both makes the test fail if the
	// service ever builds a model in memory instead of persisting it.
	var upsertedIDs []uuid.UUID
	st.providerModels.upsert = func(_ context.Context, kind, slug, display string) (domain.Model, error) {
		requireEqual(t, kind, domain.ProviderKindAnthropic)
		id := uuid.New()
		upsertedIDs = append(upsertedIDs, id)
		return domain.Model{
			ID: id, ProviderKind: kind, ModelSlug: slug, DisplayName: display, IsActive: true,
		}, nil
	}
	catalog := []domain.Model{
		{ID: uuid.New(), ProviderKind: domain.ProviderKindAnthropic, ModelSlug: "claude-sonnet-4-5", DisplayName: "claude-sonnet-4-5", IsActive: true},
		{ID: uuid.New(), ProviderKind: domain.ProviderKindAnthropic, ModelSlug: "claude-haiku-4-5", DisplayName: "claude-haiku-4-5", IsActive: true},
	}
	st.providerModels.list = func(context.Context) ([]domain.Model, error) { return catalog, nil }

	// 1. create the provider account
	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"name":          "prod",
	}, hdr)
	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var account providerAccountDTO
	decodeBody(t, resp, &account)
	requireEqual(t, account.ID, testProvAcct.String())

	// 2. store a credential; the secret reaches the secret store, not the DTO
	resp = doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/credentials", map[string]any{
		"name":   "primary",
		"kind":   domain.CredentialKindAPIKey,
		"secret": "sk-ant-e2e",
	}, hdr)
	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var cred providerCredentialDTO
	decodeBody(t, resp, &cred)
	requireEqual(t, cred.ID, testProvCred.String())
	// The service derives the tail and a SHA-256 fingerprint from the secret;
	// the DTO carries those, never the secret or the secret-store ref.
	requireEqual(t, cred.Tail, "-e2e")
	if stored.Fingerprint == "" || cred.Fingerprint != stored.Fingerprint {
		t.Fatalf("credential DTO must carry the derived fingerprint, got %q", cred.Fingerprint)
	}
	// "sha256:" + 64 hex chars, and it must not embed the secret itself.
	requireEqual(t, len(cred.Fingerprint), len("sha256:")+64)
	if !strings.HasPrefix(cred.Fingerprint, "sha256:") {
		t.Fatalf("fingerprint must be sha256-prefixed, got %q", cred.Fingerprint)
	}
	if strings.Contains(cred.Fingerprint, "sk-ant") {
		t.Fatal("fingerprint must not contain the secret")
	}
	if stored.SecretRef == "" {
		t.Fatal("service must persist the secret behind a SecretRef")
	}
	if len(st.providerSecrets.values) != 1 {
		t.Fatalf("expected exactly 1 secret stored, got %d", len(st.providerSecrets.values))
	}
	for _, v := range st.providerSecrets.values {
		requireEqual(t, v, "sk-ant-e2e")
	}

	// 3. validate the secret against the live provider
	resp = doJSON(t, srv, http.MethodPost, "/v1/provider-credentials/validate", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"api_key":       "sk-ant-e2e",
	}, hdr)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var valid struct {
		Valid bool `json:"valid"`
	}
	decodeBody(t, resp, &valid)
	if !valid.Valid {
		t.Fatal("expected valid=true for an accepted secret")
	}

	// 4. probe connectivity and persist the observed state
	resp = doJSON(t, srv, http.MethodPost, "/v1/provider-health/check", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"api_key":       "sk-ant-e2e",
	}, hdr)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var health providerHealthDTO
	decodeBody(t, resp, &health)
	requireEqual(t, health.ProviderKind, domain.ProviderKindAnthropic)
	requireEqual(t, health.Status, domain.ProviderHealthHealthy)

	// 5. read the recorded health back
	resp = doJSON(t, srv, http.MethodGet, "/v1/provider-health", nil, hdr)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var healthList struct {
		Items []providerHealthDTO `json:"items"`
	}
	decodeBody(t, resp, &healthList)
	requireEqual(t, len(healthList.Items), 1)
	requireEqual(t, healthList.Items[0].Status, domain.ProviderHealthHealthy)

	// 6. sync the live model catalog
	resp = doJSON(t, srv, http.MethodPost, "/v1/models/sync", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"api_key":       "sk-ant-e2e",
	}, hdr)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var synced struct {
		Items []modelDTO `json:"items"`
	}
	decodeBody(t, resp, &synced)
	requireEqual(t, len(synced.Items), 2)
	requireEqual(t, synced.Items[0].ModelSlug, "claude-sonnet-4-5")
	requireEqual(t, synced.Items[0].IsActive, true)
	// Every synced row must have come back from the model repository.
	requireEqual(t, len(upsertedIDs), 2)
	requireEqual(t, synced.Items[0].ID, upsertedIDs[0].String())
	requireEqual(t, synced.Items[1].ID, upsertedIDs[1].String())

	// 7. the catalog is what GET /v1/models serves
	resp = doJSON(t, srv, http.MethodGet, "/v1/models", nil, hdr)
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var listed struct {
		Items []modelDTO `json:"items"`
	}
	decodeBody(t, resp, &listed)
	requireEqual(t, len(listed.Items), 2)
	requireEqual(t, listed.Items[1].ModelSlug, "claude-haiku-4-5")

	requireAudit(t, st, "provider_account.create")
	requireAudit(t, st, "provider.models.sync")
}

// TestValidateProviderCredentialRejected covers the upstream-rejection path:
// a bad key is a 200 with valid=false, not a gateway error.
func TestValidateProviderCredentialRejected(t *testing.T) {
	srv, st := newTestServer(t)
	st.providerClient.validateErr = providerclient.ErrInvalidCredentials

	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-credentials/validate", map[string]any{
		"provider_kind": domain.ProviderKindAnthropic,
		"api_key":       "sk-ant-bad",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Valid bool `json:"valid"`
	}
	decodeBody(t, resp, &body)
	if body.Valid {
		t.Fatal("expected valid=false for a rejected secret")
	}
}

func TestUpdateProviderAccountStatus(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.updateStatus = func(_ context.Context, orgID, id uuid.UUID, status string) (domain.ProviderAccount, error) {
		requireEqual(t, orgID, testOrgID)
		requireEqual(t, id, testProvAcct)
		requireEqual(t, status, domain.ProviderStatusDisabled)
		acct := testProviderAccount
		acct.Status = status
		return acct, nil
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/provider-accounts/"+testProvAcct.String(), map[string]any{
		"status": domain.ProviderStatusDisabled,
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body providerAccountDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.Status, domain.ProviderStatusDisabled)
}

func TestDeleteProviderAccount(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return testProviderAccount, nil
	}
	st.providerCredentials.listByAccount = func(_ context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: testProvCred, SecretRef: "provider/creds/test-ref"},
		}, nil
	}
	st.providerCredentials.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
		return domain.ProviderCredential{ID: id}, nil
	}
	st.providerSecrets.values["provider/creds/test-ref"] = "sk-secret"

	deleted := false
	st.providerAccounts.delete = func(_ context.Context, orgID, id uuid.UUID) error {
		deleted = true
		return nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/provider-accounts/"+testProvAcct.String(), nil, bearerHeader(st, domain.RoleAdmin))
	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	if !deleted {
		t.Fatal("expected account to be deleted")
	}
}

func TestAccountLevelOperationsWithStoredSecret(t *testing.T) {
	srv, st := newTestServer(t)

	st.providerAccounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
		return testProviderAccount, nil
	}
	st.providerCredentials.listByAccount = func(_ context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
		return []domain.ProviderCredential{
			{ID: testProvCred, SecretRef: "provider/creds/test-ref"},
		}, nil
	}
	st.providerSecrets.values["provider/creds/test-ref"] = "sk-ant-valid"

	// 1. POST /v1/provider-accounts/{id}/validate
	resp := doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/validate", nil, bearerHeader(st, domain.RoleAdmin))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var valBody struct {
		Valid bool `json:"valid"`
	}
	decodeBody(t, resp, &valBody)
	if !valBody.Valid {
		t.Fatal("expected valid=true for stored valid secret")
	}

	// 2. POST /v1/provider-accounts/{id}/health
	resp = doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/health", nil, bearerHeader(st, domain.RoleAdmin))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var healthBody providerHealthDTO
	decodeBody(t, resp, &healthBody)
	requireEqual(t, healthBody.Status, domain.ProviderHealthHealthy)

	// 3. POST /v1/provider-accounts/{id}/sync
	st.providerClient.listModels = func(context.Context, string) ([]providerclient.Model, error) {
		return []providerclient.Model{
			{ID: "claude-sonnet-4-5", DisplayName: "Claude 3.5 Sonnet"},
		}, nil
	}
	st.providerModels.upsert = func(_ context.Context, kind, slug, name string) (domain.Model, error) {
		return domain.Model{ProviderKind: kind, ModelSlug: slug, DisplayName: name, IsActive: true}, nil
	}
	resp = doJSON(t, srv, http.MethodPost, "/v1/provider-accounts/"+testProvAcct.String()+"/sync", nil, bearerHeader(st, domain.RoleAdmin))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	var syncBody struct {
		Items []modelDTO `json:"items"`
	}
	decodeBody(t, resp, &syncBody)
	requireEqual(t, len(syncBody.Items), 1)
	requireEqual(t, syncBody.Items[0].ModelSlug, "claude-sonnet-4-5")
}
