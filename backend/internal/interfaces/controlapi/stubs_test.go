package controlapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/auth"
	"ai-governance-saas/backend/internal/ports"
	providerclient "ai-governance-saas/backend/internal/ports/provider"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"ai-governance-saas/backend/internal/ports/secrets"
)

// notConfigured aborts a test that reached a stub method no case installed.
// It mirrors the identity package's convention: unregistered calls are bugs in
// the test, not behaviours to paper over.
func notConfigured(method string) {
	panic("controlapi test: " + method + " called without a stub")
}

type stubOrgs struct {
	repositories.OrganizationRepository
	create  func(context.Context, domain.NewOrganization) (domain.Organization, error)
	getByID func(context.Context, uuid.UUID) (domain.Organization, error)
	update  func(context.Context, uuid.UUID, domain.UpdateOrganization) (domain.Organization, error)
}

func (s *stubOrgs) Create(ctx context.Context, in domain.NewOrganization) (domain.Organization, error) {
	if s.create == nil {
		notConfigured("stubOrgs.Create")
	}
	return s.create(ctx, in)
}

func (s *stubOrgs) GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	if s.getByID == nil {
		notConfigured("stubOrgs.GetByID")
	}
	return s.getByID(ctx, id)
}

func (s *stubOrgs) Update(ctx context.Context, id uuid.UUID, in domain.UpdateOrganization) (domain.Organization, error) {
	if s.update == nil {
		notConfigured("stubOrgs.Update")
	}
	return s.update(ctx, id, in)
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
	list          func(context.Context, uuid.UUID) ([]domain.Invitation, error)
	markAccepted  func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.Invitation, error)
	revoke        func(context.Context, uuid.UUID, uuid.UUID) (domain.Invitation, error)
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

