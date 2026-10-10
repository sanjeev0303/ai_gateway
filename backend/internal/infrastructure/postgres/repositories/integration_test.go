// Package repositories_test exercises the postgres repository
// implementations against a real embedded PostgreSQL server: migrations,
// seed data, tenant isolation, idempotency semantics, sentinel errors, RLS,
// and append-only triggers.
package repositories_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	repos "ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
	ports "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/testutil"
)

var (
	testCtx   = context.Background()
	testStore *postgres.Store
)

// TestMain boots one embedded Postgres for the whole package (with seed
// data) and shuts it down after the run.
func TestMain(m *testing.M) {
	tp, err := testutil.StartPostgres(testCtx, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "boot embedded postgres: %v\n", err)
		os.Exit(1)
	}
	testStore = tp.Store()
	code := m.Run()
	tp.Stop()
	os.Exit(code)
}

// newTenant creates a fresh organization with an admin user and a project
// so tests never collide on unique constraints, and returns their IDs.
func newTenant(t *testing.T) (orgID, userID, projectID uuid.UUID) {
	t.Helper()
	org, err := repos.NewOrganizationRepository(testStore).Create(testCtx, domain.NewOrganization{
		Name: "Test Org",
		Slug: "org-" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	user, err := repos.NewUserRepository(testStore).Create(testCtx, domain.NewUser{
		Email:       uuid.NewString() + "@example.test",
		DisplayName: "Test User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := repos.NewOrgMemberRepository(testStore).Add(testCtx, org.ID, user.ID, domain.RoleAdmin); err != nil {
		t.Fatalf("add member: %v", err)
	}
	proj, err := repos.NewProjectRepository(testStore).Create(testCtx, org.ID, domain.NewProject{
		Name:        "Default",
		Slug:        "proj-" + uuid.NewString(),
		Description: "test project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return org.ID, user.ID, proj.ID
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireNotFound(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func requireConflict(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ports.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

// requireRawConstraint asserts a unique-constraint violation surfaced as-is
// (plain inserts that do not use ON CONFLICT DO NOTHING must NOT be mapped to
// a sentinel).
func requireRawConstraint(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected unique-violation error, got nil")
	}
	if errors.Is(err, ports.ErrConflict) || errors.Is(err, ports.ErrNotFound) || errors.Is(err, ports.ErrInvalid) || errors.Is(err, ports.ErrForbidden) {
		t.Fatalf("expected raw constraint error, got sentinel: %v", err)
	}
}

// requireJSONBytesEqual asserts two JSON documents are semantically equal.
// jsonb columns round-trip through PostgreSQL's normalized text form (e.g.
// `{"scope": "gateway"}` gains a space after the colon), so exact byte
// comparison is not valid for jsonb-backed columns.
func requireJSONBytesEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("invalid got json %q: %v", got, err)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("invalid want json %q: %v", want, err)
	}
	if !reflect.DeepEqual(g, w) {
		t.Fatalf("json mismatch: got %s, want %s", got, want)
	}
}

func TestSeedDataAndReplayIdempotency(t *testing.T) {
	acmeOrgID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	acmeUserID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	acmeProjID := uuid.MustParse("33333333-3333-4333-8333-333333333333")

	org, err := repos.NewOrganizationRepository(testStore).GetByID(testCtx, acmeOrgID)
	requireNoErr(t, err)
	if org.Slug != "acme" || org.Name != "Acme" || org.Status != domain.OrgStatusActive {
		t.Fatalf("unexpected seeded org: %+v", org)
	}

	user, err := repos.NewUserRepository(testStore).GetByEmail(testCtx, "admin@acme.example")
	requireNoErr(t, err)
	if user.ID != acmeUserID || user.Status != domain.UserStatusActive {
		t.Fatalf("unexpected seeded user: %+v", user)
	}

	members, err := repos.NewOrgMemberRepository(testStore).List(testCtx, acmeOrgID)
	requireNoErr(t, err)
	if len(members) != 1 || members[0].User.ID != acmeUserID || members[0].Role != domain.RoleAdmin {
		t.Fatalf("unexpected seeded members: %+v", members)
	}

	proj, err := repos.NewProjectRepository(testStore).Get(testCtx, acmeOrgID, acmeProjID)
	requireNoErr(t, err)
	if proj.Slug != "default" || proj.Status != domain.ProjectStatusActive {
		t.Fatalf("unexpected seeded project: %+v", proj)
	}

	// Replaying migrations applies nothing new; replaying seed must not
	// duplicate the fixed UUIDs.
	requireNoErr(t, postgres.Migrate(testCtx, testStore.Pool()))
	requireNoErr(t, postgres.Seed(testCtx, testStore.Pool()))

	var count int
	if err := testStore.Pool().QueryRow(testCtx,
		"SELECT count(*) FROM organizations WHERE slug = 'acme'").Scan(&count); err != nil {
		t.Fatalf("count acme orgs: %v", err)
	}
	if count != 1 {
		t.Fatalf("seed replay duplicated the acme org: count=%d", count)
	}
}

func TestOrganizationRepository(t *testing.T) {
	repo := repos.NewOrganizationRepository(testStore)

	org, err := repo.Create(testCtx, domain.NewOrganization{Name: "Alpha", Slug: "alpha-" + uuid.NewString()})
	requireNoErr(t, err)
	if org.Status != domain.OrgStatusActive || org.CreatedAt.IsZero() {
		t.Fatalf("unexpected created org: %+v", org)
	}

	got, err := repo.GetByID(testCtx, org.ID)
	requireNoErr(t, err)
	if got.Name != "Alpha" || got.Slug != org.Slug {
		t.Fatalf("GetByID mismatch: %+v vs %+v", got, org)
	}

	bySlug, err := repo.GetBySlug(testCtx, org.Slug)
	requireNoErr(t, err)
	if bySlug.ID != org.ID {
		t.Fatalf("GetBySlug mismatch: %+v", bySlug)
	}

	requireNotFound(t, func() error { _, err := repo.GetByID(testCtx, uuid.New()); return err }())
	requireNotFound(t, func() error { _, err := repo.GetBySlug(testCtx, "no-such-org"); return err }())

	// Duplicate slug: unique violation mapped to ErrConflict — the
	// application layer retries org provisioning on ErrConflict.
	_, err = repo.Create(testCtx, domain.NewOrganization{Name: "Dupe", Slug: org.Slug})
	requireConflict(t, err)
}

func TestUserRepository(t *testing.T) {
	repo := repos.NewUserRepository(testStore)

	email := uuid.NewString() + "@example.test"
	user, err := repo.Create(testCtx, domain.NewUser{Email: email, DisplayName: "Jane"})
	requireNoErr(t, err)
	if user.Status != domain.UserStatusActive || user.CreatedAt.IsZero() {
		t.Fatalf("unexpected created user: %+v", user)
	}

	got, err := repo.GetByID(testCtx, user.ID)
	requireNoErr(t, err)
	if got.Email != email || got.DisplayName != "Jane" {
		t.Fatalf("GetByID mismatch: %+v vs %+v", got, user)
	}

	byEmail, err := repo.GetByEmail(testCtx, strings.ToUpper(email))
	requireNoErr(t, err)
	if byEmail.ID != user.ID {
		t.Fatalf("GetByEmail is not case-insensitive: %+v", byEmail)
	}

	requireNotFound(t, func() error { _, err := repo.GetByID(testCtx, uuid.New()); return err }())
	requireNotFound(t, func() error { _, err := repo.GetByEmail(testCtx, "nobody@example.test"); return err }())

	// Duplicate email (lower-unique): unique violation mapped to
	// ErrConflict — getOrCreateUser adopts the winner's row on ErrConflict.
	_, err = repo.Create(testCtx, domain.NewUser{Email: email, DisplayName: "Janet"})
	requireConflict(t, err)
}

func TestListOrganizationsForUser(t *testing.T) {
	userRepo := repos.NewUserRepository(testStore)
	memberRepo := repos.NewOrgMemberRepository(testStore)
	orgRepo := repos.NewOrganizationRepository(testStore)

	user, err := userRepo.Create(testCtx, domain.NewUser{Email: uuid.NewString() + "@example.test", DisplayName: "Member"})
	requireNoErr(t, err)

	orgA, err := orgRepo.Create(testCtx, domain.NewOrganization{Name: "A", Slug: "a-" + uuid.NewString()})
	requireNoErr(t, err)
	orgB, err := orgRepo.Create(testCtx, domain.NewOrganization{Name: "B", Slug: "b-" + uuid.NewString()})
	requireNoErr(t, err)

	_, err = memberRepo.Add(testCtx, orgA.ID, user.ID, domain.RoleAdmin)
	requireNoErr(t, err)
	_, err = memberRepo.Add(testCtx, orgB.ID, user.ID, domain.RoleMember)
	requireNoErr(t, err)

	orgs, err := userRepo.ListOrganizationsForUser(testCtx, user.ID)
	requireNoErr(t, err)
	if len(orgs) != 2 {
		t.Fatalf("expected 2 organizations, got %d: %+v", len(orgs), orgs)
	}
	roles := map[uuid.UUID]string{}
	for _, o := range orgs {
		roles[o.Organization.ID] = o.Role
	}
	if roles[orgA.ID] != domain.RoleAdmin || roles[orgB.ID] != domain.RoleMember {
		t.Fatalf("unexpected roles: %+v", roles)
	}

	// A user with no memberships gets an empty list, not an error.
	fresh, err := userRepo.Create(testCtx, domain.NewUser{Email: uuid.NewString() + "@example.test", DisplayName: "Fresh"})
	requireNoErr(t, err)
	orgs, err = userRepo.ListOrganizationsForUser(testCtx, fresh.ID)
	requireNoErr(t, err)
	if len(orgs) != 0 {
		t.Fatalf("expected no organizations, got %+v", orgs)
	}
}

func TestOrgMemberRepository(t *testing.T) {
	memberRepo := repos.NewOrgMemberRepository(testStore)
	userRepo := repos.NewUserRepository(testStore)
	orgID, adminID, _ := newTenant(t)

	member, err := userRepo.Create(testCtx, domain.NewUser{Email: uuid.NewString() + "@example.test", DisplayName: "Bob"})
	requireNoErr(t, err)

	added, err := memberRepo.Add(testCtx, orgID, member.ID, domain.RoleMember)
	requireNoErr(t, err)
	if added.Role != domain.RoleMember || added.OrganizationID != orgID {
		t.Fatalf("unexpected added member: %+v", added)
	}

	list, err := memberRepo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 2 {
		t.Fatalf("expected 2 members, got %d: %+v", len(list), list)
	}
	byUser := map[uuid.UUID]domain.OrgMember{}
	for _, m := range list {
		byUser[m.User.ID] = m
	}
	if byUser[adminID].Role != domain.RoleAdmin || byUser[member.ID].Role != domain.RoleMember {
		t.Fatalf("unexpected roles: %+v", list)
	}
	if byUser[member.ID].MemberSince.IsZero() {
		t.Fatal("member_since not populated")
	}

	requireNoErr(t, memberRepo.Remove(testCtx, orgID, member.ID))
	requireNotFound(t, memberRepo.Remove(testCtx, orgID, member.ID))

	list, err = memberRepo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 {
		t.Fatalf("expected 1 member after remove, got %d", len(list))
	}
}

func TestProjectRepository(t *testing.T) {
	repo := repos.NewProjectRepository(testStore)
	orgID, _, _ := newTenant(t)

	proj, err := repo.Create(testCtx, orgID, domain.NewProject{
		Name:        "Analytics",
		Slug:        "analytics-" + uuid.NewString(),
		Description: "usage analytics",
	})
	requireNoErr(t, err)
	if proj.Status != domain.ProjectStatusActive {
		t.Fatalf("unexpected project: %+v", proj)
	}

	got, err := repo.Get(testCtx, orgID, proj.ID)
	requireNoErr(t, err)
	if got.Name != "Analytics" || got.Description != "usage analytics" {
		t.Fatalf("Get mismatch: %+v", got)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 2 {
		t.Fatalf("expected 2 projects (tenant default + created), got %d: %+v", len(list), list)
	}

	updated, err := repo.SetStatus(testCtx, orgID, proj.ID, domain.ProjectStatusArchived)
	requireNoErr(t, err)
	if updated.Status != domain.ProjectStatusArchived {
		t.Fatalf("SetStatus mismatch: %+v", updated)
	}

	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, uuid.New()); return err }())
	requireNotFound(t, func() error {
		_, err := repo.SetStatus(testCtx, orgID, uuid.New(), domain.ProjectStatusArchived)
		return err
	}())

	// Cross-tenant isolation: the same id must not resolve under another org.
	otherOrg, _, _ := newTenant(t)
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, proj.ID); return err }())
}

