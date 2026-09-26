package repositories

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type projectRepo struct {
	store *postgres.Store
}

// NewProjectRepository returns a ProjectRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewProjectRepository(store *postgres.Store) ports.ProjectRepository {
	return &projectRepo{store: store}
}

func (r *projectRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewProject) (domain.Project, error) {
	var project domain.Project
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateProject(ctx, db.CreateProjectParams{
			OrganizationID: orgID,
			Name:           input.Name,
			Slug:           input.Slug,
			Description:    input.Description,
		})
		if err != nil {
			return err
		}
		project = toProject(row)
		return nil
	})
	return project, mapUniqueViolation(err)
}

func (r *projectRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Project, error) {
	var project domain.Project
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetProject(ctx, db.GetProjectParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		project = toProject(row)
		return nil
	})
	return project, mapNotFound(err)
}

func (r *projectRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Project, error) {
	var projects []domain.Project
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListProjects(ctx, orgID)
		if err != nil {
			return err
		}
		projects = make([]domain.Project, 0, len(rows))
		for _, row := range rows {
			projects = append(projects, toProject(row))
		}
		return nil
	})
	return projects, err
}

func (r *projectRepo) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Project, error) {
	var project domain.Project
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.SetProjectStatus(ctx, db.SetProjectStatusParams{OrganizationID: orgID, ID: id, Status: status})
		if err != nil {
			return err
		}
		project = toProject(row)
		return nil
	})
	return project, mapNotFound(err)
}

type teamRepo struct {
	store *postgres.Store
}

// NewTeamRepository returns a TeamRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewTeamRepository(store *postgres.Store) ports.TeamRepository {
	return &teamRepo{store: store}
}

func (r *teamRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewTeam) (domain.Team, error) {
	var team domain.Team
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateTeam(ctx, db.CreateTeamParams{OrganizationID: orgID, Name: input.Name, Description: input.Description})
		if err != nil {
			return err
		}
		team = toTeam(row)
		return nil
	})
	return team, mapUniqueViolation(err)
}

func (r *teamRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Team, error) {
	var team domain.Team
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetTeam(ctx, db.GetTeamParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		team = toTeam(row)
		return nil
	})
	return team, mapNotFound(err)
}

func (r *teamRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Team, error) {
	var teams []domain.Team
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListTeams(ctx, orgID)
		if err != nil {
			return err
		}
		teams = make([]domain.Team, 0, len(rows))
		for _, row := range rows {
			teams = append(teams, toTeam(row))
		}
		return nil
	})
	return teams, err
}

type teamMemberRepo struct {
	store *postgres.Store
}

// NewTeamMemberRepository returns a TeamMemberRepository backed by the store.
// team_members is FORCE RLS, so every operation runs inside the tenant
// context scoped to orgID.
func NewTeamMemberRepository(store *postgres.Store) ports.TeamMemberRepository {
	return &teamMemberRepo{store: store}
}

func (r *teamMemberRepo) Add(ctx context.Context, orgID, teamID, userID uuid.UUID, role string) (domain.TeamMember, error) {
	var member domain.TeamMember
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.AddTeamMember(ctx, db.AddTeamMemberParams{TeamID: teamID, OrganizationID: orgID, UserID: userID, Role: role})
		if err != nil {
			return err
		}
		member = domain.TeamMember{
			TeamID:         row.TeamID,
			OrganizationID: row.OrganizationID,
			UserID:         row.UserID,
			Role:           row.Role,
			CreatedAt:      row.CreatedAt,
		}
		return nil
	})
	return member, mapUniqueViolation(err)
}

func (r *teamMemberRepo) Remove(ctx context.Context, orgID, teamID, userID uuid.UUID) error {
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.RemoveTeamMember(ctx, db.RemoveTeamMemberParams{TeamID: teamID, OrganizationID: orgID, UserID: userID})
		if err != nil {
			return err
		}
		if n == 0 {
			return ports.ErrNotFound
		}
		return nil
	})
	return err
}

