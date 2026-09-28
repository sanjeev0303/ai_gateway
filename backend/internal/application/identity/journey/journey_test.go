// Package journey exercises the identity application service end to end
// against real Postgres: a two-tenant user journey (org -> invite -> accept ->
// team/project -> cross-tenant denial) plus machine identity and per-org
// audit-trail assertions. Test-only package; nothing here is importable.
package journey

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/auth"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	repos "ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/testutil"
)

// newJourneyService wires the real repositories and token machinery from the
// module's infrastructure layers into the identity application service.
func newJourneyService(t *testing.T, store *postgres.Store) *identity.Service {
	t.Helper()
	issuer, verifier, err := auth.NewHMACTokenManager(
		"aigw-journey", time.Hour, []byte("journey-test-signing-key-0123456789abcdef"))
	if err != nil {
		t.Fatalf("new hmac token manager: %v", err)
	}
	return identity.NewService(identity.ServiceDeps{
		Orgs:        repos.NewOrganizationRepository(store),
		Users:       repos.NewUserRepository(store),
		Members:     repos.NewOrgMemberRepository(store),
		Invitations: repos.NewInvitationRepository(store),
		Projects:    repos.NewProjectRepository(store),
		Teams:       repos.NewTeamRepository(store),
		TeamMembers: repos.NewTeamMemberRepository(store),
		ProjMembers: repos.NewProjectMemberRepository(store),
		Accounts:    repos.NewServiceAccountRepository(store),
		Credentials: repos.NewAPICredentialRepository(store),
		Audits:      repos.NewAuditLogRepository(store),
		Issuer:      issuer,
		Verifier:    verifier,
		Hasher:      auth.NewSHA256TokenHasher(),
	})
}