func TestTeamAndTeamMemberRepositories(t *testing.T) {
	teamRepo := repos.NewTeamRepository(testStore)
	memberRepo := repos.NewTeamMemberRepository(testStore)
	userRepo := repos.NewUserRepository(testStore)
	orgID, _, _ := newTenant(t)

	team, err := teamRepo.Create(testCtx, orgID, domain.NewTeam{Name: "ML Guild", Description: "model engineers"})
	requireNoErr(t, err)

	got, err := teamRepo.Get(testCtx, orgID, team.ID)
	requireNoErr(t, err)
	if got.Name != "ML Guild" {
		t.Fatalf("Get mismatch: %+v", got)
	}

	teams, err := teamRepo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(teams) != 1 {
		t.Fatalf("expected 1 team, got %d: %+v", len(teams), teams)
	}

	bob, err := userRepo.Create(testCtx, domain.NewUser{Email: uuid.NewString() + "@example.test", DisplayName: "Bob"})
	requireNoErr(t, err)
	added, err := memberRepo.Add(testCtx, orgID, team.ID, bob.ID, domain.RoleMember)
	requireNoErr(t, err)
	if added.OrganizationID != orgID || added.TeamID != team.ID {
		t.Fatalf("unexpected added team member: %+v", added)
	}

	view, err := memberRepo.List(testCtx, orgID, team.ID)
	requireNoErr(t, err)
	if len(view) != 1 || view[0].User.ID != bob.ID || view[0].Role != domain.RoleMember {
		t.Fatalf("unexpected team member view: %+v", view)
	}

	requireNoErr(t, memberRepo.Remove(testCtx, orgID, team.ID, bob.ID))
	requireNotFound(t, memberRepo.Remove(testCtx, orgID, team.ID, bob.ID))
	requireNotFound(t, func() error { _, err := teamRepo.Get(testCtx, orgID, uuid.New()); return err }())
}

func TestProviderAccountRepository(t *testing.T) {
	repo := repos.NewProviderAccountRepository(testStore)
	orgID, _, _ := newTenant(t)

	acct, err := repo.Create(testCtx, orgID, domain.NewProviderAccount{
		ProviderKind: domain.ProviderKindAnthropic,
		Name:         "anthropic-prod",
		BaseURL:      "https://api.anthropic.com",
	})
	requireNoErr(t, err)
	if acct.Status != domain.ProviderStatusActive {
		t.Fatalf("unexpected account: %+v", acct)
	}

	got, err := repo.Get(testCtx, orgID, acct.ID)
	requireNoErr(t, err)
	if got.BaseURL != "https://api.anthropic.com" || got.ProviderKind != domain.ProviderKindAnthropic {
		t.Fatalf("Get mismatch: %+v", got)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 {
		t.Fatalf("expected 1 account, got %d: %+v", len(list), list)
	}

	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, uuid.New()); return err }())
	otherOrg, _, _ := newTenant(t)
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, acct.ID); return err }())
}