func (r *teamMemberRepo) List(ctx context.Context, orgID, teamID uuid.UUID) ([]domain.TeamMemberView, error) {
	var members []domain.TeamMemberView
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListTeamMembers(ctx, db.ListTeamMembersParams{TeamID: teamID, OrganizationID: orgID})
		if err != nil {
			return err
		}
		members = make([]domain.TeamMemberView, 0, len(rows))
		for _, row := range rows {
			members = append(members, toTeamMemberView(row))
		}
		return nil
	})
	return members, err
}

type providerAccountRepo struct {
	store *postgres.Store
}

// NewProviderAccountRepository returns a ProviderAccountRepository backed by
// the store. All operations are tenant-scoped via the orgID context.
func NewProviderAccountRepository(store *postgres.Store) ports.ProviderAccountRepository {
	return &providerAccountRepo{store: store}
}

func (r *providerAccountRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewProviderAccount) (domain.ProviderAccount, error) {
	var account domain.ProviderAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateProviderAccount(ctx, db.CreateProviderAccountParams{
			OrganizationID: orgID,
			ProviderKind:   input.ProviderKind,
			Name:           input.Name,
			BaseUrl:        input.BaseURL,
			Status:         domain.ProviderStatusActive,
		})
		if err != nil {
			return err
		}
		account = toProviderAccount(row)
		return nil
	})
	return account, err
}

func (r *providerAccountRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderAccount, error) {
	var account domain.ProviderAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetProviderAccount(ctx, db.GetProviderAccountParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		account = toProviderAccount(row)
		return nil
	})
	return account, mapNotFound(err)
}

func (r *providerAccountRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderAccount, error) {
	var accounts []domain.ProviderAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListProviderAccounts(ctx, orgID)
		if err != nil {
			return err
		}
		accounts = make([]domain.ProviderAccount, 0, len(rows))
		for _, row := range rows {
			accounts = append(accounts, toProviderAccount(row))
		}
		return nil
	})
	return accounts, err
}

type providerCredentialRepo struct {
	store *postgres.Store
}

// NewProviderCredentialRepository returns a ProviderCredentialRepository
// backed by the store. All operations are tenant-scoped via the orgID context.
func NewProviderCredentialRepository(store *postgres.Store) ports.ProviderCredentialRepository {
	return &providerCredentialRepo{store: store}
}

func (r *providerCredentialRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewProviderCredential) (domain.ProviderCredential, error) {
	var credential domain.ProviderCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateProviderCredential(ctx, db.CreateProviderCredentialParams{
			ProviderAccountID: input.ProviderAccountID,
			OrganizationID:    orgID,
			Name:              input.Name,
			Kind:              input.Kind,
			SecretRef:         input.SecretRef,
			Fingerprint:       nonEmptyStringPtr(input.Fingerprint),
			Tail:              input.Tail,
			Status:            domain.ProviderCredentialStatusActive,
		})
		if err != nil {
			return err
		}
		credential = toProviderCredential(row)
		return nil
	})
	return credential, err
}

func (r *providerCredentialRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	var credential domain.ProviderCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetProviderCredential(ctx, db.GetProviderCredentialParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		credential = toProviderCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *providerCredentialRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.ProviderCredential, error) {
	var credentials []domain.ProviderCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListProviderCredentials(ctx, orgID)
		if err != nil {
			return err
		}
		credentials = make([]domain.ProviderCredential, 0, len(rows))
		for _, row := range rows {
			credentials = append(credentials, toProviderCredential(row))
		}
		return nil
	})
	return credentials, err
}