func TestFullUserJourney(t *testing.T) {
	ctx := context.Background()
	store := testutil.StoreForTest(t, false)
	svc := newJourneyService(t, store)

	// 1. The owner signs in; their first organization is provisioned and the
	// issued JWT round-trips back into an owner actor of that org.
	ownerSession, err := svc.DevLogin(ctx, "owner@example.com", "Org Owner")
	if err != nil {
		t.Fatalf("owner DevLogin: %v", err)
	}
	org1 := ownerSession.Claims.OrganizationID
	if org1 == uuid.Nil {
		t.Fatal("owner session carries no organization")
	}
	if ownerSession.Claims.Role != domain.RoleOwner {
		t.Fatalf("owner role = %q, want %q", ownerSession.Claims.Role, domain.RoleOwner)
	}
	ownerActor, err := svc.AuthenticateToken(ctx, ownerSession.Token)
	if err != nil {
		t.Fatalf("owner AuthenticateToken: %v", err)
	}
	if ownerActor.OrganizationID != org1 || ownerActor.Role != domain.RoleOwner {
		t.Fatalf("owner actor = %+v, want org1 owner", ownerActor)
	}

	// 2. The owner invites a second user as a member, and a duplicate pending
	// invite for the same email is rejected as a conflict.
	invitation, rawToken, err := svc.Invite(ctx, ownerActor, "dev@example.com", domain.RoleMember)
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if rawToken == "" || invitation.OrganizationID != org1 || invitation.Role != domain.RoleMember {
		t.Fatalf("unexpected invitation: %+v token_len=%d", invitation, len(rawToken))
	}
	if _, _, err := svc.Invite(ctx, ownerActor, "dev@example.com", domain.RoleMember); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("duplicate pending invite err = %v, want ErrConflict", err)
	}

	// 3. The invited user signs in; DevLogin provisions their own org.
	devSession, err := svc.DevLogin(ctx, "dev@example.com", "Dev Person")
	if err != nil {
		t.Fatalf("dev DevLogin: %v", err)
	}
	devID, err := uuid.Parse(devSession.Claims.Subject)
	if err != nil {
		t.Fatalf("dev subject is not a uuid: %v", err)
	}
	org2 := devSession.Claims.OrganizationID
	if org2 == uuid.Nil || org2 == org1 {
		t.Fatalf("dev org = %s, want a distinct provisioned org", org2)
	}

	// 4. The dev accepts using the token from their own (org2) session. The
	// membership and audit record land in the invited org (org1).
	devActor, err := svc.AuthenticateToken(ctx, devSession.Token)
	if err != nil {
		t.Fatalf("dev AuthenticateToken: %v", err)
	}
	member, err := svc.AcceptInvitation(ctx, devActor, rawToken)
	if err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	if member.User.ID != devID || member.Role != domain.RoleMember {
		t.Fatalf("member = %+v, want dev as %q", member, domain.RoleMember)
	}

	// Accepting again is idempotent: the same member returns, no second row.
	again, err := svc.AcceptInvitation(ctx, devActor, rawToken)
	if err != nil {
		t.Fatalf("re-accept invitation: %v", err)
	}
	if again.User.ID != member.User.ID {
		t.Fatalf("re-accept member = %+v, want %+v", again, member)
	}

	// 5. The owner creates a team and a project and adds the dev to both.
	team, err := svc.CreateTeam(ctx, ownerActor, domain.NewTeam{Name: "Core", Description: "core team"})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	project, err := svc.CreateProject(ctx, ownerActor, domain.NewProject{
		Name: "AI Gateway", Slug: "aigw", Description: "gateway project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := svc.AddTeamMember(ctx, ownerActor, team.ID, devID, domain.RoleMember); err != nil {
		t.Fatalf("add team member: %v", err)
	}
	if _, err := svc.AddProjectMember(ctx, ownerActor, project.ID, devID, domain.RoleMember); err != nil {
		t.Fatalf("add project member: %v", err)
	}

	// 6. Cross-tenant access is denied: the dev's org2-scoped token cannot see
	// the org1 team or project.
	if _, err := svc.GetProject(ctx, devActor, project.ID); !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("cross-tenant GetProject err = %v, want ErrNotFound", err)
	}
	if _, err := svc.GetTeam(ctx, devActor, team.ID); !errors.Is(err, repositories.ErrNotFound) {
		t.Fatalf("cross-tenant GetTeam err = %v, want ErrNotFound", err)
	}

	// 7. An org1-scoped actor for the same user reads both resources normally.
	devOrg1Actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         devID,
		OrganizationID: org1,
		Role:           domain.RoleMember,
		DisplayName:    "Dev Person",
	}
	if _, err := svc.GetProject(ctx, devOrg1Actor, project.ID); err != nil {
		t.Fatalf("org1 member GetProject: %v", err)
	}
	if _, err := svc.GetTeam(ctx, devOrg1Actor, team.ID); err != nil {
		t.Fatalf("org1 member GetTeam: %v", err)
	}

	// 8. Once the invitation is accepted its pending slot frees: a fresh
	// invite for the same email succeeds, while duplicate resource names still
	// conflict and write no audit rows.
	rein, _, err := svc.Invite(ctx, ownerActor, "dev@example.com", domain.RoleMember)
	if err != nil {
		t.Fatalf("re-invite after accept: %v", err)
	}
	if rein.ID == invitation.ID {
		t.Fatal("re-invite returned the original invitation")
	}
	if _, err := svc.CreateProject(ctx, ownerActor, domain.NewProject{Name: "AI Gateway", Slug: "aigw"}); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("duplicate project slug err = %v, want ErrConflict", err)
	}
	if _, err := svc.CreateTeam(ctx, ownerActor, domain.NewTeam{Name: "Core"}); !errors.Is(err, repositories.ErrConflict) {
		t.Fatalf("duplicate team name err = %v, want ErrConflict", err)
	}

	// 9. Machine identity: a service account with a scope-all credential
	// authenticates to a service actor of the owning org.
	account, err := svc.CreateServiceAccount(ctx, ownerActor, domain.NewServiceAccount{
		Name: "ci-bot", Description: "CI automation",
	})
	if err != nil {
		t.Fatalf("create service account: %v", err)
	}
	_, rawKey, err := svc.CreateCredential(ctx, ownerActor, "ci-key", string(domain.APICredentialScopeAll), &account.ID)
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	svcActor, err := svc.AuthenticateAPIKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("authenticate api key: %v", err)
	}
	if svcActor.Kind != domain.ActorKindService || svcActor.OrganizationID != org1 || svcActor.Role != domain.RoleService {
		t.Fatalf("api key actor = %+v, want org1 service actor", svcActor)
	}

	// 10. Bogus bearer tokens are rejected outright.
	if _, err := svc.AuthenticateToken(ctx, "not-a-jwt"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("bogus token err = %v, want ErrUnauthorized", err)
	}

	// 11. DevLogin is idempotent: the same user and org come back, and no new
	// organization or user rows appear (2 users, 2 orgs total).
	session2, err := svc.DevLogin(ctx, "owner@example.com", "Org Owner")
	if err != nil {
		t.Fatalf("second owner DevLogin: %v", err)
	}
	if session2.Claims.Subject != ownerSession.Claims.Subject || session2.Claims.OrganizationID != org1 {
		t.Fatalf("second owner session = %+v, want same user/org", session2.Claims)
	}

	// 12. Per-org audit trails are exact: every tenant mutation is recorded
	// under the mutated org, and failed mutations (duplicate invites, slugs,
	// team names) write nothing.
	assertAuditActions(t, ctx, store, org1, []string{
		"api_credential.create",
		"invitation.accept",
		"invitation.create",
		"invitation.create",
		"organization.create",
		"project.create",
		"project_member.add",
		"service_account.create",
		"team.create",
		"team_member.add",
	})
	assertAuditActions(t, ctx, store, org2, []string{"organization.create"})
}

// assertAuditActions asserts the exact ordered set of audit actions recorded
// for one organization. The query runs as the migration owner (superuser),
// which RLS exempts — audit_logs is intentionally append-only otherwise.
func assertAuditActions(t *testing.T, ctx context.Context, store *postgres.Store, orgID uuid.UUID, want []string) {
	t.Helper()
	rows, err := store.Pool().Query(ctx,
		`SELECT action FROM audit_logs WHERE organization_id = $1 ORDER BY action`, orgID)
	if err != nil {
		t.Fatalf("query audit_logs: %v", err)
	}
	defer rows.Close()
	got := []string{}
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			t.Fatalf("scan audit action: %v", err)
		}
		got = append(got, action)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate audit actions: %v", err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("audit actions for org %s:\n got %v\nwant %v", orgID, got, want)
	}
}