func TestProviderCredentialRepository(t *testing.T) {
	repo := repos.NewProviderCredentialRepository(testStore)
	accountRepo := repos.NewProviderAccountRepository(testStore)
	orgID, _, _ := newTenant(t)

	acct, err := accountRepo.Create(testCtx, orgID, domain.NewProviderAccount{
		ProviderKind: domain.ProviderKindOpenAI,
		Name:         "openai-prod",
		BaseURL:      "https://api.openai.com",
	})
	requireNoErr(t, err)

	cred, err := repo.Create(testCtx, orgID, domain.NewProviderCredential{
		ProviderAccountID: acct.ID,
		Name:              "main-key",
		Kind:              domain.CredentialKindAPIKey,
		SecretRef:         "vault:openai/main",
	})
	requireNoErr(t, err)
	if cred.Status != domain.ProviderCredentialStatusActive {
		t.Fatalf("unexpected credential: %+v", cred)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 || list[0].SecretRef != "vault:openai/main" {
		t.Fatalf("unexpected credentials: %+v", list)
	}

	revoked, err := repo.Revoke(testCtx, orgID, cred.ID)
	requireNoErr(t, err)
	if revoked.Status != domain.ProviderCredentialStatusRevoked {
		t.Fatalf("Revoke mismatch: %+v", revoked)
	}
	requireNotFound(t, func() error { _, err := repo.Revoke(testCtx, orgID, uuid.New()); return err }())
}

func TestAPICredentialRepository(t *testing.T) {
	repo := repos.NewAPICredentialRepository(testStore)
	saRepo := repos.NewServiceAccountRepository(testStore)
	orgID, _, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	// Service-account-first: machine identity exists before its keys, so a
	// control-plane credential is born bound to a revocable account.
	sa, err := saRepo.Create(testCtx, orgID, domain.NewServiceAccount{
		Name:        "ci-runner",
		Description: "control-plane automation",
	})
	requireNoErr(t, err)

	// A control-scope key without a service account is rejected as invalid
	// before any write reaches Postgres.
	_, err = repo.Create(testCtx, orgID, domain.NewAPICredential{
		Name:      "unbound-control-key",
		KeyPrefix: "aigw_live_",
		KeyHash:   "sha256:" + uuid.NewString(),
		Scope:     string(domain.APICredentialScopeControl),
	})
	if !errors.Is(err, ports.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for unbound control key, got %v", err)
	}

	key, err := repo.Create(testCtx, orgID, domain.NewAPICredential{
		Name:             "control-key",
		KeyPrefix:        "aigw_live_",
		KeyHash:          "sha256:" + uuid.NewString(),
		Scope:            string(domain.APICredentialScopeControl),
		ServiceAccountID: &sa.ID,
	})
	requireNoErr(t, err)
	if key.Status != domain.APICredentialStatusActive {
		t.Fatalf("unexpected credential: %+v", key)
	}
	if key.ServiceAccountID == nil || *key.ServiceAccountID != sa.ID {
		t.Fatalf("service account not bound: %+v", key)
	}

	// Gateway-scope keys are exempt from the SA binding.
	gw, err := repo.Create(testCtx, orgID, domain.NewAPICredential{
		Name:      "gateway-key",
		KeyPrefix: "aigw_",
		KeyHash:   "sha256:" + uuid.NewString(),
		Scope:     string(domain.APICredentialScopeGateway),
	})
	requireNoErr(t, err)
	if gw.ServiceAccountID != nil {
		t.Fatalf("gateway key should be unbound: %+v", gw)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 2 || list[0].KeyHash != key.KeyHash {
		t.Fatalf("unexpected credentials: %+v", list)
	}
	other, err := repo.List(testCtx, otherOrg)
	requireNoErr(t, err)
	if len(other) != 0 {
		t.Fatalf("tenant isolation violated: %+v", other)
	}

	// ResolveByKeyHash is the cross-tenant gateway hot path: it resolves the
	// owning org via SECURITY DEFINER, so it works even though the caller
	// session is scoped to another org.
	resolved, err := repo.ResolveByKeyHash(testCtx, key.KeyHash)
	requireNoErr(t, err)
	if resolved.ID != key.ID || resolved.OrganizationID != orgID {
		t.Fatalf("ResolveByKeyHash mismatch: %+v", resolved)
	}
	if resolved.ServiceAccountID == nil || *resolved.ServiceAccountID != sa.ID {
		t.Fatalf("resolved credential lost service account binding: %+v", resolved)
	}

	// The resolver has no status filter: a revoked key still resolves, and
	// callers reject it by status.
	revoked, err := repo.Revoke(testCtx, orgID, key.ID)
	requireNoErr(t, err)
	if revoked.Status != domain.APICredentialStatusRevoked {
		t.Fatalf("Revoke mismatch: %+v", revoked)
	}
	resolved, err = repo.ResolveByKeyHash(testCtx, key.KeyHash)
	requireNoErr(t, err)
	if resolved.Status != domain.APICredentialStatusRevoked {
		t.Fatalf("expected revoked status after revoke, got %+v", resolved)
	}

	requireNotFound(t, func() error { _, err := repo.ResolveByKeyHash(testCtx, "sha256:unknown"); return err }())
	requireNotFound(t, func() error { _, err := repo.Revoke(testCtx, orgID, uuid.New()); return err }())
}

func TestGatewayCredentialRepository(t *testing.T) {
	repo := repos.NewGatewayCredentialRepository(testStore)
	orgID, userID, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	userSubject := domain.NewGatewayCredential{
		Name:        "gateway-user-key",
		KeyPrefix:   "aigw_live_",
		KeyHash:     "sha256:" + uuid.NewString(),
		SubjectType: domain.SubjectTypeUser,
		SubjectID:   userID,
		Scopes:      []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
	}
	cred, err := repo.Create(testCtx, orgID, userSubject)
	requireNoErr(t, err)
	if cred.Status != domain.GatewayCredentialStatusActive {
		t.Fatalf("unexpected credential: %+v", cred)
	}
	if cred.OrganizationID != orgID || cred.SubjectID != userID {
		t.Fatalf("credential not bound to org/subject: %+v", cred)
	}

	// Org-scoped read: Get under orgB returns ErrNotFound (org-scoped query
	// + RLS) even though the row exists under orgA.
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, cred.ID); return err }())

	got, err := repo.Get(testCtx, orgID, cred.ID)
	requireNoErr(t, err)
	if got.ID != cred.ID {
		t.Fatalf("Get mismatch: %+v", got)
	}

	// List is org-scoped: orgA sees its own row, orgB sees none.
	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 || list[0].ID != cred.ID {
		t.Fatalf("unexpected credentials: %+v", list)
	}
	other, err := repo.List(testCtx, otherOrg)
	requireNoErr(t, err)
	if len(other) != 0 {
		t.Fatalf("tenant isolation violated: %+v", other)
	}

	// Cross-org mutations are denied as if absent.
	requireNotFound(t, func() error { _, err := repo.Revoke(testCtx, otherOrg, cred.ID); return err }())
	requireNotFound(t, func() error {
		_, err := repo.UpdateStatus(testCtx, otherOrg, cred.ID, domain.GatewayCredentialStatusRevoked)
		return err
	}())

	// ResolveByKeyHash is the cross-tenant gateway hot path (SECURITY
	// DEFINER): it resolves the owning org regardless of the caller's tenant.
	resolved, err := repo.ResolveByKeyHash(testCtx, cred.KeyHash)
	requireNoErr(t, err)
	if resolved.ID != cred.ID || resolved.OrganizationID != orgID {
		t.Fatalf("ResolveByKeyHash mismatch: %+v", resolved)
	}

	// Unlike the API-credential resolver, the gateway resolver filters on
	// status: a revoked credential no longer resolves (ErrNotFound).
	revoked, err := repo.Revoke(testCtx, orgID, cred.ID)
	requireNoErr(t, err)
	if revoked.Status != domain.GatewayCredentialStatusRevoked {
		t.Fatalf("Revoke mismatch: %+v", revoked)
	}
	requireNotFound(t, func() error { _, err := repo.ResolveByKeyHash(testCtx, cred.KeyHash); return err }())

	requireNotFound(t, func() error { _, err := repo.ResolveByKeyHash(testCtx, "sha256:unknown"); return err }())

	// Rotate creates the follow-on credential (the service layer revokes the
	// old key first; this repo method only inserts the new one).
	rotateInput := userSubject
	rotateInput.KeyHash = "sha256:" + uuid.NewString()
	rotated, _, err := repo.Rotate(testCtx, orgID, cred.ID, rotateInput)
	requireNoErr(t, err)
	if rotated.KeyHash != rotateInput.KeyHash || rotated.Status != domain.GatewayCredentialStatusActive {
		t.Fatalf("unexpected rotated credential: %+v", rotated)
	}
}