func (r *providerCredentialRepo) ListByAccount(ctx context.Context, orgID, accountID uuid.UUID) ([]domain.ProviderCredential, error) {
	var credentials []domain.ProviderCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListProviderCredentialsByAccount(ctx, db.ListProviderCredentialsByAccountParams{OrganizationID: orgID, ProviderAccountID: accountID})
		if err != nil {
			return err
		}
		credentials = make([]domain.ProviderCredential, 0, len(rows))
		for _, row := range rows {
			credentials = append(credentials, toProviderCredential(row))
		}
		return nil
	})
	return credentials, err
}

func (r *providerCredentialRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ProviderCredential, error) {
	var credential domain.ProviderCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RevokeProviderCredential(ctx, db.RevokeProviderCredentialParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		credential = toProviderCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

type apiCredentialRepo struct {
	store *postgres.Store
}

// NewAPICredentialRepository returns an APICredentialRepository backed by the
// store. Create/List/Revoke are tenant-scoped; ResolveByKeyHash reads across
// tenants via the SECURITY DEFINER app.resolve_api_credential.
func NewAPICredentialRepository(store *postgres.Store) ports.APICredentialRepository {
	return &apiCredentialRepo{store: store}
}

func (r *apiCredentialRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewAPICredential) (domain.APICredential, error) {
	if input.Scope != string(domain.APICredentialScopeGateway) && input.ServiceAccountID == nil {
		return domain.APICredential{}, fmt.Errorf("%w: non-gateway scoped credentials require a service account", ports.ErrInvalid)
	}
	var credential domain.APICredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateAPICredential(ctx, db.CreateAPICredentialParams{
			OrganizationID:   orgID,
			Name:             input.Name,
			KeyPrefix:        input.KeyPrefix,
			KeyHash:          input.KeyHash,
			Scope:            input.Scope,
			Status:           domain.APICredentialStatusActive,
			ServiceAccountID: input.ServiceAccountID,
		})
		if err != nil {
			return err
		}
		credential = toAPICredential(row)
		return nil
	})
	return credential, mapUniqueViolation(err)
}

func (r *apiCredentialRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.APICredential, error) {
	var credentials []domain.APICredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListAPICredentials(ctx, orgID)
		if err != nil {
			return err
		}
		credentials = make([]domain.APICredential, 0, len(rows))
		for _, row := range rows {
			credentials = append(credentials, toAPICredential(row))
		}
		return nil
	})
	return credentials, err
}

func (r *apiCredentialRepo) ResolveByKeyHash(ctx context.Context, keyHash string) (domain.APICredential, error) {
	var credential domain.APICredential
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.ResolveAPICredentialByKeyHash(ctx, keyHash)
		if err != nil {
			return err
		}
		credential = toAPICredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *apiCredentialRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.APICredential, error) {
	var credential domain.APICredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RevokeAPICredential(ctx, db.RevokeAPICredentialParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		credential = toAPICredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

type serviceAccountRepo struct {
	store *postgres.Store
}

// NewServiceAccountRepository returns a ServiceAccountRepository backed by
// the store. All operations are tenant-scoped via the orgID context.
func NewServiceAccountRepository(store *postgres.Store) ports.ServiceAccountRepository {
	return &serviceAccountRepo{store: store}
}

func (r *serviceAccountRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewServiceAccount) (domain.ServiceAccount, error) {
	var account domain.ServiceAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateServiceAccount(ctx, db.CreateServiceAccountParams{
			OrganizationID: orgID,
			Name:           input.Name,
			Description:    input.Description,
			Status:         domain.ServiceAccountStatusActive,
		})
		if err != nil {
			return err
		}
		account = toServiceAccount(row)
		return nil
	})
	return account, mapUniqueViolation(err)
}

func (r *serviceAccountRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.ServiceAccount, error) {
	var accounts []domain.ServiceAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListServiceAccounts(ctx, orgID)
		if err != nil {
			return err
		}
		accounts = make([]domain.ServiceAccount, 0, len(rows))
		for _, row := range rows {
			accounts = append(accounts, toServiceAccount(row))
		}
		return nil
	})
	return accounts, err
}

