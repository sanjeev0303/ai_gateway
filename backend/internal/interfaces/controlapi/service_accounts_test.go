package controlapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ---- create service account -------------------------------------------------

func TestCreateServiceAccountHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	var captured domain.NewServiceAccount
	st.accounts.create = func(_ context.Context, orgID uuid.UUID, in domain.NewServiceAccount) (domain.ServiceAccount, error) {
		captured = in
		return testServiceAccount, nil
	}

	resp := doJSON(t, srv, http.MethodPost, "/v1/service-accounts", map[string]any{
		"name":        "ci-bot",
		"description": "CI",
	}, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusCreated)
	var body serviceAccountDTO
	decodeBody(t, resp, &body)
	requireEqual(t, body.ID, testAcctID.String())
	requireEqual(t, body.OrganizationID, testOrgID.String())
	requireEqual(t, body.Name, "ci-bot")
	requireEqual(t, body.Description, "CI")
	requireEqual(t, body.Status, domain.ServiceAccountStatusActive)

	requireEqual(t, captured.Name, "ci-bot")
	requireEqual(t, captured.Description, "CI")
	requireAudit(t, st, "service_account.create")
}

func TestCreateServiceAccountRejectsBlankName(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/service-accounts", map[string]any{
		"name": "   ",
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_argument")
}

func TestCreateServiceAccountRejectsInvalidBody(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/service-accounts", map[string]any{
		"name": 42,
	}, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_json")
}

func TestCreateServiceAccountForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPost, "/v1/service-accounts", map[string]any{
		"name": "ci-bot",
	}, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- list service accounts --------------------------------------------------

func TestListServiceAccountsHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ServiceAccount, error) {
		return []domain.ServiceAccount{testServiceAccount}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/service-accounts", nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusOK)
	var body struct {
		Items []serviceAccountDTO `json:"items"`
	}
	decodeBody(t, resp, &body)
	requireEqual(t, len(body.Items), 1)
	requireEqual(t, body.Items[0].ID, testAcctID.String())
	requireEqual(t, body.Items[0].Name, "ci-bot")
	requireEqual(t, body.Items[0].Status, domain.ServiceAccountStatusActive)
}

func TestListServiceAccountsStoreFailure(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.list = func(_ context.Context, orgID uuid.UUID) ([]domain.ServiceAccount, error) {
		return nil, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/service-accounts", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestListServiceAccountsForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodGet, "/v1/service-accounts", nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}

// ---- revoke service account --------------------------------------------------

func TestRevokeServiceAccountHappyPath(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return testServiceAccount, nil
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/service-accounts/"+testAcctID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	requireEqual(t, resp.StatusCode, http.StatusNoContent)
	resp.Body.Close()
	requireAudit(t, st, "service_account.revoke")
}

func TestRevokeServiceAccountNotFound(t *testing.T) {
	srv, st := newTestServer(t)

	st.accounts.revoke = func(_ context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
		return domain.ServiceAccount{}, repositories.ErrNotFound
	}

	resp := doJSON(t, srv, http.MethodDelete, "/v1/service-accounts/"+testAcctID.String(), nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusNotFound)
	requireEqual(t, msg, "not_found")
}

func TestRevokeServiceAccountRejectsInvalidID(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/service-accounts/not-a-uuid", nil, bearerHeader(st, domain.RoleAdmin))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
	requireEqual(t, msg, "invalid_id")
}

func TestRevokeServiceAccountForbiddenForMember(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodDelete, "/v1/service-accounts/"+testAcctID.String(), nil, bearerHeader(st, domain.RoleMember))

	code, msg := decodeError(t, resp)
	requireEqual(t, code, http.StatusForbidden)
	requireEqual(t, msg, "permission_denied")
}