func TestServiceAccountRepository(t *testing.T) {
	repo := repos.NewServiceAccountRepository(testStore)
	orgID, _, _ := newTenant(t)

	sa, err := repo.Create(testCtx, orgID, domain.NewServiceAccount{
		Name:        "ci-runner",
		Description: "runs integration tests",
	})
	requireNoErr(t, err)
	if sa.Status != domain.ServiceAccountStatusActive {
		t.Fatalf("unexpected service account: %+v", sa)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 || list[0].Name != "ci-runner" {
		t.Fatalf("unexpected service accounts: %+v", list)
	}

	revoked, err := repo.Revoke(testCtx, orgID, sa.ID)
	requireNoErr(t, err)
	if revoked.Status != domain.ServiceAccountStatusRevoked {
		t.Fatalf("Revoke mismatch: %+v", revoked)
	}
	requireNotFound(t, func() error { _, err := repo.Revoke(testCtx, orgID, uuid.New()); return err }())
}

func TestInvitationRepository(t *testing.T) {
	repo := repos.NewInvitationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	expires := time.Now().UTC().Add(72 * time.Hour)
	inv, err := repo.Create(testCtx, orgID, domain.NewInvitation{
		Email:     "new-hire@example.test",
		Role:      domain.RoleMember,
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   expires,
		InvitedBy: &userID,
	})
	requireNoErr(t, err)
	if inv.Status != domain.InvitationStatusPending || inv.OrganizationID != orgID {
		t.Fatalf("unexpected invitation: %+v", inv)
	}
	if inv.InvitedBy == nil || *inv.InvitedBy != userID {
		t.Fatalf("invited_by not recorded: %+v", inv)
	}
	if inv.ExpiresAt.IsZero() || inv.IsExpired(time.Now().UTC()) {
		t.Fatalf("expiry not persisted: %+v", inv)
	}

	// At most one pending invitation per (organization, email), matched
	// case-insensitively by the partial unique index.
	_, err = repo.Create(testCtx, orgID, domain.NewInvitation{
		Email:     "NEW-HIRE@example.test",
		Role:      domain.RoleMember,
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   expires,
	})
	requireConflict(t, err)

	// The same email may be invited by a different organization.
	other, err := repo.Create(testCtx, otherOrg, domain.NewInvitation{
		Email:     "new-hire@example.test",
		Role:      domain.RoleAdmin,
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   expires,
	})
	requireNoErr(t, err)
	if other.OrganizationID != otherOrg {
		t.Fatalf("unexpected invitation: %+v", other)
	}

	// A token hash handed out twice collides on global uniqueness regardless
	// of organization.
	_, err = repo.Create(testCtx, otherOrg, domain.NewInvitation{
		Email:     "another@example.test",
		Role:      domain.RoleMember,
		TokenHash: inv.TokenHash,
		Expires:   expires,
	})
	requireConflict(t, err)

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 || list[0].ID != inv.ID {
		t.Fatalf("unexpected invitations: %+v", list)
	}

	got, err := repo.Get(testCtx, orgID, inv.ID)
	requireNoErr(t, err)
	if got.Email != "new-hire@example.test" || got.Role != domain.RoleMember {
		t.Fatalf("Get mismatch: %+v", got)
	}

	// Get under another org returns no rows (org-scoped query + RLS).
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, inv.ID); return err }())

	// ResolveByTokenHash works across tenants via SECURITY DEFINER because
	// the accept flow only knows the token, not the org.
	resolved, err := repo.ResolveByTokenHash(testCtx, inv.TokenHash)
	requireNoErr(t, err)
	if resolved.ID != inv.ID || resolved.OrganizationID != orgID {
		t.Fatalf("ResolveByTokenHash mismatch: %+v", resolved)
	}

	accepted, err := repo.MarkAccepted(testCtx, orgID, inv.ID, userID)
	requireNoErr(t, err)
	if accepted.Status != domain.InvitationStatusAccepted || accepted.AcceptedAt == nil {
		t.Fatalf("MarkAccepted mismatch: %+v", accepted)
	}
	if accepted.AcceptedBy == nil || *accepted.AcceptedBy != userID {
		t.Fatalf("accepted_by not recorded: %+v", accepted)
	}

	// Acceptance frees the pending slot for a follow-up invitation.
	followup, err := repo.Create(testCtx, orgID, domain.NewInvitation{
		Email:     "new-hire@example.test",
		Role:      domain.RoleMember,
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   expires,
	})
	requireNoErr(t, err)

	revoked, err := repo.Revoke(testCtx, orgID, followup.ID)
	requireNoErr(t, err)
	if revoked.Status != domain.InvitationStatusRevoked {
		t.Fatalf("Revoke mismatch: %+v", revoked)
	}
	requireNotFound(t, func() error { _, err := repo.Revoke(testCtx, orgID, uuid.New()); return err }())

	// The role CHECK is enforced by the database, surfaced as a raw error.
	_, err = repo.Create(testCtx, otherOrg, domain.NewInvitation{
		Email:     "bad-role@example.test",
		Role:      "superuser",
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   expires,
	})
	requireRawConstraint(t, err)

	// Expired invitations stay pending until a resolution action observes
	// them; expiry is a derived state.
	past, err := repo.Create(testCtx, otherOrg, domain.NewInvitation{
		Email:     "stale@example.test",
		Role:      domain.RoleMember,
		TokenHash: "sha256:" + uuid.NewString(),
		Expires:   time.Now().UTC().Add(-time.Hour),
	})
	requireNoErr(t, err)
	if past.Status != domain.InvitationStatusPending || !past.IsExpired(time.Now().UTC()) {
		t.Fatalf("expected expired pending invitation: %+v", past)
	}
}

func TestProjectMemberRepository(t *testing.T) {
	repo := repos.NewProjectMemberRepository(testStore)
	orgID, userID, projectID := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	added, err := repo.Add(testCtx, orgID, projectID, userID, domain.ProjectMemberRoleMember)
	requireNoErr(t, err)
	if added.ProjectID != projectID || added.OrganizationID != orgID || added.UserID != userID || added.Role != domain.ProjectMemberRoleMember {
		t.Fatalf("unexpected member: %+v", added)
	}

	// (project_id, user_id) is the primary key.
	_, err = repo.Add(testCtx, orgID, projectID, userID, domain.ProjectMemberRoleAdmin)
	requireConflict(t, err)

	got, err := repo.Get(testCtx, orgID, projectID, userID)
	requireNoErr(t, err)
	if got.Role != domain.ProjectMemberRoleMember {
		t.Fatalf("Get mismatch: %+v", got)
	}

	// Cross-tenant lookup returns no rows (org-scoped query + FORCE RLS).
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, projectID, userID); return err }())

	view, err := repo.List(testCtx, orgID, projectID)
	requireNoErr(t, err)
	if len(view) != 1 || view[0].User.ID != userID || view[0].Role != domain.ProjectMemberRoleMember {
		t.Fatalf("unexpected member views: %+v", view)
	}
	if view[0].MemberSince.IsZero() {
		t.Fatalf("member_since not populated: %+v", view[0])
	}

	requireNoErr(t, repo.Remove(testCtx, orgID, projectID, userID))
	requireNotFound(t, repo.Remove(testCtx, orgID, projectID, userID))
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, projectID, userID); return err }())

	// The role CHECK is enforced by the database, surfaced as a raw error.
	_, err = repo.Add(testCtx, orgID, projectID, userID, "superuser")
	requireRawConstraint(t, err)
}

func TestPolicyAndRuleRepositories(t *testing.T) {
	policyRepo := repos.NewPolicyRepository(testStore)
	ruleRepo := repos.NewPolicyRuleRepository(testStore)
	orgID, _, _ := newTenant(t)

	policy, err := policyRepo.Create(testCtx, orgID, domain.NewPolicy{
		Name:        "budget-guard",
		Description: "enforce spend budgets",
		Enforcement: domain.EnforcementActive,
		Priority:    10,
	})
	requireNoErr(t, err)

	got, err := policyRepo.Get(testCtx, orgID, policy.ID)
	requireNoErr(t, err)
	if got.Enforcement != domain.EnforcementActive || got.Priority != 10 {
		t.Fatalf("Get mismatch: %+v", got)
	}



	rule, err := ruleRepo.Create(testCtx, orgID, domain.NewPolicyRule{
		PolicyID:    policy.ID,
		RuleType:    domain.RuleTypeRateLimit,
		Description: "100 rpm",
		Conditions:  []byte(`{"scope":"gateway"}`),
		Parameters:  []byte(`{"requests_per_minute":100}`),
		Enabled:     true,
	})
	requireNoErr(t, err)
	if rule.Enabled != true || rule.OrganizationID != orgID {
		t.Fatalf("unexpected rule: %+v", rule)
	}

	rules, err := ruleRepo.List(testCtx, orgID, policy.ID)
	requireNoErr(t, err)
	if len(rules) != 1 {
		t.Fatalf("unexpected rules: %+v", rules)
	}
	requireJSONBytesEqual(t, rules[0].Conditions, []byte(`{"scope":"gateway"}`))
	requireJSONBytesEqual(t, rules[0].Parameters, []byte(`{"requests_per_minute":100}`))

	requireNotFound(t, func() error { _, err := policyRepo.Get(testCtx, orgID, uuid.New()); return err }())
}

