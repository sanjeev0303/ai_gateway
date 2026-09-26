package controlapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- create credential ------------------------------------------------------

func TestCreateCredentialGatewayHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var capturedRaw string
	var captured domain.NewAPICredential
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		capturedRaw = secret
		return "hash:" + secret, nil
	}
	st.credentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
		captured = in
		return domain.APICredential{
			ID: testOtherID, OrganizationID: orgID, Name: in.Name,
			KeyPrefix: in.KeyPrefix, KeyHash: in.KeyHash, Scope: in.Scope,
			Status: domain.APICredentialStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "gateway key",
		"scope": "gateway",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body createCredentialResponse
	decodeBody(t, resp, &body)

	requireEqual(t, len(capturedRaw), 37)
	requireEqual(t, strings.HasPrefix(capturedRaw, "aigw_"), true)
	requireEqual(t, body.APIKey, capturedRaw)
	requireEqual(t, body.Credential.ID, testOtherID.String())
	requireEqual(t, body.Credential.KeyPrefix, capturedRaw[:12])
	requireEqual(t, body.Credential.Scope, "gateway")
	requireEqual(t, body.Credential.Status, domain.APICredentialStatusActive)

	requireEqual(t, captured.Name, "gateway key")
	requireEqual(t, captured.KeyPrefix, capturedRaw[:12])
	requireEqual(t, captured.KeyHash, "hash:"+capturedRaw)
	requireEqual(t, captured.Scope, string(domain.APICredentialScopeGateway))
	if captured.ServiceAccountID != nil {
		t.Fatalf("ServiceAccountID = %v, want nil", captured.ServiceAccountID)
	}
	requireAudit(t, st, "api_credential.create")
}

func TestCreateCredentialControlHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return testServiceAccount, nil
	}
	var captured domain.NewAPICredential
	st.hasher.hash = func(_ context.Context, secret string) (string, error) {
		return "hash:" + secret, nil
	}
	st.credentials.create = func(_ context.Context, orgID uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
		captured = in
		return domain.APICredential{
			ID: testOtherID, OrganizationID: orgID, Name: in.Name,
			KeyPrefix: in.KeyPrefix, KeyHash: in.KeyHash, Scope: in.Scope,
			ServiceAccountID: in.ServiceAccountID,
			Status:           domain.APICredentialStatusActive, CreatedAt: testNow, UpdatedAt: testNow,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":               "control key",
		"scope":              "control",
		"service_account_id": testAcctID.String(),
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body createCredentialResponse
	decodeBody(t, resp, &body)

	if captured.ServiceAccountID == nil || *captured.ServiceAccountID != testAcctID {
		t.Fatalf("NewAPICredential.ServiceAccountID = %v, want %s", captured.ServiceAccountID, testAcctID)
	}
	if body.Credential.ServiceAccountID == nil || *body.Credential.ServiceAccountID != testAcctID.String() {
		t.Fatalf("Credential.ServiceAccountID = %v, want %s", body.Credential.ServiceAccountID, testAcctID.String())
	}
	requireEqual(t, captured.Scope, string(domain.APICredentialScopeControl))
	requireEqual(t, body.Credential.Scope, "control")
	requireAudit(t, st, "api_credential.create")
}

func TestCreateCredentialRejectsBlankName(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "   ",
		"scope": "gateway",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateCredentialRejectsUnknownScope(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "key",
		"scope": "bogus",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateCredentialControlRequiresServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "control key",
		"scope": "control",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateCredentialAllScopeRequiresServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "all key",
		"scope": "all",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateCredentialRejectsUnknownServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return domain.ServiceAccount{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":               "control key",
		"scope":              "control",
		"service_account_id": testAcctID.String(),
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestCreateCredentialRejectsRevokedServiceAccount(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.get = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		sa := testServiceAccount
		sa.Status = domain.ServiceAccountStatusRevoked
		return sa, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":               "control key",
		"scope":              "control",
		"service_account_id": testAcctID.String(),
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateCredentialRejectsInvalidServiceAccountID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":               "control key",
		"scope":              "control",
		"service_account_id": "not-a-uuid",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestCreateCredentialRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "key",
		"scope": 42,
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestCreateCredentialForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/credentials", map[string]any{
		"name":  "key",
		"scope": "gateway",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list credentials -------------------------------------------------------

func TestListCredentialsHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.credentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.APICredential, error) {
		return []domain.APICredential{testCredential}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/credentials", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []credentialDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testOtherID.String())
	requireEqual(t, body.Items[0].Name, "gateway key")
	requireEqual(t, body.Items[0].KeyPrefix, "aigw_ab12cd")
	requireEqual(t, body.Items[0].Scope, "gateway")
	requireEqual(t, body.Items[0].Status, domain.APICredentialStatusActive)
}

func TestListCredentialsStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)

	st.credentials.list = func(_ context.Context, orgID uuid.UUID) ([]domain.APICredential, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/credentials", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestListCredentialsForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/credentials", nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- revoke credential --------------------------------------------------------

func TestRevokeCredentialHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.credentials.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.APICredential, error) {
		return testCredential, nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/credentials/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
	requireAudit(t, st, "api_credential.revoke")
}

func TestRevokeCredentialNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.credentials.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.APICredential, error) {
		return domain.APICredential{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/credentials/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRevokeCredentialRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/credentials/not-a-uuid", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestRevokeCredentialForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/credentials/"+testOtherID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}
