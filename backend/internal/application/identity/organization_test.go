package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

func TestGetOrganization(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.orgs.getByID = func(_ context.Context, id uuid.UUID) (domain.Organization, error) {
		requireEqual(t, id, testOrgID)
		return domain.Organization{ID: testOrgID, Name: "Acme", Slug: "acme", Status: domain.OrgStatusActive}, nil
	}
	// A never-subscribed organization is the common case: no row, no plan.
	st.subs.activeWithPlan = func(_ context.Context, id uuid.UUID) (domain.OrganizationPlan, error) {
		requireEqual(t, id, testOrgID)
		return domain.OrganizationPlan{}, repositories.ErrNotFound
	}

	org, plan, err := svc.GetOrganization(ctx, userActor(domain.RoleMember))
	requireNoErr(t, err)
	requireEqual(t, org.ID, testOrgID)
	requireEqual(t, org.Name, "Acme")
	if plan != nil {
		t.Fatalf("plan = %+v, want nil for an organization with no subscription", *plan)
	}
}

func TestGetOrganizationReturnsActivePlan(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: testOrgID, Name: "Acme", Slug: "acme", Status: domain.OrgStatusActive}, nil
	}
	start := testNow.Add(-24 * time.Hour)
	end := testNow.Add(30 * 24 * time.Hour)
	st.subs.activeWithPlan = func(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
		return domain.OrganizationPlan{
			PlanCode:           "scale",
			PlanName:           "Scale",
			Status:             "active",
			CurrentPeriodStart: &start,
			CurrentPeriodEnd:   &end,
			CancelAtPeriodEnd:  false,
		}, nil
	}

	_, plan, err := svc.GetOrganization(ctx, userActor(domain.RoleMember))
	requireNoErr(t, err)
	if plan == nil {
		t.Fatal("plan = nil, want the active subscription's plan")
	}
	requireEqual(t, plan.PlanCode, "scale")
	requireEqual(t, plan.PlanName, "Scale")
	requireEqual(t, plan.Status, "active")
	requireEqual(t, plan.CurrentPeriodStart.Format(time.RFC3339), start.Format(time.RFC3339))
	requireEqual(t, plan.CurrentPeriodEnd.Format(time.RFC3339), end.Format(time.RFC3339))
	requireEqual(t, plan.CancelAtPeriodEnd, false)
}

// A subscription lookup failure is a real error: reporting the organization
// with a silent nil plan would hide broken billing reads.
func TestGetOrganizationFailsWhenPlanLookupFails(t *testing.T) {
	ctx := context.Background()
	svc, st := newTestService()
	st.orgs.getByID = func(context.Context, uuid.UUID) (domain.Organization, error) {
		return domain.Organization{ID: testOrgID}, nil
	}
	boom := errors.New("subscription store unavailable")
	st.subs.activeWithPlan = func(context.Context, uuid.UUID) (domain.OrganizationPlan, error) {
		return domain.OrganizationPlan{}, boom
	}

	_, _, err := svc.GetOrganization(ctx, userActor(domain.RoleMember))
	requireErrIs(t, err, boom)
}

func TestGetOrganizationDeniedWithoutPermission(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	_, _, err := svc.GetOrganization(ctx, userActor("nonsense"))
	requireErrIs(t, err, repositories.ErrForbidden)
}