func (r *serviceAccountRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
	var account domain.ServiceAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetServiceAccount(ctx, db.GetServiceAccountParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		account = toServiceAccount(row)
		return nil
	})
	return account, mapNotFound(err)
}

func (r *serviceAccountRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.ServiceAccount, error) {
	var account domain.ServiceAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RevokeServiceAccount(ctx, db.RevokeServiceAccountParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		account = toServiceAccount(row)
		return nil
	})
	return account, mapNotFound(err)
}

type policyRepo struct {
	store *postgres.Store
}

// NewPolicyRepository returns a PolicyRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewPolicyRepository(store *postgres.Store) ports.PolicyRepository {
	return &policyRepo{store: store}
}

func (r *policyRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewPolicy) (domain.Policy, error) {
	var policy domain.Policy
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreatePolicy(ctx, db.CreatePolicyParams{
			OrganizationID: orgID,
			Name:           input.Name,
			Description:    input.Description,
			Enforcement:    input.Enforcement,
			Priority:       input.Priority,
		})
		if err != nil {
			return err
		}
		policy = toPolicy(row)
		return nil
	})
	return policy, err
}

func (r *policyRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Policy, error) {
	var policy domain.Policy
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetPolicy(ctx, db.GetPolicyParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		policy = toPolicy(row)
		return nil
	})
	return policy, mapNotFound(err)
}

func (r *policyRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Policy, error) {
	var policies []domain.Policy
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListPolicies(ctx, orgID)
		if err != nil {
			return err
		}
		policies = make([]domain.Policy, 0, len(rows))
		for _, row := range rows {
			policies = append(policies, toPolicy(row))
		}
		return nil
	})
	return policies, err
}

func (r *policyRepo) SetEnforcement(ctx context.Context, orgID, id uuid.UUID, enforcement string) (domain.Policy, error) {
	var policy domain.Policy
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.SetPolicyEnforcement(ctx, db.SetPolicyEnforcementParams{OrganizationID: orgID, ID: id, Enforcement: enforcement})
		if err != nil {
			return err
		}
		policy = toPolicy(row)
		return nil
	})
	return policy, mapNotFound(err)
}

type policyRuleRepo struct {
	store *postgres.Store
}

// NewPolicyRuleRepository returns a PolicyRuleRepository backed by the store.
// All operations are tenant-scoped via the orgID context.
func NewPolicyRuleRepository(store *postgres.Store) ports.PolicyRuleRepository {
	return &policyRuleRepo{store: store}
}

func (r *policyRuleRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewPolicyRule) (domain.PolicyRule, error) {
	var rule domain.PolicyRule
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreatePolicyRule(ctx, db.CreatePolicyRuleParams{
			PolicyID:       input.PolicyID,
			OrganizationID: orgID,
			RuleType:       input.RuleType,
			Description:    input.Description,
			Column5:        input.Conditions,
			Column6:        input.Parameters,
			Enabled:        input.Enabled,
		})
		if err != nil {
			return err
		}
		rule = toPolicyRule(row)
		return nil
	})
	return rule, err
}

func (r *policyRuleRepo) List(ctx context.Context, orgID, policyID uuid.UUID) ([]domain.PolicyRule, error) {
	var rules []domain.PolicyRule
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListPolicyRules(ctx, db.ListPolicyRulesParams{OrganizationID: orgID, PolicyID: policyID})
		if err != nil {
			return err
		}
		rules = make([]domain.PolicyRule, 0, len(rows))
		for _, row := range rows {
			rules = append(rules, toPolicyRule(row))
		}
		return nil
	})
	return rules, err
}

