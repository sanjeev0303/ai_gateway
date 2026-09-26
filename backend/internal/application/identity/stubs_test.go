package identity

// White-box unit-test scaffolding for the identity service: stub repo and
// token ports wired against a fixed clock. Stubs embed their port interface so
// tests preregister only the methods a path exercises; an unregistered call
// panics, turning an unexpected repository interaction into a loud failure.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

var (
	testUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testOrgID     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testAcctID    = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	testTeamID    = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	testProjID    = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	testInviteID  = uuid.MustParse("00000000-0000-0000-0000-000000000006")
	testOtherID   = uuid.MustParse("00000000-0000-0000-0000-000000000007")
	testNow       = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	testInviteTTL = 8 * time.Hour
)

func notConfigured(method string) {
	panic("identity test: " + method + " called without a stub")
}

// testStubs bundles every stub a Service depends on.
type testStubs struct {
	orgs        *stubOrgs
	users       *stubUsers
	members     *stubMembers
	invitations *stubInvitations
	projects    *stubProjects
	teams       *stubTeams
	teamMembers *stubTeamMembers
	projMembers *stubProjMembers
	accounts    *stubAccounts
	credentials *stubCredentials
	audits      *stubAudits
	issuer      *stubIssuer
	verifier    *stubVerifier
	hasher      *stubHasher
}

// newTestService wires a Service against the stubs with a fixed clock and a
// non-default invite TTL so the defaulting path in NewService stays distinct.
func newTestService() (*Service, *testStubs) {
	st := &testStubs{
		orgs:        &stubOrgs{},
		users:       &stubUsers{},
		members:     &stubMembers{},
		invitations: &stubInvitations{},
		projects:    &stubProjects{},
		teams:       &stubTeams{},
		teamMembers: &stubTeamMembers{},
		projMembers: &stubProjMembers{},
		accounts:    &stubAccounts{},
		credentials: &stubCredentials{},
		audits:      &stubAudits{},
		issuer:      &stubIssuer{},
		verifier:    &stubVerifier{},
		hasher:      &stubHasher{},
	}
	svc := NewService(ServiceDeps{
		Orgs:        st.orgs,
		Users:       st.users,
		Members:     st.members,
		Invitations: st.invitations,
		Projects:    st.projects,
		Teams:       st.teams,
		TeamMembers: st.teamMembers,
		ProjMembers: st.projMembers,
		Accounts:    st.accounts,
		Credentials: st.credentials,
		Audits:      st.audits,
		Issuer:      st.issuer,
		Verifier:    st.verifier,
		Hasher:      st.hasher,
		Now:         func() time.Time { return testNow },
		InviteTTL:   testInviteTTL,
	})
	return svc, st
}

// userActor builds a user-kind actor for the fixed test user and org.
func userActor(role string) domain.Actor {
	return domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         testUserID,
		OrganizationID: testOrgID,
		Role:           role,
		DisplayName:    "Test User",
	}
}

func testServiceActor() domain.Actor {
	return domain.ServiceActor(testOrgID, testAcctID, "ci-bot")
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireErrIs(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func requireEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func requireAudit(t *testing.T, st *testStubs, action string) {
	t.Helper()
	for _, a := range st.audits.actions {
		if a == action {
			return
		}
	}
	t.Fatalf("audit actions = %v, missing %q", st.audits.actions, action)
}

type stubOrgs struct {
	repositories.OrganizationRepository
	create func(context.Context, domain.NewOrganization) (domain.Organization, error)
}

func (s *stubOrgs) Create(ctx context.Context, in domain.NewOrganization) (domain.Organization, error) {
	if s.create == nil {
		notConfigured("stubOrgs.Create")
	}
	return s.create(ctx, in)
}

type stubUsers struct {
	repositories.UserRepository
	create     func(context.Context, domain.NewUser) (domain.User, error)
	getByID    func(context.Context, uuid.UUID) (domain.User, error)
	getByEmail func(context.Context, string) (domain.User, error)
	listOrgs   func(context.Context, uuid.UUID) ([]domain.UserOrganization, error)
}

func (s *stubUsers) Create(ctx context.Context, in domain.NewUser) (domain.User, error) {
	if s.create == nil {
		notConfigured("stubUsers.Create")
	}
	return s.create(ctx, in)
}

func (s *stubUsers) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	if s.getByID == nil {
		notConfigured("stubUsers.GetByID")
	}
	return s.getByID(ctx, id)
}

func (s *stubUsers) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if s.getByEmail == nil {
		notConfigured("stubUsers.GetByEmail")
	}
	return s.getByEmail(ctx, email)
}