func TestLlmRequestRepository_HonorsRequestTime(t *testing.T) {
	repo := repos.NewLlmRequestRepository(testStore)
	orgID, _, _ := newTenant(t)

	requestTime := time.Date(2026, 2, 14, 3, 21, 0, 0, time.UTC)
	recorded, err := repo.Record(testCtx, orgID, domain.NewLlmRequest{
		RequestID:    "timed-" + uuid.NewString(),
		ProviderKind: domain.ProviderKindOpenAI,
		Model:        "gpt-4o",
		Status:       domain.LlmRequestStatusSucceeded,
		PromptTokens: 1, CompletionTokens: 1, TotalTokens: 2,
		CreatedAt: requestTime,
	})
	requireNoErr(t, err)
	if !recorded.CreatedAt.Equal(requestTime) {
		t.Fatalf("CreatedAt = %s, want %s", recorded.CreatedAt, requestTime)
	}

	persisted, err := repo.GetByRequestID(testCtx, orgID, recorded.RequestID)
	requireNoErr(t, err)
	if !persisted.CreatedAt.Equal(requestTime) {
		t.Fatalf("persisted created_at = %s, want %s", persisted.CreatedAt, requestTime)
	}

	zeroTime, err := repo.Record(testCtx, orgID, domain.NewLlmRequest{
		RequestID:    "untimed-" + uuid.NewString(),
		ProviderKind: domain.ProviderKindOpenAI,
		Model:        "gpt-4o",
		Status:       domain.LlmRequestStatusSucceeded,
	})
	requireNoErr(t, err)
	if zeroTime.CreatedAt.IsZero() {
		t.Fatal("a zero CreatedAt must fall back to now(), not a zero timestamp")
	}
}

func TestLlmRequestRepository(t *testing.T) {
	repo := repos.NewLlmRequestRepository(testStore)
	orgID, userID, projectID := newTenant(t)

	req, err := repo.Record(testCtx, orgID, domain.NewLlmRequest{
		ProjectID:         &projectID,
		UserID:            &userID,
		RequestID:         "req-" + uuid.NewString(),
		ProviderRequestID: ptr("prov-" + uuid.NewString()),
		ProviderKind:      domain.ProviderKindOpenAI,
		Model:             "gpt-4o",
		Status:            domain.LlmRequestStatusSucceeded,
		PromptTokens:      int32(10),
		CompletionTokens:  int32(20),
		CachedInputTokens: int32(2),
		TotalTokens:       int32(32),
		LatencyMs:         int32(150),
		CostMicros:        int64(500),
	})
	requireNoErr(t, err)
	if req.TotalTokens != 32 || req.LatencyMs != 150 {
		t.Fatalf("unexpected request: %+v", req)
	}

	// Duplicate (org, request_id): idempotent insert → ErrConflict, and the
	// original row is untouched.
	_, err = repo.Record(testCtx, orgID, domain.NewLlmRequest{
		ProjectID:    &projectID,
		RequestID:    req.RequestID,
		ProviderKind: domain.ProviderKindOpenAI,
		Model:        "gpt-4o",
		Status:       domain.LlmRequestStatusSucceeded,
	})
	requireConflict(t, err)

	got, err := repo.GetByRequestID(testCtx, orgID, req.RequestID)
	requireNoErr(t, err)
	if got.ID != req.ID {
		t.Fatalf("GetByRequestID mismatch: %+v vs %+v", got, req)
	}

	// List uses a half-open [from, to) window on created_at.
	inWin, err := repo.List(testCtx, orgID, req.CreatedAt.Add(-time.Hour), req.CreatedAt.Add(time.Hour))
	requireNoErr(t, err)
	if len(inWin) != 1 || inWin[0].ID != req.ID {
		t.Fatalf("expected request inside window, got %+v", inWin)
	}
	outWin, err := repo.List(testCtx, orgID, req.CreatedAt.Add(time.Minute), req.CreatedAt.Add(2*time.Hour))
	requireNoErr(t, err)
	if len(outWin) != 0 {
		t.Fatalf("expected no requests outside window, got %+v", outWin)
	}

	requireNotFound(t, func() error { _, err := repo.GetByRequestID(testCtx, orgID, "req-missing"); return err }())
	// Same request_id is legal in a different org. projects is FORCE RLS, so
	// the FK check must reference a project visible under otherOrg's scope.
	otherOrg, _, otherProj := newTenant(t)
	other, err := repo.Record(testCtx, otherOrg, domain.NewLlmRequest{
		ProjectID:    &otherProj,
		RequestID:    req.RequestID,
		ProviderKind: domain.ProviderKindAnthropic,
		Model:        "claude-3-5-sonnet",
		Status:       domain.LlmRequestStatusSucceeded,
	})
	requireNoErr(t, err)
	if other.ID == req.ID {
		t.Fatal("request_id collision across orgs")
	}
}

func TestUsageEventRepository(t *testing.T) {
	repo := repos.NewUsageEventRepository(testStore)
	orgID, _, _ := newTenant(t)

	reqID := "req-" + uuid.NewString()
	ev, err := repo.Record(testCtx, orgID, domain.NewUsageEvent{
		RequestID: reqID,
		EventType: domain.UsageEventTypeTokenUsage,
		Quantity:  int64(1234),
		Unit:      domain.UsageUnitTokens,
		Metadata:  nil, // must default to {}
	})
	requireNoErr(t, err)
	if ev.Metadata == nil || string(ev.Metadata) != "{}" {
		t.Fatalf("expected metadata default {}, got %q", ev.Metadata)
	}

	_, err = repo.Record(testCtx, orgID, domain.NewUsageEvent{
		RequestID: reqID,
		EventType: domain.UsageEventTypeTokenUsage,
		Quantity:  int64(999),
		Unit:      domain.UsageUnitTokens,
	})
	requireConflict(t, err)

	got, err := repo.Get(testCtx, orgID, reqID, domain.UsageEventTypeTokenUsage)
	requireNoErr(t, err)
	if got.Quantity != int64(1234) {
		t.Fatalf("Get mismatch: %+v", got)
	}

	events, err := repo.ListByRequest(testCtx, orgID, reqID)
	requireNoErr(t, err)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d: %+v", len(events), events)
	}

	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, reqID, domain.UsageEventTypeCost); return err }())
}

func TestUsageRollupRepository(t *testing.T) {
	llm := repos.NewLlmRequestRepository(testStore)
	repo := repos.NewUsageRollupRepository(testStore)
	orgID, _, projectID := newTenant(t)

	record := func(project *uuid.UUID, model string, prompt, completion, cached, total int32, cost int64) {
		t.Helper()
		_, err := llm.Record(testCtx, orgID, domain.NewLlmRequest{
			ProjectID:         project,
			RequestID:         "rollup-" + uuid.NewString(),
			ProviderKind:      "openai",
			Model:             model,
			Status:            domain.LlmRequestStatusSucceeded,
			PromptTokens:      prompt,
			CompletionTokens:  completion,
			CachedInputTokens: cached,
			TotalTokens:       total,
			CostMicros:        cost,
		})
		requireNoErr(t, err)
	}

	record(&projectID, "gpt-4o", 100, 50, 5, 155, 1000)
	record(&projectID, "gpt-4o", 40, 10, 0, 50, 250)
	record(nil, "gpt-4o", 999, 999, 0, 1998, 9999)

	from := time.Now().UTC().Add(-time.Hour)
	to := time.Now().UTC().Add(time.Hour)

	type rollup struct {
		requests   int32
		prompt     int64
		completion int64
		cached     int64
		total      int64
		cost       int64
	}
	hourly := func() rollup {
		t.Helper()
		var r rollup
		err := testStore.Pool().QueryRow(testCtx,
			`SELECT coalesce(sum(requests),0)::int, coalesce(sum(prompt_tokens),0), coalesce(sum(completion_tokens),0),
			        coalesce(sum(cached_input_tokens),0), coalesce(sum(total_tokens),0), coalesce(sum(cost_micros),0)
			 FROM usage_hourly
			 WHERE organization_id = $1 AND project_id = $2 AND model = $3`,
			orgID, projectID, "gpt-4o").
			Scan(&r.requests, &r.prompt, &r.completion, &r.cached, &r.total, &r.cost)
		requireNoErr(t, err)
		return r
	}
	rowsIn := func(table string) int {
		t.Helper()
		var n int
		err := testStore.Pool().QueryRow(testCtx,
			`SELECT count(*)::int FROM `+table+` WHERE organization_id = $1 AND project_id = $2`,
			orgID, projectID).Scan(&n)
		requireNoErr(t, err)
		return n
	}

	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))

	// The projectless request cannot be represented in a project-keyed bucket,
	// so the two scoped requests are all that may be counted.
	want := rollup{requests: 2, prompt: 140, completion: 60, cached: 5, total: 205, cost: 1250}
	if got := hourly(); got != want {
		t.Fatalf("hourly rollup mismatch: got %+v want %+v", got, want)
	}
	for _, table := range []string{"usage_daily", "usage_monthly"} {
		if n := rowsIn(table); n != 1 {
			t.Fatalf("%s expected 1 bucket row, got %d", table, n)
		}
	}

	// Recompute rebuilds its window, so a second pass must not double-count.
	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))
	if got := hourly(); got != want {
		t.Fatalf("recompute was not idempotent: got %+v want %+v", got, want)
	}

	// Deleting a project nulls its llm_requests rows, which must not leave the
	// old buckets behind: they are cleared and nothing refills them.
	if _, err := testStore.Pool().Exec(testCtx,
		`UPDATE llm_requests SET project_id = NULL WHERE organization_id = $1 AND project_id = $2`,
		orgID, projectID); err != nil {
		t.Fatalf("detach project: %v", err)
	}
	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))
	for _, table := range []string{"usage_hourly", "usage_daily", "usage_monthly"} {
		if n := rowsIn(table); n != 0 {
			t.Fatalf("%s kept %d stale bucket rows after project detach", table, n)
		}
	}
}