func (s *stubInvitations) List(ctx context.Context, orgID uuid.UUID) ([]domain.Invitation, error) {
	if s.list == nil {
		notConfigured("stubInvitations.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubInvitations) MarkAccepted(ctx context.Context, orgID, id, acceptedBy uuid.UUID) (domain.Invitation, error) {
	if s.markAccepted == nil {
		notConfigured("stubInvitations.MarkAccepted")
	}
	return s.markAccepted(ctx, orgID, id, acceptedBy)
}

func (s *stubInvitations) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.Invitation, error) {
	if s.revoke == nil {
		notConfigured("stubInvitations.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

type stubProjects struct {
	repositories.ProjectRepository
	create func(context.Context, uuid.UUID, domain.NewProject) (domain.Project, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.Project, error)
	list   func(context.Context, uuid.UUID) ([]domain.Project, error)
}

func (s *stubProjects) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProject) (domain.Project, error) {
	if s.create == nil {
		notConfigured("stubProjects.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubProjects) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Project, error) {
	if s.get == nil {
		notConfigured("stubProjects.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubProjects) List(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error) {
	if s.list == nil {
		notConfigured("stubProjects.List")
	}
	return s.list(ctx, orgID)
}

type stubTeams struct {
	repositories.TeamRepository
	create func(context.Context, uuid.UUID, domain.NewTeam) (domain.Team, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.Team, error)
	list   func(context.Context, uuid.UUID) ([]domain.Team, error)
}

func (s *stubTeams) Create(ctx context.Context, orgID uuid.UUID, in domain.NewTeam) (domain.Team, error) {
	if s.create == nil {
		notConfigured("stubTeams.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubTeams) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Team, error) {
	if s.get == nil {
		notConfigured("stubTeams.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubTeams) List(ctx context.Context, orgID uuid.UUID) ([]domain.Team, error) {
	if s.list == nil {
		notConfigured("stubTeams.List")
	}
	return s.list(ctx, orgID)
}

type stubTeamMembers struct {
	repositories.TeamMemberRepository
	add    func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (domain.TeamMember, error)
	remove func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	list   func(context.Context, uuid.UUID, uuid.UUID) ([]domain.TeamMemberView, error)
}

func (s *stubTeamMembers) Add(ctx context.Context, orgID, teamID, userID uuid.UUID, role string) (domain.TeamMember, error) {
	if s.add == nil {
		notConfigured("stubTeamMembers.Add")
	}
	return s.add(ctx, orgID, teamID, userID, role)
}

func (s *stubTeamMembers) Remove(ctx context.Context, orgID, teamID, userID uuid.UUID) error {
	if s.remove == nil {
		notConfigured("stubTeamMembers.Remove")
	}
	return s.remove(ctx, orgID, teamID, userID)
}

func (s *stubTeamMembers) List(ctx context.Context, orgID, teamID uuid.UUID) ([]domain.TeamMemberView, error) {
	if s.list == nil {
		notConfigured("stubTeamMembers.List")
	}
	return s.list(ctx, orgID, teamID)
}

type stubProjMembers struct {
	repositories.ProjectMemberRepository
	add    func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (domain.ProjectMember, error)
	remove func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	list   func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProjectMemberView, error)
}

func (s *stubProjMembers) Add(ctx context.Context, orgID, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error) {
	if s.add == nil {
		notConfigured("stubProjMembers.Add")
	}
	return s.add(ctx, orgID, projectID, userID, role)
}

func (s *stubProjMembers) Remove(ctx context.Context, orgID, projectID, userID uuid.UUID) error {
	if s.remove == nil {
		notConfigured("stubProjMembers.Remove")
	}
	return s.remove(ctx, orgID, projectID, userID)
}

func (s *stubProjMembers) List(ctx context.Context, orgID, projectID uuid.UUID) ([]domain.ProjectMemberView, error) {
	if s.list == nil {
		notConfigured("stubProjMembers.List")
	}
	return s.list(ctx, orgID, projectID)
}

type stubAccounts struct {
	repositories.ServiceAccountRepository
	create func(context.Context, uuid.UUID, domain.NewServiceAccount) (domain.ServiceAccount, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.ServiceAccount, error)
	list   func(context.Context, uuid.UUID) ([]domain.ServiceAccount, error)
	revoke func(context.Context, uuid.UUID, uuid.UUID) (domain.ServiceAccount, error)
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

func (s *stubAccounts) List(ctx context.Context, orgID uuid.UUID) ([]domain.ServiceAccount, error) {
	if s.list == nil {
		notConfigured("stubAccounts.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubAccounts) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
	if s.revoke == nil {
		notConfigured("stubAccounts.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

type stubCredentials struct {
	repositories.APICredentialRepository
	create        func(context.Context, uuid.UUID, domain.NewAPICredential) (domain.APICredential, error)
	list          func(context.Context, uuid.UUID) ([]domain.APICredential, error)
	resolveByHash func(context.Context, string) (domain.APICredential, error)
	revoke        func(context.Context, uuid.UUID, uuid.UUID) (domain.APICredential, error)
}

func (s *stubCredentials) Create(ctx context.Context, orgID uuid.UUID, in domain.NewAPICredential) (domain.APICredential, error) {
	if s.create == nil {
		notConfigured("stubCredentials.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubCredentials) List(ctx context.Context, orgID uuid.UUID) ([]domain.APICredential, error) {
	if s.list == nil {
		notConfigured("stubCredentials.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubCredentials) ResolveByKeyHash(ctx context.Context, hash string) (domain.APICredential, error) {
	if s.resolveByHash == nil {
		notConfigured("stubCredentials.ResolveByKeyHash")
	}
	return s.resolveByHash(ctx, hash)
}

func (s *stubCredentials) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.APICredential, error) {
	if s.revoke == nil {
		notConfigured("stubCredentials.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

type stubGatewayCredentials struct {
	repositories.GatewayCredentialRepository
	create       func(context.Context, uuid.UUID, domain.NewGatewayCredential) (domain.GatewayCredential, error)
	get          func(context.Context, uuid.UUID, uuid.UUID) (domain.GatewayCredential, error)
	list         func(context.Context, uuid.UUID) ([]domain.GatewayCredential, error)
	revoke       func(context.Context, uuid.UUID, uuid.UUID) (domain.GatewayCredential, error)
	updateStatus func(context.Context, uuid.UUID, uuid.UUID, domain.GatewayCredentialStatus) (domain.GatewayCredential, error)
	rotate       func(context.Context, uuid.UUID, uuid.UUID, domain.NewGatewayCredential) (domain.GatewayCredential, string, error)
}

func (s *stubGatewayCredentials) Create(ctx context.Context, orgID uuid.UUID, in domain.NewGatewayCredential) (domain.GatewayCredential, error) {
	if s.create == nil {
		notConfigured("stubGatewayCredentials.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubGatewayCredentials) Get(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	if s.get == nil {
		notConfigured("stubGatewayCredentials.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubGatewayCredentials) List(ctx context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
	if s.list == nil {
		notConfigured("stubGatewayCredentials.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubGatewayCredentials) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	if s.revoke == nil {
		notConfigured("stubGatewayCredentials.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

func (s *stubGatewayCredentials) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status domain.GatewayCredentialStatus) (domain.GatewayCredential, error) {
	if s.updateStatus == nil {
		notConfigured("stubGatewayCredentials.UpdateStatus")
	}
	return s.updateStatus(ctx, orgID, id, status)
}

func (s *stubGatewayCredentials) Rotate(ctx context.Context, orgID, id uuid.UUID, in domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	if s.rotate == nil {
		notConfigured("stubGatewayCredentials.Rotate")
	}
	return s.rotate(ctx, orgID, id, in)
}

// stubAudits records every append and defaults to success, mirroring the
// identity package's recording stub.
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

type stubProviderAccounts struct {
	repositories.ProviderAccountRepository
	create       func(context.Context, uuid.UUID, domain.NewProviderAccount) (domain.ProviderAccount, error)
	get          func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderAccount, error)
	list         func(context.Context, uuid.UUID) ([]domain.ProviderAccount, error)
	updateStatus func(context.Context, uuid.UUID, uuid.UUID, string) (domain.ProviderAccount, error)
	delete       func(context.Context, uuid.UUID, uuid.UUID) error
}

func (s *stubProviderAccounts) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProviderAccount) (domain.ProviderAccount, error) {
	if s.create == nil {
		notConfigured("stubProviderAccounts.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubProviderAccounts) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
	if s.get == nil {
		notConfigured("stubProviderAccounts.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubProviderAccounts) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
	if s.list == nil {
		notConfigured("stubProviderAccounts.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubProviderAccounts) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status string) (domain.ProviderAccount, error) {
	if s.updateStatus == nil {
		notConfigured("stubProviderAccounts.UpdateStatus")
	}
	return s.updateStatus(ctx, orgID, id, status)
}

func (s *stubProviderAccounts) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if s.delete == nil {
		notConfigured("stubProviderAccounts.Delete")
	}
	return s.delete(ctx, orgID, id)
}

type stubProviderCredentials struct {
	repositories.ProviderCredentialRepository
	create        func(context.Context, uuid.UUID, domain.NewProviderCredential) (domain.ProviderCredential, error)
	get           func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error)
	list          func(context.Context, uuid.UUID) ([]domain.ProviderCredential, error)
	listByAccount func(context.Context, uuid.UUID, uuid.UUID) ([]domain.ProviderCredential, error)
	revoke        func(context.Context, uuid.UUID, uuid.UUID) (domain.ProviderCredential, error)
}

func (s *stubProviderCredentials) Create(ctx context.Context, orgID uuid.UUID, in domain.NewProviderCredential) (domain.ProviderCredential, error) {
	if s.create == nil {
		notConfigured("stubProviderCredentials.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubProviderCredentials) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	if s.get == nil {
		notConfigured("stubProviderCredentials.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubProviderCredentials) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
	if s.list == nil {
		notConfigured("stubProviderCredentials.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubProviderCredentials) ListByAccount(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
	if s.listByAccount == nil {
		notConfigured("stubProviderCredentials.ListByAccount")
	}
	return s.listByAccount(ctx, orgID, accountID)
}

func (s *stubProviderCredentials) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	if s.revoke == nil {
		notConfigured("stubProviderCredentials.Revoke")
	}
	return s.revoke(ctx, orgID, id)
}

type stubModels struct {
	repositories.ModelRepository
	list    func(context.Context) ([]domain.Model, error)
	listAll func(context.Context) ([]domain.Model, error)
	upsert  func(context.Context, string, string, string) (domain.Model, error)
}

func (s *stubModels) List(ctx context.Context) ([]domain.Model, error) {
	if s.list == nil {
		notConfigured("stubModels.List")
	}
	return s.list(ctx)
}

func (s *stubModels) Upsert(ctx context.Context, providerKind, modelSlug, displayName string) (domain.Model, error) {
	if s.upsert == nil {
		notConfigured("stubModels.Upsert")
	}
	return s.upsert(ctx, providerKind, modelSlug, displayName)
}

func (s *stubModels) ListAll(ctx context.Context) ([]domain.Model, error) {
	if s.listAll != nil {
		return s.listAll(ctx)
	}
	if s.list != nil {
		return s.list(ctx)
	}
	return nil, nil
}

func (s *stubModels) ListByProvider(ctx context.Context, providerKind string) ([]domain.Model, error) {
	if s.list != nil {
		models, err := s.list(ctx)
		if err != nil {
			return nil, err
		}
		var filtered []domain.Model
		for _, m := range models {
			if m.ProviderKind == providerKind {
				filtered = append(filtered, m)
			}
		}
		return filtered, nil
	}
	return nil, nil
}

func (s *stubModels) GetBySlug(ctx context.Context, providerKind, modelSlug string) (domain.Model, error) {
	return domain.Model{}, nil
}

func (s *stubModels) UpsertWithMetadata(ctx context.Context, m domain.Model) (domain.Model, error) {
	if s.upsert != nil {
		return s.upsert(ctx, m.ProviderKind, m.ModelSlug, m.DisplayName)
	}
	return m, nil
}

func (s *stubModels) MarkInactiveExcept(ctx context.Context, providerKind string, activeSlugs []string) error {
	return nil
}

type stubProviderHealth struct {
	repositories.ProviderHealthRepository
	get    func(context.Context, string) (domain.ProviderHealthState, error)
	list   func(context.Context) ([]domain.ProviderHealthState, error)
	upsert func(context.Context, domain.ProviderHealthState) error
	states []domain.ProviderHealthState
}

// Get defaults to ErrNotFound ("never probed") rather than panicking, because
// CheckHealth reads the prior state on every call to detect a transition. Tests
// that exercise transitions set get explicitly.
func (s *stubProviderHealth) Get(ctx context.Context, providerKind string) (domain.ProviderHealthState, error) {
	if s.get == nil {
		return domain.ProviderHealthState{}, repositories.ErrNotFound
	}
	return s.get(ctx, providerKind)
}

func (s *stubProviderHealth) List(ctx context.Context) ([]domain.ProviderHealthState, error) {
	if s.list != nil {
		return s.list(ctx)
	}
	return append([]domain.ProviderHealthState(nil), s.states...), nil
}

func (s *stubProviderHealth) Upsert(ctx context.Context, state domain.ProviderHealthState) error {
	if s.upsert != nil {
		return s.upsert(ctx, state)
	}
	s.states = append(s.states, state)
	return nil
}

// stubSecrets is an in-memory secrets store. Deleting a missing ref is a
// no-op by contract; NotFound is returned only by Get on a missing ref.
type stubSecrets struct {
	secrets.Store
	values map[string]string
}

func (s *stubSecrets) Get(_ context.Context, ref string) (string, error) {
	v, ok := s.values[ref]
	if !ok {
		return "", secrets.ErrNotFound
	}
	return v, nil
}

func (s *stubSecrets) Put(_ context.Context, ref, value string) error {
	s.values[ref] = value
	return nil
}

func (s *stubSecrets) Delete(_ context.Context, ref string) error {
	delete(s.values, ref)
	return nil
}

// stubProvider is an in-memory provider client; ValidateCredentials and
// HealthCheck pass by default so tests opt into failures.
type stubProvider struct {
	providerclient.Provider
	validateErr error
	healthErr   error
	listModels  func(context.Context, string) ([]providerclient.Model, error)
}

func (s *stubProvider) ValidateCredentials(_ context.Context, _ string) error {
	return s.validateErr
}

func (s *stubProvider) HealthCheck(_ context.Context, _ string) error {
	return s.healthErr
}

func (s *stubProvider) ListModels(ctx context.Context, apiKey string) ([]providerclient.Model, error) {
	if s.listModels == nil {
		notConfigured("stubProvider.ListModels")
	}
	return s.listModels(ctx, apiKey)
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
	_ repositories.OrganizationRepository       = (*stubOrgs)(nil)
	_ repositories.UserRepository               = (*stubUsers)(nil)
	_ repositories.OrgMemberRepository          = (*stubMembers)(nil)
	_ repositories.InvitationRepository         = (*stubInvitations)(nil)
	_ repositories.ProjectRepository            = (*stubProjects)(nil)
	_ repositories.TeamRepository               = (*stubTeams)(nil)
	_ repositories.TeamMemberRepository         = (*stubTeamMembers)(nil)
	_ repositories.ProjectMemberRepository      = (*stubProjMembers)(nil)
	_ repositories.ServiceAccountRepository     = (*stubAccounts)(nil)
	_ repositories.APICredentialRepository      = (*stubCredentials)(nil)
	_ repositories.AuditLogRepository           = (*stubAudits)(nil)
	_ repositories.ProviderAccountRepository    = (*stubProviderAccounts)(nil)
	_ repositories.ProviderCredentialRepository = (*stubProviderCredentials)(nil)
	_ repositories.ModelRepository              = (*stubModels)(nil)
	_ repositories.ProviderHealthRepository     = (*stubProviderHealth)(nil)
	_ secrets.Store                             = (*stubSecrets)(nil)
	_ providerclient.Provider                   = (*stubProvider)(nil)
	_ ports.TokenIssuer                         = (*stubIssuer)(nil)
	_ ports.TokenVerifier                       = (*stubVerifier)(nil)
	_ ports.TokenHasher                         = (*stubHasher)(nil)
)

type stubPolicyRepo struct {
	repositories.PolicyRepository
	create func(context.Context, uuid.UUID, domain.NewPolicy) (domain.Policy, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.Policy, error)
	list   func(context.Context, uuid.UUID) ([]domain.Policy, error)
	update func(context.Context, uuid.UUID, uuid.UUID, domain.UpdatePolicy) (domain.Policy, error)
	delete func(context.Context, uuid.UUID, uuid.UUID) error
}

func (s *stubPolicyRepo) Create(ctx context.Context, orgID uuid.UUID, in domain.NewPolicy) (domain.Policy, error) {
	if s.create == nil {
		notConfigured("stubPolicyRepo.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubPolicyRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	if s.get == nil {
		notConfigured("stubPolicyRepo.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubPolicyRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	if s.list == nil {
		notConfigured("stubPolicyRepo.List")
	}
	return s.list(ctx, orgID)
}

func (s *stubPolicyRepo) Update(ctx context.Context, orgID, id uuid.UUID, in domain.UpdatePolicy) (domain.Policy, error) {
	if s.update == nil {
		notConfigured("stubPolicyRepo.Update")
	}
	return s.update(ctx, orgID, id, in)
}

func (s *stubPolicyRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if s.delete == nil {
		notConfigured("stubPolicyRepo.Delete")
	}
	return s.delete(ctx, orgID, id)
}

type stubPolicyRuleRepo struct {
	repositories.PolicyRuleRepository
	create func(context.Context, uuid.UUID, domain.NewPolicyRule) (domain.PolicyRule, error)
	get    func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.PolicyRule, error)
	list   func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error)
	update func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdatePolicyRule) (domain.PolicyRule, error)
	delete func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (s *stubPolicyRuleRepo) Create(ctx context.Context, orgID uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
	if s.create == nil {
		notConfigured("stubPolicyRuleRepo.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubPolicyRuleRepo) Get(ctx context.Context, orgID, policyID, id uuid.UUID) (domain.PolicyRule, error) {
	if s.get == nil {
		notConfigured("stubPolicyRuleRepo.Get")
	}
	return s.get(ctx, orgID, policyID, id)
}

func (s *stubPolicyRuleRepo) List(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	if s.list == nil {
		notConfigured("stubPolicyRuleRepo.List")
	}
	return s.list(ctx, orgID, policyID)
}

func (s *stubPolicyRuleRepo) Update(ctx context.Context, orgID, policyID, id uuid.UUID, in domain.UpdatePolicyRule) (domain.PolicyRule, error) {
	if s.update == nil {
		notConfigured("stubPolicyRuleRepo.Update")
	}
	return s.update(ctx, orgID, policyID, id, in)
}

func (s *stubPolicyRuleRepo) Delete(ctx context.Context, orgID, policyID, id uuid.UUID) error {
	if s.delete == nil {
		notConfigured("stubPolicyRuleRepo.Delete")
	}
	return s.delete(ctx, orgID, policyID, id)
}

// requireNoErr and requireEqual are local copies of the identity package
// helpers; each test package owns its assertions.
func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func requireAudit(t *testing.T, st *controlStubs, action string) {
	t.Helper()
	for _, a := range st.audits.actions {
		if a == action {
			return
		}
	}
	t.Fatalf("audit actions = %v, missing %q", st.audits.actions, action)
}

// ---- auth, refresh token, and password stubs -----------------------------
//
// Signup, signin, and refresh need the credential repositories that the
// bearer/dev-login tests never touch, so they get their own stubs here. Each
// embeds its port so an unconfigured method fails loudly instead of silently
// returning a zero value.

type stubAuth struct {
	repositories.AuthRepository
	registerFn           func(context.Context, domain.Registration) (domain.User, domain.Organization, error)
	credentialByEmail    func(context.Context, string) (domain.UserCredential, error)
	setPasswordHashFn    func(context.Context, uuid.UUID, string) error
	touchLastLoginFn     func(context.Context, uuid.UUID) error
	capturedRegistration domain.Registration
}

func (s *stubAuth) Register(ctx context.Context, reg domain.Registration) (domain.User, domain.Organization, error) {
	s.capturedRegistration = reg
	if s.registerFn != nil {
		return s.registerFn(ctx, reg)
	}
	return testUser, testOrg, nil
}

func (s *stubAuth) GetCredentialByEmail(ctx context.Context, email string) (domain.UserCredential, error) {
	if s.credentialByEmail != nil {
		return s.credentialByEmail(ctx, email)
	}
	notConfigured("stubAuth.GetCredentialByEmail")
	return domain.UserCredential{}, nil
}

func (s *stubAuth) SetPasswordHash(ctx context.Context, userID uuid.UUID, encoded string) error {
	if s.setPasswordHashFn != nil {
		return s.setPasswordHashFn(ctx, userID, encoded)
	}
	return nil
}

func (s *stubAuth) TouchLastLogin(ctx context.Context, userID uuid.UUID) error {
	if s.touchLastLoginFn != nil {
		return s.touchLastLoginFn(ctx, userID)
	}
	return nil
}

type stubRefreshTokens struct {
	repositories.RefreshTokenRepository
	createFn     func(context.Context, uuid.UUID, string, time.Time) (domain.RefreshToken, error)
	getByHashFn  func(context.Context, string) (domain.RefreshToken, error)
	revokeFn     func(context.Context, string) error
	revokeAllFn  func(context.Context, uuid.UUID) error
	createdHash  string
	revokedHash  string
	revokeCalls  int
	revokeAllAny int
}

func (s *stubRefreshTokens) Create(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (domain.RefreshToken, error) {
	s.createdHash = tokenHash
	if s.createFn != nil {
		return s.createFn(ctx, userID, tokenHash, expiresAt)
	}
	return domain.RefreshToken{UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}, nil
}

func (s *stubRefreshTokens) GetByHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	if s.getByHashFn != nil {
		return s.getByHashFn(ctx, tokenHash)
	}
	notConfigured("stubRefreshTokens.GetByHash")
	return domain.RefreshToken{}, nil
}

func (s *stubRefreshTokens) Revoke(ctx context.Context, tokenHash string) error {
	s.revokeCalls++
	s.revokedHash = tokenHash
	if s.revokeFn != nil {
		return s.revokeFn(ctx, tokenHash)
	}
	return nil
}

func (s *stubRefreshTokens) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	s.revokeAllAny++
	if s.revokeAllFn != nil {
		return s.revokeAllFn(ctx, userID)
	}
	return nil
}

// stubPasswords stands in for the PBKDF2 hasher: the control API tests assert
// HTTP behaviour, and the real KDF is already covered by the identity package's
// own tests. Verify accepts only testPassword so both the success and the
// uniform-failure paths stay reachable.
type stubPasswords struct {
	ports.PasswordHasher
	hashFn    func(context.Context, string) (string, error)
	verifyFn  func(context.Context, string, string) error
	verified  string
	decoyHits int
}

func (s *stubPasswords) Hash(ctx context.Context, password string) (string, error) {
	if s.hashFn != nil {
		return s.hashFn(ctx, password)
	}
	return testPasswordDigest, nil
}

func (s *stubPasswords) Verify(ctx context.Context, encoded, password string) error {
	s.verified = password
	if s.verifyFn != nil {
		return s.verifyFn(ctx, encoded, password)
	}
	if password == testPassword {
		return nil
	}
	return auth.ErrPasswordMismatch
}

func (s *stubPasswords) DecoyDigest() string {
	s.decoyHits++
	return testPasswordDigest
}

type stubSubscriptions struct {
	repositories.SubscriptionRepository
	activeWithPlanFn func(context.Context, uuid.UUID) (domain.OrganizationPlan, error)
}

// ActiveWithPlan defaults to ErrNotFound — the state of every organization that
// has never subscribed — so plan-less responses need no per-test setup.
func (s *stubSubscriptions) ActiveWithPlan(ctx context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	if s.activeWithPlanFn != nil {
		return s.activeWithPlanFn(ctx, orgID)
	}
	return domain.OrganizationPlan{}, repositories.ErrNotFound
}

type stubBudgets struct {
	repositories.BudgetRepository
	create func(context.Context, uuid.UUID, domain.NewBudget) (domain.Budget, error)
	get    func(context.Context, uuid.UUID, uuid.UUID) (domain.Budget, error)
	list   func(context.Context, uuid.UUID, repositories.BudgetFilter) ([]domain.Budget, error)
	update func(context.Context, uuid.UUID, uuid.UUID, domain.UpdateBudget) (domain.Budget, error)
	delete func(context.Context, uuid.UUID, uuid.UUID) error
}

func (s *stubBudgets) Create(ctx context.Context, orgID uuid.UUID, in domain.NewBudget) (domain.Budget, error) {
	if s.create == nil {
		notConfigured("stubBudgets.Create")
	}
	return s.create(ctx, orgID, in)
}

func (s *stubBudgets) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Budget, error) {
	if s.get == nil {
		notConfigured("stubBudgets.Get")
	}
	return s.get(ctx, orgID, id)
}

func (s *stubBudgets) List(ctx context.Context, orgID uuid.UUID, filter repositories.BudgetFilter) ([]domain.Budget, error) {
	if s.list == nil {
		notConfigured("stubBudgets.List")
	}
	return s.list(ctx, orgID, filter)
}

func (s *stubBudgets) Update(ctx context.Context, orgID, id uuid.UUID, in domain.UpdateBudget) (domain.Budget, error) {
	if s.update == nil {
		notConfigured("stubBudgets.Update")
	}
	return s.update(ctx, orgID, id, in)
}

func (s *stubBudgets) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if s.delete == nil {
		notConfigured("stubBudgets.Delete")
	}
	return s.delete(ctx, orgID, id)
}

type stubPolicy struct {
	createPolicy func(context.Context, uuid.UUID, domain.NewPolicy) (domain.Policy, error)
	getPolicy    func(context.Context, uuid.UUID, uuid.UUID) (domain.Policy, error)
	listPolicies func(context.Context, uuid.UUID) ([]domain.Policy, error)
	updatePolicy func(context.Context, uuid.UUID, uuid.UUID, domain.UpdatePolicy) (domain.Policy, error)
	deletePolicy func(context.Context, uuid.UUID, uuid.UUID) error
	CreateStub   func(context.Context, uuid.UUID, domain.NewPolicy) (domain.Policy, error)
	createRule   func(context.Context, uuid.UUID, domain.NewPolicyRule) (domain.PolicyRule, error)
	getRule      func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (domain.PolicyRule, error)
	listRules    func(context.Context, uuid.UUID, uuid.UUID) ([]domain.PolicyRule, error)
	updateRule   func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, domain.UpdatePolicyRule) (domain.PolicyRule, error)
	deleteRule   func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	invalidator  func(uuid.UUID)
	auditFn      func(context.Context, uuid.UUID, domain.NewAuditLog) error
}

func (s *stubPolicy) Create(ctx context.Context, orgID uuid.UUID, in domain.NewPolicy) (domain.Policy, error) {
	if s.createPolicy == nil {
		notConfigured("stubPolicy.Create")
	}
	return s.createPolicy(ctx, orgID, in)
}

func (s *stubPolicy) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	if s.getPolicy == nil {
		notConfigured("stubPolicy.Get")
	}
	return s.getPolicy(ctx, orgID, id)
}

func (s *stubPolicy) List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	if s.listPolicies == nil {
		notConfigured("stubPolicy.List")
	}
	return s.listPolicies(ctx, orgID)
}

func (s *stubPolicy) Update(ctx context.Context, orgID, id uuid.UUID, in domain.UpdatePolicy) (domain.Policy, error) {
	if s.updatePolicy == nil {
		notConfigured("stubPolicy.Update")
	}
	return s.updatePolicy(ctx, orgID, id, in)
}

func (s *stubPolicy) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if s.deletePolicy == nil {
		notConfigured("stubPolicy.Delete")
	}
	return s.deletePolicy(ctx, orgID, id)
}

func (s *stubPolicy) CreateRule(ctx context.Context, orgID uuid.UUID, in domain.NewPolicyRule) (domain.PolicyRule, error) {
	if s.createRule == nil {
		notConfigured("stubPolicy.CreateRule")
	}
	return s.createRule(ctx, orgID, in)
}

func (s *stubPolicy) GetRule(ctx context.Context, orgID, policyID, id uuid.UUID) (domain.PolicyRule, error) {
	if s.getRule == nil {
		notConfigured("stubPolicy.GetRule")
	}
	return s.getRule(ctx, orgID, policyID, id)
}

func (s *stubPolicy) ListRules(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	if s.listRules == nil {
		notConfigured("stubPolicy.ListRules")
	}
	return s.listRules(ctx, orgID, policyID)
}

func (s *stubPolicy) UpdateRule(ctx context.Context, orgID, policyID, id uuid.UUID, in domain.UpdatePolicyRule) (domain.PolicyRule, error) {
	if s.updateRule == nil {
		notConfigured("stubPolicy.UpdateRule")
	}
	return s.updateRule(ctx, orgID, policyID, id, in)
}

func (s *stubPolicy) DeleteRule(ctx context.Context, orgID, policyID, id uuid.UUID) error {
	if s.deleteRule == nil {
		notConfigured("stubPolicy.DeleteRule")
	}
	return s.deleteRule(ctx, orgID, policyID, id)
}

// stubInvalidator is a test invalidator that does nothing
type stubInvalidator struct{}

func (stubInvalidator) Invalidate(uuid.UUID) {}