func (s *stubUsers) ListOrganizationsForUser(ctx context.Context, id uuid.UUID) ([]domain.UserOrganization, error) {
	if s.listOrgs == nil {
		notConfigured("stubUsers.ListOrganizationsForUser")
	}
	return s.listOrgs(ctx, id)
}

type stubMembers struct {
	repositories.OrgMemberRepository
	add         func(context.Context, uuid.UUID, uuid.UUID, string) (domain.OrganizationMember, error)
	get         func(context.Context, uuid.UUID, uuid.UUID) (domain.OrgMember, error)
	remove      func(context.Context, uuid.UUID, uuid.UUID) error
	setRole     func(context.Context, uuid.UUID, uuid.UUID, string) (domain.OrganizationMember, error)
	countByRole func(context.Context, uuid.UUID, string) (int64, error)
	list        func(context.Context, uuid.UUID) ([]domain.OrgMember, error)
}

func (s *stubMembers) Add(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
	if s.add == nil {
		notConfigured("stubMembers.Add")
	}
	return s.add(ctx, orgID, userID, role)
}

func (s *stubMembers) Get(ctx context.Context, orgID, userID uuid.UUID) (domain.OrgMember, error) {
	if s.get == nil {
		notConfigured("stubMembers.Get")
	}
	return s.get(ctx, orgID, userID)
}

func (s *stubMembers) Remove(ctx context.Context, orgID, userID uuid.UUID) error {
	if s.remove == nil {
		notConfigured("stubMembers.Remove")
	}
	return s.remove(ctx, orgID, userID)
}

func (s *stubMembers) SetRole(ctx context.Context, orgID, userID uuid.UUID, role string) (domain.OrganizationMember, error) {
	if s.setRole == nil {
		notConfigured("stubMembers.SetRole")
	}
	return s.setRole(ctx, orgID, userID, role)
}

func (s *stubMembers) CountByRole(ctx context.Context, orgID uuid.UUID, role string) (int64, error) {
	if s.countByRole == nil {
		notConfigured("stubMembers.CountByRole")
	}
	return s.countByRole(ctx, orgID, role)
}

func (s *stubMembers) List(ctx context.Context, orgID uuid.UUID) ([]domain.OrgMember, error) {
	if s.list == nil {
		notConfigured("stubMembers.List")
	}
	return s.list(ctx, orgID)
}

type stubInvitations struct {
	repositories.InvitationRepository
	create        func(context.Context, uuid.UUID, domain.NewInvitation) (domain.Invitation, error)
	resolveByHash func(context.Context, string) (domain.Invitation, error)
	markAccepted  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.Invitation, error)
}