func TestUsageRollupCostSnapshot(t *testing.T) {
	llm := repos.NewLlmRequestRepository(testStore)
	models := repos.NewPricingModelRepository(testStore)
	versions := repos.NewPricingVersionRepository(testStore)
	repo := repos.NewUsageRollupRepository(testStore)
	orgID, _, projectID := newTenant(t)

	now := time.Now().UTC()
	pm, err := models.Upsert(testCtx, orgID, domain.NewPricingModel{
		ModelSlug: "gpt-4o", ProviderKind: "openai", DisplayName: "GPT-4o",
	})
	requireNoErr(t, err)
	for _, v := range []domain.NewPricingVersion{
		{PricingModelID: pm.ID, Version: 1, InputUnitPriceMicros: 3, OutputUnitPriceMicros: 15, EffectiveFrom: now.Add(-2 * time.Hour)},
		{PricingModelID: pm.ID, Version: 2, InputUnitPriceMicros: 4, OutputUnitPriceMicros: 20, EffectiveFrom: now.Add(-30 * time.Minute)},
	} {
		_, err := versions.Create(testCtx, orgID, v)
		requireNoErr(t, err)
	}

	record := func(requestID, model string) {
		t.Helper()
		_, err := llm.Record(testCtx, orgID, domain.NewLlmRequest{
			ProjectID: &projectID, RequestID: requestID, ProviderKind: "openai",
			Model: model, Status: domain.LlmRequestStatusSucceeded,
			PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
		})
		requireNoErr(t, err)
	}
	record("priced-v2", "gpt-4o")
	record("priced-v1", "gpt-4o")
	record("no-price-sheet", "mystery-model")

	if _, err := testStore.Pool().Exec(testCtx,
		`UPDATE llm_requests SET created_at = $1 WHERE organization_id = $2 AND request_id = $3`,
		now.Add(-90*time.Minute), orgID, "priced-v1"); err != nil {
		t.Fatalf("backdate request: %v", err)
	}

	costOf := func(requestID string) int64 {
		t.Helper()
		var c int64
		err := testStore.Pool().QueryRow(testCtx,
			`SELECT cost_micros FROM llm_requests WHERE organization_id = $1 AND request_id = $2`,
			orgID, requestID).Scan(&c)
		requireNoErr(t, err)
		return c
	}
	bucketCost := func() int64 {
		t.Helper()
		var c int64
		err := testStore.Pool().QueryRow(testCtx,
			`SELECT coalesce(sum(cost_micros),0) FROM usage_hourly
			 WHERE organization_id = $1 AND project_id = $2 AND model = 'gpt-4o'`,
			orgID, projectID).Scan(&c)
		requireNoErr(t, err)
		return c
	}

	from, to := now.Add(-3*time.Hour), now.Add(time.Hour)
	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))

	for requestID, want := range map[string]int64{"priced-v2": 1400, "priced-v1": 1050, "no-price-sheet": 0} {
		if got := costOf(requestID); got != want {
			t.Fatalf("%s cost_micros = %d, want %d", requestID, got, want)
		}
	}
	if got, want := bucketCost(), int64(2450); got != want {
		t.Fatalf("hourly cost = %d, want %d", got, want)
	}

	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))
	if got, want := bucketCost(), int64(2450); got != want {
		t.Fatalf("cost was not idempotent: %d, want %d", got, want)
	}

	_, err = versions.Create(testCtx, orgID, domain.NewPricingVersion{
		PricingModelID: pm.ID, Version: 3, InputUnitPriceMicros: 99, OutputUnitPriceMicros: 99,
		EffectiveFrom: now,
	})
	requireNoErr(t, err)
	requireNoErr(t, repo.Recompute(testCtx, orgID, from, to))
	if got := costOf("priced-v1"); got != 1050 {
		t.Fatalf("historical request repriced to v3: %d", got)
	}
}

func TestPricingVersionTieBreakOnEqualEffectiveFrom(t *testing.T) {
	llm := repos.NewLlmRequestRepository(testStore)
	models := repos.NewPricingModelRepository(testStore)
	versions := repos.NewPricingVersionRepository(testStore)
	repo := repos.NewUsageRollupRepository(testStore)
	orgID, _, _ := newTenant(t)

	now := time.Now().UTC()
	pm, err := models.Upsert(testCtx, orgID, domain.NewPricingModel{
		ModelSlug: "tiebreak-model", ProviderKind: "openai", DisplayName: "Tiebreak",
	})
	requireNoErr(t, err)

	effectiveFrom := now.Add(-time.Hour)
	highestVersionFirst := []domain.NewPricingVersion{
		{PricingModelID: pm.ID, Version: 2, InputUnitPriceMicros: 10, OutputUnitPriceMicros: 20, EffectiveFrom: effectiveFrom},
		{PricingModelID: pm.ID, Version: 1, InputUnitPriceMicros: 2, OutputUnitPriceMicros: 4, EffectiveFrom: effectiveFrom},
	}
	for _, v := range highestVersionFirst {
		_, err := versions.Create(testCtx, orgID, v)
		requireNoErr(t, err)
	}

	_, err = llm.Record(testCtx, orgID, domain.NewLlmRequest{
		RequestID: "tiebreak-request", ProviderKind: "openai", Model: "tiebreak-model",
		Status: domain.LlmRequestStatusSucceeded, PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150,
	})
	requireNoErr(t, err)

	requireNoErr(t, repo.Recompute(testCtx, orgID, now.Add(-3*time.Hour), now.Add(time.Hour)))

	var cost int64
	err = testStore.Pool().QueryRow(testCtx,
		`SELECT cost_micros FROM llm_requests WHERE organization_id = $1 AND request_id = $2`,
		orgID, "tiebreak-request").Scan(&cost)
	requireNoErr(t, err)

	if want := int64(2000); cost != want {
		t.Fatalf("cost_micros = %d, want %d (higher version must win an equal effective_from tie)", cost, want)
	}
}

func TestIdempotencyRepository(t *testing.T) {
	repo := repos.NewIdempotencyRepository(testStore)
	orgID, _, _ := newTenant(t)

	expires := time.Now().UTC().Add(time.Hour)
	key, err := repo.Create(testCtx, orgID, domain.NewIdempotencyKey{
		ScopeKey:  "billing",
		Key:       "idem-" + uuid.NewString(),
		ExpiresAt: expires,
	})
	requireNoErr(t, err)
	if key.ExpiresAt.IsZero() {
		t.Fatalf("unexpected key: %+v", key)
	}

	_, err = repo.Create(testCtx, orgID, domain.NewIdempotencyKey{
		ScopeKey:  "billing",
		Key:       key.Key,
		ExpiresAt: expires,
	})
	requireConflict(t, err)

	got, err := repo.Get(testCtx, orgID, "billing", key.Key)
	requireNoErr(t, err)
	if got.ID != key.ID {
		t.Fatalf("Get mismatch: %+v", got)
	}

	// Same key under a different scope is a different idempotency domain.
	other, err := repo.Create(testCtx, orgID, domain.NewIdempotencyKey{
		ScopeKey:  "chat",
		Key:       key.Key,
		ExpiresAt: expires,
	})
	requireNoErr(t, err)
	if other.ID == key.ID {
		t.Fatal("idempotency key collision across scopes")
	}

	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, "billing", "idem-missing"); return err }())
}

func TestPlanRepository(t *testing.T) {
	repo := repos.NewPlanRepository(testStore)

	plan, err := repo.Create(testCtx, domain.NewPlan{
		Code:        "plan-" + uuid.NewString(),
		Name:        "Scale",
		Description: "scale tier",
	})
	requireNoErr(t, err)
	if plan.Status != domain.PlanStatusActive {
		t.Fatalf("unexpected plan: %+v", plan)
	}

	plans, err := repo.List(testCtx)
	requireNoErr(t, err)
	found := false
	for _, p := range plans {
		if p.ID == plan.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created plan not listed: %+v", plans)
	}

	// Plans are global: duplicate code is a plain unique violation.
	_, err = repo.Create(testCtx, domain.NewPlan{Code: plan.Code, Name: "Dupe", Description: ""})
	requireRawConstraint(t, err)
}

