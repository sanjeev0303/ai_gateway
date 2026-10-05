package controlapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// The organization payload must only carry persisted columns. Anything the UI
// used to invent (region, MRR, seat counts, billing email) is a regression if
// it reappears, so assert absence explicitly rather than by DTO inspection.
func assertNoFabricatedFields(t *testing.T, raw map[string]any) {
	t.Helper()
	for _, k := range []string{
		"region", "mrr", "mrr_cents", "seats_used", "seats_total",
		"seats", "billing_email", "subscription_tier", "tier",
	} {
		if _, ok := raw[k]; ok {
			t.Errorf("organization payload contains unsupported field %q", k)
		}
	}
}

func decodeRaw(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := map[string]any{}
	decodeBody(t, resp, &raw)
	return raw
}

func TestGetOrganizationWithoutSubscription(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		requireEqual(t, id, testOrgID)
		return testOrg, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organization", nil, bearerHeader(st, domain.RoleMember))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)

	requireEqual(t, raw["id"].(string), testOrgID.String())
	requireEqual(t, raw["name"], "Acme")
	requireEqual(t, raw["slug"], "acme")
	requireEqual(t, raw["status"], domain.OrgStatusActive)
	// No subscription row means no plan and no billing block, not empty strings
	// or zeroes that a UI could mistake for a real reading.
	if raw["plan"] != nil {
		t.Errorf("plan = %v, want null when the organization has no subscription", raw["plan"])
	}
	if raw["billing"] != nil {
		t.Errorf("billing = %v, want null when the organization has no subscription", raw["billing"])
	}
	assertNoFabricatedFields(t, raw)
}

func TestGetOrganizationWithActiveSubscription(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return testOrg, nil
	}
	start := testNow.Add(-24 * time.Hour)
	end := testNow.Add(30 * 24 * time.Hour)
	st.subs.activeWithPlanFn = func(_ context.Context, id uuid.UUID) (domain.OrganizationPlan, error) {
		requireEqual(t, id, testOrgID)
		return domain.OrganizationPlan{
			PlanCode:           "scale",
			PlanName:           "Scale",
			Status:             "active",
			CurrentPeriodStart: &start,
			CurrentPeriodEnd:   &end,
			CancelAtPeriodEnd:  true,
		}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organization", nil, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)

	plan, ok := raw["plan"].(map[string]any)
	if !ok {
		t.Fatalf("plan = %#v, want an object", raw["plan"])
	}
	requireEqual(t, plan["code"], "scale")
	requireEqual(t, plan["name"], "Scale")
	requireEqual(t, len(plan), 2)

	billing, ok := raw["billing"].(map[string]any)
	if !ok {
		t.Fatalf("billing = %#v, want an object", raw["billing"])
	}
	requireEqual(t, billing["status"], "active")
	requireEqual(t, billing["cancel_at_period_end"], true)
	requireEqual(t, billing["current_period_start"].(string), start.UTC().Format(time.RFC3339))
	requireEqual(t, billing["current_period_end"].(string), end.UTC().Format(time.RFC3339))
	assertNoFabricatedFields(t, raw)
}

// A trialing subscription is surfaced like an active one; only the stored
// status string differs.
func TestGetOrganizationTrialingSubscription(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return testOrg, nil
	}
	st.subs.activeWithPlanFn = func(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
		return domain.OrganizationPlan{PlanCode: "starter", PlanName: "Starter", Status: "trialing"}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organization", nil, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)

	billing, ok := raw["billing"].(map[string]any)
	if !ok {
		t.Fatalf("billing = %#v, want an object", raw["billing"])
	}
	requireEqual(t, billing["status"], "trialing")
}

// NULL period columns must not serialize as a fabricated timestamp.
func TestGetOrganizationNullBillingPeriods(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return testOrg, nil
	}
	st.subs.activeWithPlanFn = func(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
		return domain.OrganizationPlan{PlanCode: "starter", PlanName: "Starter", Status: "active"}, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organization", nil, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)

	billing := raw["billing"].(map[string]any)
	for _, k := range []string{"current_period_start", "current_period_end"} {
		if billing[k] != "" {
			t.Errorf("billing.%s = %#v, want empty string for a NULL column", k, billing[k])
		}
	}
}

func TestGetOrganizationPlanLookupFailureIsSurfaced(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return testOrg, nil
	}
	st.subs.activeWithPlanFn = func(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
		return domain.OrganizationPlan{}, repositories.ErrInvalid
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organization", nil, bearerHeader(st, domain.RoleOwner))
	defer resp.Body.Close()

	// A broken billing read must not be reported as a healthy organization.
	if resp.StatusCode == http.StatusOK {
		t.Fatal("status = 200, want the plan lookup failure surfaced")
	}
}

// GET /v1/organizations/me is the alias the web client actually calls.
func TestGetOrganizationViaMeAlias(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return testOrg, nil
	}

	resp := doJSON(t, srv, http.MethodGet, "/v1/organizations/me", nil, bearerHeader(st, domain.RoleMember))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)
	requireEqual(t, raw["slug"], "acme")
}

// PATCH returns the updated core fields with no plan block: the write path does
// not re-read billing, so claiming a plan here would be inventing it.
func TestUpdateOrganizationOmitsPlan(t *testing.T) {
	srv, st := newTestServer(t)
	name := "Acme Renamed"
	st.orgs.update = func(_ context.Context, id uuid.UUID, in domain.UpdateOrganization) (domain.Organization, error) {
		requireEqual(t, id, testOrgID)
		requireEqual(t, *in.Name, name)
		requireEqual(t, *in.Slug, "acme-renamed")
		updated := testOrg
		updated.Name = name
		updated.Slug = "acme-renamed"
		return updated, nil
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/organization",
		map[string]string{"name": name, "slug": "acme-renamed"},
		bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	raw := decodeRaw(t, resp)

	requireEqual(t, raw["name"].(string), name)
	requireEqual(t, raw["slug"], "acme-renamed")
	if raw["plan"] != nil || raw["billing"] != nil {
		t.Errorf("PATCH response carried plan/billing (%v/%v), want null on a write", raw["plan"], raw["billing"])
	}
	assertNoFabricatedFields(t, raw)
}

func TestUpdateOrganizationRejectsBlankName(t *testing.T) {
	srv, st := newTestServer(t)

	resp := doJSON(t, srv, http.MethodPatch, "/v1/organization",
		map[string]string{"name": "   "}, bearerHeader(st, domain.RoleOwner))
	defer resp.Body.Close()

	code, _ := decodeError(t, resp)
	requireEqual(t, code, http.StatusBadRequest)
}

// Guards the contract the web client depends on: PATCH targets
// /v1/organization, not /v1/organizations/me.
func TestUpdateOrganizationRouteShape(t *testing.T) {
	srv, st := newTestServer(t)
	st.orgs.update = func(context.Context, uuid.UUID, domain.UpdateOrganization) (domain.Organization, error) {
		return testOrg, nil
	}

	resp := doJSON(t, srv, http.MethodPatch, "/v1/organization",
		map[string]string{"name": "Acme"}, bearerHeader(st, domain.RoleOwner))
	requireEqual(t, resp.StatusCode, http.StatusOK)
	resp.Body.Close()

	// The path the web client used to call must not silently accept PATCH.
	alias := doJSON(t, srv, http.MethodPatch, "/v1/organizations/me",
		map[string]string{"name": "Acme"}, bearerHeader(st, domain.RoleOwner))
	alias.Body.Close()
	requireEqual(t, alias.StatusCode, http.StatusMethodNotAllowed)
}