func (r *policyRuleRepo) SetEnabled(ctx context.Context, orgID, id uuid.UUID, enabled bool) (domain.PolicyRule, error) {
	var rule domain.PolicyRule
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.SetPolicyRuleEnabled(ctx, db.SetPolicyRuleEnabledParams{OrganizationID: orgID, ID: id, Enabled: enabled})
		if err != nil {
			return err
		}
		rule = toPolicyRule(row)
		return nil
	})
	return rule, mapNotFound(err)
}

func toProject(p db.Project) domain.Project {
	return domain.Project{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		Slug:           p.Slug,
		Description:    p.Description,
		Status:         p.Status,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toTeam(t db.Team) domain.Team {
	return domain.Team{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		Name:           t.Name,
		Description:    t.Description,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

func toTeamMemberView(r db.ListTeamMembersRow) domain.TeamMemberView {
	return domain.TeamMemberView{
		User:        toUserProfile(r.ID, r.Email, r.DisplayName, r.Status, r.CreatedAt, r.UpdatedAt, r.LastLoginAt),
		Role:        r.Role,
		MemberSince: r.MemberSince,
	}
}

func toProviderAccount(a db.ProviderAccount) domain.ProviderAccount {
	return domain.ProviderAccount{
		ID:             a.ID,
		OrganizationID: a.OrganizationID,
		ProviderKind:   a.ProviderKind,
		Name:           a.Name,
		BaseURL:        a.BaseUrl,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func toProviderCredential(c db.ProviderCredential) domain.ProviderCredential {
	fingerprint := ""
	if c.Fingerprint != nil {
		fingerprint = *c.Fingerprint
	}
	return domain.ProviderCredential{
		ID:                c.ID,
		OrganizationID:    c.OrganizationID,
		ProviderAccountID: c.ProviderAccountID,
		Name:              c.Name,
		Kind:              c.Kind,
		SecretRef:         c.SecretRef,
		Fingerprint:       fingerprint,
		Tail:              c.Tail,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func toAPICredential(c db.ApiCredential) domain.APICredential {
	return domain.APICredential{
		ID:               c.ID,
		OrganizationID:   c.OrganizationID,
		Name:             c.Name,
		KeyPrefix:        c.KeyPrefix,
		KeyHash:          c.KeyHash,
		Scope:            c.Scope,
		Status:           c.Status,
		ServiceAccountID: c.ServiceAccountID,
		LastUsedAt:       c.LastUsedAt,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func toServiceAccount(a db.ServiceAccount) domain.ServiceAccount {
	return domain.ServiceAccount{
		ID:             a.ID,
		OrganizationID: a.OrganizationID,
		Name:           a.Name,
		Description:    a.Description,
		Status:         a.Status,
		CreatedAt:      a.CreatedAt,
		UpdatedAt:      a.UpdatedAt,
	}
}

func toPolicy(p db.Policy) domain.Policy {
	return domain.Policy{
		ID:             p.ID,
		OrganizationID: p.OrganizationID,
		Name:           p.Name,
		Description:    p.Description,
		Enforcement:    p.Enforcement,
		Priority:       p.Priority,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toPolicyRule(r db.PolicyRule) domain.PolicyRule {
	return domain.PolicyRule{
		ID:             r.ID,
		PolicyID:       r.PolicyID,
		OrganizationID: r.OrganizationID,
		RuleType:       r.RuleType,
		Description:    r.Description,
		Conditions:     r.Conditions,
		Parameters:     r.Parameters,
		Enabled:        r.Enabled,
		CreatedAt:      r.CreatedAt,
	}
}

func toGatewayCredential(g db.GatewayCredential) domain.GatewayCredential {
	scopes := make([]domain.GatewayCredentialScope, len(g.Scopes))
	for i, s := range g.Scopes {
		scopes[i] = domain.GatewayCredentialScope(s)
	}
	return domain.GatewayCredential{
		ID:             g.ID,
		OrganizationID: g.OrganizationID,
		Name:           g.Name,
		KeyPrefix:      g.KeyPrefix,
		KeyHash:        g.KeyHash,
		SubjectType:    domain.SubjectType(g.SubjectType),
		SubjectID:      g.SubjectID,
		TeamID:         g.TeamID,
		ProjectID:      g.ProjectID,
		Scopes:         scopes,
		Status:         domain.GatewayCredentialStatus(g.Status),
		ExpiresAt:      g.ExpiresAt,
		LastUsedAt:     g.LastUsedAt,
		CreatedAt:      g.CreatedAt,
		UpdatedAt:      g.UpdatedAt,
	}
}

type gatewayCredentialRepo struct {
	store *postgres.Store
}

// NewGatewayCredentialRepository returns a GatewayCredentialRepository backed
// by the store. Create/Get/List/Revoke/UpdateStatus are tenant-scoped;
// ResolveByKeyHash/ResolveByKeyPrefix read across tenants via SECURITY DEFINER
// functions.
func NewGatewayCredentialRepository(store *postgres.Store) ports.GatewayCredentialRepository {
	return &gatewayCredentialRepo{store: store}
}

func (r *gatewayCredentialRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewGatewayCredential) (domain.GatewayCredential, error) {
	var credential domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateGatewayCredential(ctx, db.CreateGatewayCredentialParams{
			OrganizationID: orgID,
			Name:           input.Name,
			KeyPrefix:      input.KeyPrefix,
			KeyHash:        input.KeyHash,
			SubjectType:    db.GatewaySubjectType(input.SubjectType),
			SubjectID:      input.SubjectID,
			TeamID:         input.TeamID,
			ProjectID:      input.ProjectID,
			Scopes:         toDBScopeSlice(input.Scopes),
			ExpiresAt:      input.ExpiresAt,
		})
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, err
}

func toDBScopeSlice(scopes []domain.GatewayCredentialScope) []db.GatewayCredentialScope {
	out := make([]db.GatewayCredentialScope, len(scopes))
	for i, s := range scopes {
		out[i] = db.GatewayCredentialScope(s)
	}
	return out
}

func (r *gatewayCredentialRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	var credential domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetGatewayCredential(ctx, db.GetGatewayCredentialParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *gatewayCredentialRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.GatewayCredential, error) {
	var credentials []domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListGatewayCredentials(ctx, orgID)
		if err != nil {
			return err
		}
		credentials = make([]domain.GatewayCredential, 0, len(rows))
		for _, row := range rows {
			credentials = append(credentials, toGatewayCredential(row))
		}
		return nil
	})
	return credentials, err
}

func (r *gatewayCredentialRepo) ResolveByKeyHash(ctx context.Context, keyHash string) (domain.GatewayCredential, error) {
	var credential domain.GatewayCredential
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.ResolveGatewayCredentialByKeyHash(ctx, keyHash)
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *gatewayCredentialRepo) ResolveByKeyPrefix(ctx context.Context, keyPrefix string) ([]domain.GatewayCredential, error) {
	var credentials []domain.GatewayCredential
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		rows, err := q.ResolveGatewayCredentialByKeyPrefix(ctx, keyPrefix)
		if err != nil {
			return err
		}
		credentials = make([]domain.GatewayCredential, 0, len(rows))
		for _, row := range rows {
			credentials = append(credentials, toGatewayCredential(row))
		}
		return nil
	})
	return credentials, err
}

func (r *gatewayCredentialRepo) Revoke(ctx context.Context, orgID, id uuid.UUID) (domain.GatewayCredential, error) {
	var credential domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RevokeGatewayCredential(ctx, db.RevokeGatewayCredentialParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *gatewayCredentialRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status domain.GatewayCredentialStatus) (domain.GatewayCredential, error) {
	var credential domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.UpdateGatewayCredentialStatus(ctx, db.UpdateGatewayCredentialStatusParams{
			OrganizationID: orgID,
			ID:             id,
			Status:         db.GatewayCredentialStatus(status),
		})
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, mapNotFound(err)
}

func (r *gatewayCredentialRepo) Rotate(ctx context.Context, orgID, id uuid.UUID, input domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	// This is called from the service layer which generates the raw key and hash.
	// The service handles the rotation logic: revoke old, create new.
	// This method just creates the new credential after the old one is revoked.
	var credential domain.GatewayCredential
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateGatewayCredentialAfterRotate(ctx, db.CreateGatewayCredentialAfterRotateParams{
			OrganizationID: orgID,
			Name:           input.Name,
			KeyPrefix:      input.KeyPrefix,
			KeyHash:        input.KeyHash,
			SubjectType:    db.GatewaySubjectType(input.SubjectType),
			SubjectID:      input.SubjectID,
			TeamID:         input.TeamID,
			ProjectID:      input.ProjectID,
			Scopes:         toDBScopeSlice(input.Scopes),
			ExpiresAt:      input.ExpiresAt,
		})
		if err != nil {
			return err
		}
		credential = toGatewayCredential(row)
		return nil
	})
	return credential, "", err
}

type modelsRepo struct {
	store *postgres.Store
}

// NewModelRepository returns a ModelRepository backed by the store. The
// catalog is global, so reads run via WithTx (no tenant binding).
func NewModelRepository(store *postgres.Store) ports.ModelRepository {
	return &modelsRepo{store: store}
}

func (r *modelsRepo) List(ctx context.Context) ([]domain.Model, error) {
	var models []domain.Model
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		rows, err := q.ListModels(ctx)
		if err != nil {
			return err
		}
		models = make([]domain.Model, 0, len(rows))
		for _, row := range rows {
			models = append(models, toModel(row))
		}
		return nil
	})
	return models, err
}

func toModel(m db.Model) domain.Model {
	return domain.Model{
		ID:           m.ID,
		ProviderKind: m.ProviderKind,
		ModelSlug:    m.ModelSlug,
		DisplayName:  m.DisplayName,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type providerHealthRepo struct {
	store *postgres.Store
}

// NewProviderHealthRepository returns a ProviderHealthRepository backed by
// the store. Health is global, so reads/writes run via WithTx (no tenant
// binding).
func NewProviderHealthRepository(store *postgres.Store) ports.ProviderHealthRepository {
	return &providerHealthRepo{store: store}
}

func (r *providerHealthRepo) Get(ctx context.Context, providerKind string) (domain.ProviderHealthState, error) {
	var state domain.ProviderHealthState
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.GetProviderHealth(ctx, providerKind)
		if err != nil {
			return mapNotFound(err)
		}
		state = toProviderHealthState(row)
		return nil
	})
	return state, err
}

func (r *providerHealthRepo) List(ctx context.Context) ([]domain.ProviderHealthState, error) {
	var states []domain.ProviderHealthState
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		rows, err := q.ListProviderHealth(ctx)
		if err != nil {
			return err
		}
		states = make([]domain.ProviderHealthState, 0, len(rows))
		for _, row := range rows {
			states = append(states, toProviderHealthState(row))
		}
		return nil
	})
	return states, err
}

func (r *providerHealthRepo) Upsert(ctx context.Context, state domain.ProviderHealthState) error {
	return r.store.WithTx(ctx, func(q *db.Queries) error {
		_, err := q.UpsertProviderHealth(ctx, db.UpsertProviderHealthParams{
			ProviderKind: state.ProviderKind,
			Status:       state.Status,
			StatusReason: state.StatusReason,
			LastError:    state.LastError,
			CheckedAt:    state.CheckedAt,
		})
		return err
	})
}

func toProviderHealthState(h db.ProviderHealth) domain.ProviderHealthState {
	return domain.ProviderHealthState{
		ProviderKind: h.ProviderKind,
		Status:       h.Status,
		StatusReason: h.StatusReason,
		LastError:    h.LastError,
		CheckedAt:    h.CheckedAt,
	}
}