func TestSubscriptionRepository(t *testing.T) {
	repo := repos.NewSubscriptionRepository(testStore)
	planRepo := repos.NewPlanRepository(testStore)
	orgID, _, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	plan, err := planRepo.Create(testCtx, domain.NewPlan{Code: "plan-" + uuid.NewString(), Name: "Scale", Description: ""})
	requireNoErr(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	sub, err := repo.Create(testCtx, orgID, domain.NewSubscription{
		PlanID:             plan.ID,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   ptr(now.Add(30 * 24 * time.Hour)),
		CancelAtPeriodEnd:  false,
	})
	requireNoErr(t, err)
	if sub.Status != domain.SubscriptionStatusActive {
		t.Fatalf("unexpected subscription: %+v", sub)
	}

	// One active subscription per org: the second plain insert violates the
	// partial unique index and is surfaced raw (not ErrConflict).
	_, err = repo.Create(testCtx, orgID, domain.NewSubscription{PlanID: plan.ID})
	requireRawConstraint(t, err)

	got, err := repo.Get(testCtx, orgID, sub.ID)
	requireNoErr(t, err)
	if got.PlanID != plan.ID || got.CancelAtPeriodEnd {
		t.Fatalf("Get mismatch: %+v", got)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 {
		t.Fatalf("expected 1 subscription, got %d: %+v", len(list), list)
	}

	// Cross-tenant isolation.
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, sub.ID); return err }())
}

func TestBillingEventRepository(t *testing.T) {
	repo := repos.NewBillingEventRepository(testStore)
	orgID, _, _ := newTenant(t)

	amount := int64(42000000) // $420.00 in micros
	ev, err := repo.Record(testCtx, orgID, domain.NewBillingEvent{
		BillingEventID: "evt-" + uuid.NewString(),
		EventType:      "invoice.finalized",
		AmountMicros:   &amount,
		Currency:       "USD",
		Payload:        []byte(`{"line_items":3}`),
	})
	requireNoErr(t, err)
	if ev.Status != domain.BillingEventStatusPending {
		t.Fatalf("unexpected billing event: %+v", ev)
	}

	_, err = repo.Record(testCtx, orgID, domain.NewBillingEvent{
		BillingEventID: ev.BillingEventID,
		EventType:      "invoice.finalized",
		Currency:       "USD",
	})
	requireConflict(t, err)

	got, err := repo.Get(testCtx, orgID, ev.BillingEventID)
	requireNoErr(t, err)
	if got.AmountMicros == nil || *got.AmountMicros != amount {
		t.Fatalf("Get mismatch: %+v", got)
	}

	requireNotFound(t, func() error { _, err := repo.Get(testCtx, orgID, "evt-missing"); return err }())
}

func TestPricingModelAndVersionRepositories(t *testing.T) {
	modelRepo := repos.NewPricingModelRepository(testStore)
	versionRepo := repos.NewPricingVersionRepository(testStore)
	orgID, _, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)

	m1, err := modelRepo.Upsert(testCtx, orgID, domain.NewPricingModel{
		ModelSlug:    "gpt-4o",
		ProviderKind: domain.ProviderKindOpenAI,
		DisplayName:  "GPT-4o",
		Currency:     "USD",
	})
	requireNoErr(t, err)
	m2, err := modelRepo.Upsert(testCtx, orgID, domain.NewPricingModel{
		ModelSlug:    "gpt-4o",
		ProviderKind: domain.ProviderKindOpenAI,
		DisplayName:  "GPT-4o (updated)",
		Currency:     "USD",
	})
	requireNoErr(t, err)
	if m1.ID != m2.ID {
		t.Fatalf("upsert should update in place: %v vs %v", m1.ID, m2.ID)
	}
	models, err := modelRepo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(models) != 1 || models[0].DisplayName != "GPT-4o (updated)" {
		t.Fatalf("unexpected pricing models: %+v", models)
	}

	// Same model slug is fine in a different org (org-scoped key).
	om, err := modelRepo.Upsert(testCtx, otherOrg, domain.NewPricingModel{
		ModelSlug:    "gpt-4o",
		ProviderKind: domain.ProviderKindOpenAI,
		DisplayName:  "GPT-4o",
		Currency:     "USD",
	})
	requireNoErr(t, err)
	if om.ID == m1.ID {
		t.Fatal("pricing model collided across orgs")
	}

	v1, err := versionRepo.Create(testCtx, orgID, domain.NewPricingVersion{
		PricingModelID:        m1.ID,
		Version:               int32(1),
		InputUnitPriceMicros:  int64(250000),
		OutputUnitPriceMicros: int64(1000000),
		EffectiveFrom:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	requireNoErr(t, err)
	v2, err := versionRepo.Create(testCtx, orgID, domain.NewPricingVersion{
		PricingModelID:        m1.ID,
		Version:               int32(2),
		InputUnitPriceMicros:  int64(200000),
		OutputUnitPriceMicros: int64(800000),
		EffectiveFrom:         time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	})
	requireNoErr(t, err)

	versions, err := versionRepo.List(testCtx, orgID, m1.ID)
	requireNoErr(t, err)
	if len(versions) != 2 || versions[0].ID != v2.ID || versions[1].ID != v1.ID {
		t.Fatalf("expected versions ordered by effective_from DESC: %+v", versions)
	}
}

func TestProviderHealthRepository(t *testing.T) {
	repo := repos.NewProviderHealthRepository(testStore)
	checkedAt := time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)

	// Get on an empty table maps to ErrNotFound.
	requireNotFound(t, func() error {
		_, err := repo.Get(testCtx, domain.ProviderKindAnthropic)
		return err
	}())

	// Upsert inserts the first snapshot, keyed globally by provider kind.
	err := repo.Upsert(testCtx, domain.ProviderHealthState{
		ProviderKind: domain.ProviderKindAnthropic,
		Status:       domain.ProviderHealthHealthy,
		CheckedAt:    checkedAt,
	})
	requireNoErr(t, err)

	got, err := repo.Get(testCtx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	if got.Status != domain.ProviderHealthHealthy || got.ProviderKind != domain.ProviderKindAnthropic || !got.CheckedAt.Equal(checkedAt) {
		t.Fatalf("unexpected health state: %+v", got)
	}

	// Upsert on the same kind updates the row in place (PK provider_kind).
	updatedAt := checkedAt.Add(time.Minute)
	reason := "health_check_failed"
	err = repo.Upsert(testCtx, domain.ProviderHealthState{
		ProviderKind: domain.ProviderKindAnthropic,
		Status:       domain.ProviderHealthUnhealthy,
		StatusReason: reason,
		LastError:    "dial tcp: refused",
		CheckedAt:    updatedAt,
	})
	requireNoErr(t, err)

	got, err = repo.Get(testCtx, domain.ProviderKindAnthropic)
	requireNoErr(t, err)
	if got.Status != domain.ProviderHealthUnhealthy || got.StatusReason != reason || !got.CheckedAt.Equal(updatedAt) {
		t.Fatalf("expected in-place update, got %+v", got)
	}

	// A second kind coexists; List returns one row per kind.
	err = repo.Upsert(testCtx, domain.ProviderHealthState{
		ProviderKind: domain.ProviderKindOpenAI,
		Status:       domain.ProviderHealthDegraded,
		CheckedAt:    checkedAt,
	})
	requireNoErr(t, err)

	states, err := repo.List(testCtx)
	requireNoErr(t, err)
	if len(states) != 2 {
		t.Fatalf("expected 2 health states, got %+v", states)
	}
}

func TestAlertRepository(t *testing.T) {
	repo := repos.NewAlertRepository(testStore)
	orgID, _, _ := newTenant(t)

	alert, err := repo.Create(testCtx, orgID, domain.NewAlert{
		AlertType:     domain.AlertTypeSpendLimit,
		Name:          "monthly budget",
		Severity:      domain.AlertSeverityWarning,
		Configuration: []byte(`{"threshold_micros":1000000}`),
	})
	requireNoErr(t, err)
	if alert.Status != domain.AlertStatusActive {
		t.Fatalf("unexpected alert: %+v", alert)
	}

	time.Sleep(2 * time.Millisecond) // guarantee a distinct updated_at
	muted, err := repo.Update(testCtx, orgID, alert.ID, domain.AlertUpdate{
		Name:          alert.Name,
		Severity:      alert.Severity,
		Configuration: alert.Configuration,
		Status:        domain.AlertStatusMuted,
	})
	requireNoErr(t, err)
	if muted.Status != domain.AlertStatusMuted || !muted.UpdatedAt.After(muted.CreatedAt) {
		t.Fatalf("Update mismatch: %+v", muted)
	}

	list, err := repo.List(testCtx, orgID)
	requireNoErr(t, err)
	if len(list) != 1 || list[0].ID != alert.ID {
		t.Fatalf("unexpected alerts: %+v", list)
	}

	requireNotFound(t, func() error {
		_, err := repo.Update(testCtx, orgID, uuid.New(), domain.AlertUpdate{
			Name:          "missing",
			Severity:      domain.AlertSeverityWarning,
			Configuration: []byte(`{}`),
			Status:        domain.AlertStatusMuted,
		})
		return err
	}())
}

func TestAuditLogAppendOnly(t *testing.T) {
	repo := repos.NewAuditLogRepository(testStore)
	orgID, userID, _ := newTenant(t)

	log, err := repo.Append(testCtx, orgID, domain.NewAuditLog{
		ActorUserID:  &userID,
		Action:       "organization.update",
		ResourceType: "organization",
		ResourceID:   &orgID,
		Before:       []byte(`{"status":"active"}`),
		After:        []byte(`{"status":"suspended"}`),
		IPAddress:    ptr("203.0.113.7"),
		RequestID:    ptr("req-" + uuid.NewString()),
	})
	requireNoErr(t, err)
	if log.ID == uuid.Nil || log.CreatedAt.IsZero() {
		t.Fatalf("unexpected audit log: %+v", log)
	}
	if log.ActorUserID == nil || *log.ActorUserID != userID {
		t.Fatalf("actor mismatch: %+v", log)
	}

	// The row is real and queryable directly.
	var count int
	err = testStore.Pool().QueryRow(testCtx,
		`SELECT count(*) FROM audit_logs WHERE organization_id = $1 AND id = $2`,
		orgID, log.ID).Scan(&count)
	requireNoErr(t, err)
	if count != 1 {
		t.Fatalf("audit row missing: count=%d", count)
	}

	// Append-only: UPDATE and DELETE are rejected by the trigger.
	_, err = testStore.Pool().Exec(testCtx, `UPDATE audit_logs SET action = 'tampered' WHERE id = $1`, log.ID)
	if err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("expected UPDATE to be blocked by append-only trigger, got %v", err)
	}
	_, err = testStore.Pool().Exec(testCtx, `DELETE FROM audit_logs WHERE id = $1`, log.ID)
	if err == nil || !strings.Contains(err.Error(), "append-only") {
		t.Fatalf("expected DELETE to be blocked by append-only trigger, got %v", err)
	}
}