func (s *stubInvitations) Create(ctx context.Context, orgID uuid.UUID, in domain.NewInvitation) (domain.Invitation, error) {
	if s.create == nil {
		notConfigured("stubInvitations.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubInvitations) ResolveByTokenHash(ctx context.Context, hash string) (domain.Invitation, error) {
	if s.resolveByHash == nil {
		notConfigured("stubInvitations.ResolveByTokenHash")
	}
	return s.resolveByHash(ctx, hash)
}

func (s *stubInvitations) MarkAccepted(ctx context.Context, orgID, id, acceptedBy uuid.UUID) (domain.Invitation, error) {
	if s.markAccepted == nil {
		notConfigured("stubInvitations.MarkAccepted")
	}
	return s.markAccepted(ctx, orgID, id, acceptedBy)
}

type stubProjects struct {
	repositories.ProjectRepository
	create func(context.Context, uuid.UUID, domain.NewProject) (domain.Project, error)
}

func (s *stubProjects) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProject) (domain.Project, error) {
	if s.create == nil {
		notConfigured("stubProjects.Create")
	}
	return s.create(ctx, orgID, in)
}

type stubTeams struct {
	repositories.TeamRepository
	create func(context.Context, uuid.UUID, domain.NewTeam) (domain.Team, error)
}

func (s *stubTeams) Create(ctx context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
	if s.create == nil {
		notConfigured("stubTeams.Create")
	}
	return s.create(ctx, orgID, in)
}

type stubTeamMembers struct {
	repositories.TeamMemberRepository
	add func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (domain.TeamMember, error)
}

func (s *stubTeamMembers) Add(ctx context.Context, orgID, teamID, userID uuid.UUID, role string) (domain.TeamMember, error) {
	if s.add == nil {
		notConfigured("stubTeamMembers.Add")
	}
	return s.add(ctx, orgID, teamID, userID, role)
}

type stubProjMembers struct {
	repositories.ProjectMemberRepository
	add func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (domain.ProjectMember, error)
}

func (s *stubProjMembers) Add(ctx context.Context, orgID, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error) {
	if s.add == nil {
		notConfigured("stubProjMembers.Add")
	}
	return s.add(ctx, orgID, projectID, userID, role)
}

type stubAccounts struct {
	repositories.ServiceAccountRepository
	create func(context.Context, uuid.UUID, domain.NewServiceAccount) (domain.ServiceAccount, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.ServiceAccount, error)
}

func (s *stubAccounts) Create(ctx context.Context, orgID uuid.UUID, in domain.NewServiceAccount) (domain.ServiceAccount, error) {
	if s.create == nil {
		notConfigured("stubAccounts.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubAccounts) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
	if s.get == nil {
		notConfigured("stubAccounts.Get")
	}
	return s.get(ctx, orgID, id)
}

type stubCredentials struct {
	repositories.APICredentialRepository
	create        func(context.Context, uuid.UUID, domain.NewAPICredential) (domain.APICredential, error)
	resolveByHash func(context.Context, string) (domain.APICredential, error)
}

func (s *stubCredentials) Create(ctx context.Context, orgID uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
	if s.create == nil {
		notConfigured("stubCredentials.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubCredentials) ResolveByKeyHash(ctx context.Context, hash string) (domain.APICredential, error) {
	if s.resolveByHash == nil {
		notConfigured("stubCredentials.ResolveByKeyHash")
	}
	return s.resolveByHash(ctx, hash)
}

// stubAudits records every append and defaults to success.
type stubAudits struct {
	repositories.AuditLogRepository
	appendFn func(context.Context, uuid.UUID, domain.NewAuditLog) (domain.AuditLog, error)
	actions  []string
}

func (s *stubAudits) Append(ctx context.Context, orgID uuid.UUID, in domain.NewAuditLog) (domain.AuditLog, error) {
	if s.appendFn != nil {
		return s.appendFn(ctx, orgID, in)
	}
	s.actions = append(s.actions, in.Action)
	return domain.AuditLog{}, nil
}

type stubIssuer struct {
	ports.TokenIssuer
	issue func(context.Context, domain.TokenClaims) (string, error)
}

func (s *stubIssuer) Issue(ctx context.Context, claims domain.TokenClaims) (string, error) {
	if s.issue == nil {
		notConfigured("stubIssuer.Issue")
	}
	return s.issue(ctx, claims)
}

type stubVerifier struct {
	ports.TokenVerifier
	verify func(context.Context, string) (domain.TokenClaims, error)
}

func (s *stubVerifier) Verify(ctx context.Context, token string) (domain.TokenClaims, error) {
	if s.verify == nil {
		notConfigured("stubVerifier.Verify")
	}
	return s.verify(ctx, token)
}

type stubHasher struct {
	ports.TokenHasher
	hash func(context.Context, string) (string, error)
}

func (s *stubHasher) Hash(ctx context.Context, secret string) (string, error) {
	if s.hash == nil {
		notConfigured("stubHasher.Hash")
	}
	return s.hash(ctx, secret)
}

// Compile-time proof that the stubs satisfy their ports.
var (
	_ repositories.OrganizationRepository   = (*stubOrgs)(nil)
	_ repositories.UserRepository           = (*stubUsers)(nil)
	_ repositories.OrgMemberRepository      = (*stubMembers)(nil)
	_ repositories.InvitationRepository     = (*stubInvitations)(nil)
	_ repositories.ProjectRepository        = (*stubProjects)(nil)
	_ repositories.TeamRepository           = (*stubTeams)(nil)
	_ repositories.TeamMemberRepository     = (*stubTeamMembers)(nil)
	_ repositories.ProjectMemberRepository  = (*stubProjMembers)(nil)
	_ repositories.ServiceAccountRepository = (*stubAccounts)(nil)
	_ repositories.APICredentialRepository  = (*stubCredentials)(nil)
	_ repositories.AuditLogRepository       = (*stubAudits)(nil)
	_ ports.TokenIssuer                     = (*stubIssuer)(nil)
	_ ports.TokenVerifier                   = (*stubVerifier)(nil)
	_ ports.TokenHasher                     = (*stubHasher)(nil)
)