// TestRLSEnforcesTenantScope proves the FORCE RLS policies work for a
// non-superuser role: without app.org_id nothing is visible, and setting the
// GUC reveals only that organization's rows. The app's own sessions run as
// the migration owner (superuser) and are exempt from RLS, so repository
// logic is additionally guarded by orgID-scoped queries; this test verifies
// the defense-in-depth policy itself.
func TestRLSEnforcesTenantScope(t *testing.T) {
	orgID, userID, projectID := newTenant(t)
	otherOrg, otherUserID, otherProjID := newTenant(t)

	// Seed one gateway credential per org so the 00021 policy on
	// gateway_credentials is exercised alongside the 00012 baseline policies.
	gwRepo := repos.NewGatewayCredentialRepository(testStore)
	credA, err := gwRepo.Create(testCtx, orgID, domain.NewGatewayCredential{
		Name:        "org-a-key",
		KeyPrefix:   "aigw_live_",
		KeyHash:     "sha256:" + uuid.NewString(),
		SubjectType: domain.SubjectTypeUser,
		SubjectID:   userID,
		Scopes:      []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
	})
	requireNoErr(t, err)
	credB, err := gwRepo.Create(testCtx, otherOrg, domain.NewGatewayCredential{
		Name:        "org-b-key",
		KeyPrefix:   "aigw_live_",
		KeyHash:     "sha256:" + uuid.NewString(),
		SubjectType: domain.SubjectTypeUser,
		SubjectID:   otherUserID,
		Scopes:      []domain.GatewayCredentialScope{domain.GatewayCredentialScopeGateway},
	})
	requireNoErr(t, err)

	role := "aigw_rls_viewer_" + uuid.NewString()[:8]
	conn, err := testStore.Pool().Acquire(testCtx)
	if err != nil {
		t.Fatalf("acquire conn: %v", err)
	}
	defer conn.Release()
	t.Cleanup(func() {
		// Runs after the test body returns (and the conn is released), so go
		// through the pool rather than the stale *pgxpool.Conn.
		_, _ = testStore.Pool().Exec(testCtx, "DROP ROLE IF EXISTS "+role)
	})

	_, err = conn.Exec(testCtx, "CREATE ROLE "+role+" NOLOGIN")
	if err != nil {
		t.Fatalf("create role: %v", err)
	}
	_, err = conn.Exec(testCtx, "GRANT USAGE ON SCHEMA public TO "+role)
	requireNoErr(t, err)
	_, err = conn.Exec(testCtx, "GRANT SELECT ON ALL TABLES IN SCHEMA public TO "+role)
	requireNoErr(t, err)
	_, err = conn.Exec(testCtx, "GRANT USAGE ON SCHEMA app TO "+role)
	requireNoErr(t, err)
	_, err = conn.Exec(testCtx, "GRANT EXECUTE ON FUNCTION app.current_org_id() TO "+role)
	requireNoErr(t, err)

	_, err = conn.Exec(testCtx, "SET ROLE "+role)
	requireNoErr(t, err)
	defer conn.Exec(testCtx, "RESET ROLE") //nolint:errcheck // best-effort before Release

	scanCount := func(table string) int {
		t.Helper()
		var n int
		if err := conn.QueryRow(testCtx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("count %s as %s: %v", table, role, err)
		}
		return n
	}

	// No GUC: current_org_id() is NULL, so the USING clause matches nothing.
	if n := scanCount("projects"); n != 0 {
		t.Fatalf("expected 0 rows without app.org_id, got %d", n)
	}
	if n := scanCount("gateway_credentials"); n != 0 {
		t.Fatalf("expected 0 gateway_credentials without app.org_id, got %d", n)
	}

	// Scoped to orgA: only orgA's project is visible.
	_, err = conn.Exec(testCtx, "SET app.org_id = '"+orgID.String()+"'")
	requireNoErr(t, err)
	if n := scanCount("projects"); n != 1 {
		t.Fatalf("expected 1 row for org, got %d", n)
	}
	if n := scanCount("gateway_credentials"); n != 1 {
		t.Fatalf("expected 1 gateway_credential for org, got %d", n)
	}
	var visible uuid.UUID
	if err := conn.QueryRow(testCtx, "SELECT id FROM projects").Scan(&visible); err != nil {
		t.Fatalf("read project id: %v", err)
	}
	if visible != projectID {
		t.Fatalf("RLS returned wrong row: %s", visible)
	}
	var visibleCred uuid.UUID
	if err := conn.QueryRow(testCtx, "SELECT id FROM gateway_credentials").Scan(&visibleCred); err != nil {
		t.Fatalf("read gateway credential id: %v", err)
	}
	if visibleCred != credA.ID {
		t.Fatalf("RLS returned wrong gateway credential: %s", visibleCred)
	}

	// Scoped to orgB: orgA's rows disappear; only orgB's own project is
	// visible.
	_, err = conn.Exec(testCtx, "SET app.org_id = '"+otherOrg.String()+"'")
	requireNoErr(t, err)
	if n := scanCount("projects"); n != 1 {
		t.Fatalf("expected 1 row for other org, got %d", n)
	}
	if n := scanCount("gateway_credentials"); n != 1 {
		t.Fatalf("expected 1 gateway_credential for other org, got %d", n)
	}
	var visibleB uuid.UUID
	if err := conn.QueryRow(testCtx, "SELECT id FROM projects").Scan(&visibleB); err != nil {
		t.Fatalf("read project id: %v", err)
	}
	if visibleB != otherProjID {
		t.Fatalf("RLS returned wrong row: %s", visibleB)
	}
	var visibleCredB uuid.UUID
	if err := conn.QueryRow(testCtx, "SELECT id FROM gateway_credentials").Scan(&visibleCredB); err != nil {
		t.Fatalf("read gateway credential id: %v", err)
	}
	if visibleCredB != credB.ID {
		t.Fatalf("RLS returned wrong gateway credential: %s", visibleCredB)
	}
}

func ptr[T any](v T) *T { return &v }
