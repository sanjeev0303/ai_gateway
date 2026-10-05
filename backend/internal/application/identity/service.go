// Package identity implements the application-layer service for the identity
// milestone: organizations, members, invitations, teams, projects, service
// accounts, API credentials, gateway credentials, and the token flows that
// sit in front of them.
//
// Every user-facing method takes a domain.Actor (a verified principal) and
// authorizes it via RequirePermission before touching a repository. Input
// validation, repository, and authorization failures surface the port-level
// sentinels (repositories.ErrNotFound, ErrConflict, ErrInvalid, ErrForbidden)
// unchanged; only credential failures are wrapped into identity sentinels.
package identity

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/credentialcache"
	"ai-governance-saas/backend/internal/ports"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// defaultInviteTTL is how long an outstanding invitation stays acceptable
// unless NewService is configured with a different duration.
const defaultInviteTTL = 7 * 24 * time.Hour

// Session is the outcome of a successful login: the signed JWT plus the
// claims it carries, so callers can derive the actor without re-parsing.
// RefreshToken is the raw rotating credential and is non-empty only on the
// paths that mint a session; it is returned exactly once, for the caller to
// place in an HttpOnly cookie.
type Session struct {
	Token        string
	RefreshToken string
	Claims       domain.TokenClaims
}

// ServiceDeps carries the dependencies of the identity service. Log, Now, and
// InviteTTL are optional (sane defaults apply); the rest are required.
type ServiceDeps struct {
	Orgs               repositories.OrganizationRepository
	Users              repositories.UserRepository
	Members            repositories.OrgMemberRepository
	Invitations        repositories.InvitationRepository
	Projects           repositories.ProjectRepository
	Teams              repositories.TeamRepository
	TeamMembers        repositories.TeamMemberRepository
	ProjMembers        repositories.ProjectMemberRepository
	Accounts           repositories.ServiceAccountRepository
	Credentials        repositories.APICredentialRepository
	GatewayCredentials repositories.GatewayCredentialRepository
	Audits             repositories.AuditLogRepository
	Auth               repositories.AuthRepository
	RefreshTokens      repositories.RefreshTokenRepository
	Subs               repositories.SubscriptionRepository

	Issuer                 ports.TokenIssuer
	Verifier               ports.TokenVerifier
	Hasher                 ports.TokenHasher
	Passwords              ports.PasswordHasher
	GatewayCredentialCache *credentialcache.CredentialCache
	Log                    *slog.Logger
	Now                    func() time.Time
	InviteTTL              time.Duration
	RefreshTokenTTL        time.Duration
}

// Service is the identity application service.
type Service struct {
	orgs                   repositories.OrganizationRepository
	users                  repositories.UserRepository
	members                repositories.OrgMemberRepository
	invitations            repositories.InvitationRepository
	projects               repositories.ProjectRepository
	teams                  repositories.TeamRepository
	teamMembers            repositories.TeamMemberRepository
	projMembers            repositories.ProjectMemberRepository
	accounts               repositories.ServiceAccountRepository
	credentials            repositories.APICredentialRepository
	gatewayCredentials     repositories.GatewayCredentialRepository
	audits                 repositories.AuditLogRepository
	auth                   repositories.AuthRepository
	refreshTokens          repositories.RefreshTokenRepository
	subs                   repositories.SubscriptionRepository
	gatewayCredentialCache *credentialcache.CredentialCache

	issuer     ports.TokenIssuer
	verifier   ports.TokenVerifier
	hasher     ports.TokenHasher
	passwords  ports.PasswordHasher
	log        *slog.Logger
	now        func() time.Time
	inviteTTL  time.Duration
	refreshTTL time.Duration
}

// NewService wires the identity service from its dependencies. Log, Now, and
// InviteTTL default to slog.Default(), time.Now, and one week respectively.
func NewService(deps ServiceDeps) *Service {
	log := deps.Log
	if log == nil {
		log = slog.Default()
	}
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	ttl := deps.InviteTTL
	if ttl <= 0 {
		ttl = defaultInviteTTL
	}
	refreshTTL := deps.RefreshTokenTTL
	if refreshTTL <= 0 {
		refreshTTL = defaultRefreshTokenTTL
	}
	return &Service{
		orgs:                   deps.Orgs,
		users:                  deps.Users,
		members:                deps.Members,
		invitations:            deps.Invitations,
		projects:               deps.Projects,
		teams:                  deps.Teams,
		teamMembers:            deps.TeamMembers,
		projMembers:            deps.ProjMembers,
		accounts:               deps.Accounts,
		credentials:            deps.Credentials,
		gatewayCredentials:     deps.GatewayCredentials,
		audits:                 deps.Audits,
		auth:                   deps.Auth,
		refreshTokens:          deps.RefreshTokens,
		subs:                   deps.Subs,
		gatewayCredentialCache: deps.GatewayCredentialCache,
		issuer:                 deps.Issuer,
		verifier:               deps.Verifier,
		hasher:                 deps.Hasher,
		passwords:              deps.Passwords,
		log:                    log,
		now:                    now,
		inviteTTL:              ttl,
		refreshTTL:             refreshTTL,
	}
}

// RequirePermission authorizes actor for perm, returning
// repositories.ErrForbidden when the actor's role does not grant it. Every
// service method starts with a RequirePermission gate.
func (s *Service) RequirePermission(actor domain.Actor, perm string) error {
	if !actor.Can(perm) {
		return repositories.ErrForbidden
	}
	return nil
}

// DevLogin signs in a user by email without a password. It provisions the
// user and their first organization on demand, so a fresh install can boot an
// owner immediately. The control API only exposes this when dev login is
// enabled.
func (s *Service) DevLogin(ctx context.Context, email, displayName string) (Session, error) {
	email = normalizeEmail(email)
	if email == "" {
		return Session{}, repositories.ErrInvalid
	}
	user, err := s.getOrCreateUser(ctx, email, displayName)
	if err != nil {
		return Session{}, err
	}
	orgs, err := s.users.ListOrganizationsForUser(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	orgID := uuid.Nil
	role := ""
	if len(orgs) > 0 {
		orgID = orgs[0].Organization.ID
		role = orgs[0].Role
	} else {
		org, err := s.provisionOrganization(ctx, user, displayName)
		if err != nil {
			return Session{}, err
		}
		orgID = org.ID
		role = domain.RoleOwner
	}
	// newSession, not issueSession: a dev-login account has no password, so
	// giving it a month-long refresh cookie would make the throwaway dev
	// credential the longest-lived secret in the system.
	return s.newSession(ctx, user, orgID, role)
}

// Invite issues an organization invitation and returns it together with the
// raw token — the only time the token is ever visible. Granting owner or
// admin requires the inviting actor to be the owner themselves.
func (s *Service) Invite(ctx context.Context, actor domain.Actor, email, role string) (domain.Invitation, string, error) {
	if err := s.RequirePermission(actor, domain.PermissionMembersManage); err != nil {
		return domain.Invitation{}, "", err
	}
	email = normalizeEmail(email)
	if email == "" {
		return domain.Invitation{}, "", repositories.ErrInvalid
	}
	if !domain.IsOrgRole(role) {
		return domain.Invitation{}, "", repositories.ErrInvalid
	}
	if (role == domain.RoleOwner || role == domain.RoleAdmin) && actor.Role != domain.RoleOwner {
		return domain.Invitation{}, "", repositories.ErrForbidden
	}
	raw, err := randomHex(32)
	if err != nil {
		return domain.Invitation{}, "", err
	}
	hash, err := s.hasher.Hash(ctx, raw)
	if err != nil {
		return domain.Invitation{}, "", err
	}
	invitedBy := actor.UserID
	invitation, err := s.invitations.Create(ctx, actor.OrganizationID, domain.NewInvitation{
		Email:     email,
		Role:      role,
		TokenHash: hash,
		Expires:   s.now().Add(s.inviteTTL),
		InvitedBy: &invitedBy,
	})
	if err != nil {
		return domain.Invitation{}, "", err
	}
	s.audit(ctx, actor, "invitation.create", "invitation", &invitation.ID, nil, nil)
	return invitation, raw, nil
}

// AcceptInvitation joins the authenticated user to the organization the
// invitation targets. Accepting is idempotent: a member who presents a
// still-resolvable token for an org they already belong to is returned as-is.
func (s *Service) AcceptInvitation(ctx context.Context, actor domain.Actor, rawToken string) (domain.OrgMember, error) {
	if actor.Kind != domain.ActorKindUser {
		return domain.OrgMember{}, repositories.ErrForbidden
	}
	hash, err := s.hasher.Hash(ctx, rawToken)
	if err != nil {
		return domain.OrgMember{}, err
	}
	// ErrNotFound from the token lookup passes through: the token is unknown.
	invitation, err := s.invitations.ResolveByTokenHash(ctx, hash)
	if err != nil {
		return domain.OrgMember{}, err
	}
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		return domain.OrgMember{}, err
	}
	if !strings.EqualFold(normalizeEmail(user.Email), normalizeEmail(invitation.Email)) {
		return domain.OrgMember{}, ErrInvitationEmailMismatch
	}
	if member, err := s.members.Get(ctx, invitation.OrganizationID, actor.UserID); err == nil {
		return member, nil
	} else if !errors.Is(err, repositories.ErrNotFound) {
		return domain.OrgMember{}, err
	}
	if invitation.Status != domain.InvitationStatusPending {
		return domain.OrgMember{}, ErrInvalidInvitation
	}
	if invitation.IsExpired(s.now()) {
		return domain.OrgMember{}, ErrInvitationExpired
	}
	if _, err := s.members.Add(ctx, invitation.OrganizationID, actor.UserID, invitation.Role); err != nil {
		if errors.Is(err, repositories.ErrConflict) {
			// A concurrent accept won the race; the membership now exists.
			return s.members.Get(ctx, invitation.OrganizationID, actor.UserID)
		}
		return domain.OrgMember{}, err
	}
	if _, err := s.invitations.MarkAccepted(ctx, invitation.OrganizationID, invitation.ID, actor.UserID); err != nil {
		return domain.OrgMember{}, err
	}
	member, err := s.members.Get(ctx, invitation.OrganizationID, actor.UserID)
	if err != nil {
		return domain.OrgMember{}, err
	}
	// The mutated tenant is the invited organization (a member joined), so the
	// audit record is attributed to that org, not the acceptor's own org.
	acceptActor := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         user.ID,
		OrganizationID: invitation.OrganizationID,
		Role:           invitation.Role,
		DisplayName:    user.DisplayName,
	}
	s.audit(ctx, acceptActor, "invitation.accept", "invitation", &invitation.ID, nil, nil)
	return member, nil
}

// RevokeInvitation rescinds a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, actor domain.Actor, invitationID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionMembersManage); err != nil {
		return err
	}
	if _, err := s.invitations.Revoke(ctx, actor.OrganizationID, invitationID); err != nil {
		return err
	}
	s.audit(ctx, actor, "invitation.revoke", "invitation", &invitationID, nil, nil)
	return nil
}

// ListInvitations returns the organization's outstanding invitations.
func (s *Service) ListInvitations(ctx context.Context, actor domain.Actor) ([]domain.Invitation, error) {
	if err := s.RequirePermission(actor, domain.PermissionMembersRead); err != nil {
		return nil, err
	}
	return s.invitations.List(ctx, actor.OrganizationID)
}

// ListMembers returns the organization's members with their profiles.
func (s *Service) ListMembers(ctx context.Context, actor domain.Actor) ([]domain.OrgMember, error) {
	if err := s.RequirePermission(actor, domain.PermissionMembersRead); err != nil {
		return nil, err
	}
	return s.members.List(ctx, actor.OrganizationID)
}

// GetOrganization returns the actor's own organization, plus its current plan
// when it has an active subscription.
func (s *Service) GetOrganization(ctx context.Context, actor domain.Actor) (domain.Organization, *domain.OrganizationPlan, error) {
	if err := s.RequirePermission(actor, domain.PermissionOrgRead); err != nil {
		return domain.Organization{}, nil, err
	}
	org, err := s.orgs.GetByID(ctx, actor.OrganizationID)
	if err != nil {
		return domain.Organization{}, nil, err
	}
	plan, err := s.subs.ActiveWithPlan(ctx, actor.OrganizationID)
	switch {
	case errors.Is(err, repositories.ErrNotFound):
		return org, nil, nil
	case err != nil:
		return domain.Organization{}, nil, err
	}
	return org, &plan, nil
}

// UpdateOrganization updates the actor's own organization.
func (s *Service) UpdateOrganization(ctx context.Context, actor domain.Actor, input domain.UpdateOrganization) (domain.Organization, error) {
	if err := s.RequirePermission(actor, domain.PermissionOrgUpdate); err != nil {
		return domain.Organization{}, err
	}
	if input.Name != nil && strings.TrimSpace(*input.Name) == "" {
		return domain.Organization{}, repositories.ErrInvalid
	}
	if input.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*input.Slug))
		if slug == "" {
			return domain.Organization{}, repositories.ErrInvalid
		}
		input.Slug = &slug
	}
	org, err := s.orgs.Update(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.Organization{}, err
	}
	s.audit(ctx, actor, "organization.update", "organization", &actor.OrganizationID, nil, nil)
	return org, nil
}

// SetMemberRole changes a member's role. Members cannot change their own
// role; granting owner or admin requires the acting owner; and the last
// owner can never be demoted.
func (s *Service) SetMemberRole(ctx context.Context, actor domain.Actor, userID uuid.UUID, role string) (domain.OrgMember, error) {
	if err := s.RequirePermission(actor, domain.PermissionMembersManage); err != nil {
		return domain.OrgMember{}, err
	}
	if !domain.IsOrgRole(role) {
		return domain.OrgMember{}, repositories.ErrInvalid
	}
	if userID == actor.UserID {
		return domain.OrgMember{}, repositories.ErrForbidden
	}
	if (role == domain.RoleOwner || role == domain.RoleAdmin) && actor.Role != domain.RoleOwner {
		return domain.OrgMember{}, repositories.ErrForbidden
	}
	current, err := s.members.Get(ctx, actor.OrganizationID, userID)
	if err != nil {
		return domain.OrgMember{}, err
	}
	if current.Role == domain.RoleOwner && role != domain.RoleOwner {
		count, err := s.members.CountByRole(ctx, actor.OrganizationID, domain.RoleOwner)
		if err != nil {
			return domain.OrgMember{}, err
		}
		if count <= 1 {
			return domain.OrgMember{}, repositories.ErrForbidden
		}
	}
	if _, err := s.members.SetRole(ctx, actor.OrganizationID, userID, role); err != nil {
		return domain.OrgMember{}, err
	}
	s.audit(ctx, actor, "member.role.update", "user", &userID, nil, nil)
	return s.members.Get(ctx, actor.OrganizationID, userID)
}

// RemoveMember removes a member from the organization. Members cannot remove
// themselves, and the last owner can never be removed.
func (s *Service) RemoveMember(ctx context.Context, actor domain.Actor, userID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionMembersManage); err != nil {
		return err
	}
	if userID == actor.UserID {
		return repositories.ErrForbidden
	}
	current, err := s.members.Get(ctx, actor.OrganizationID, userID)
	if err != nil {
		return err
	}
	if current.Role == domain.RoleOwner {
		count, err := s.members.CountByRole(ctx, actor.OrganizationID, domain.RoleOwner)
		if err != nil {
			return err
		}
		if count <= 1 {
			return repositories.ErrForbidden
		}
	}
	if err := s.members.Remove(ctx, actor.OrganizationID, userID); err != nil {
		return err
	}
	s.audit(ctx, actor, "member.remove", "user", &userID, nil, nil)
	return nil
}

// CreateProject creates a project inside the actor's organization.
func (s *Service) CreateProject(ctx context.Context, actor domain.Actor, input domain.NewProject) (domain.Project, error) {
	if err := s.RequirePermission(actor, domain.PermissionProjectsManage); err != nil {
		return domain.Project{}, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return domain.Project{}, repositories.ErrInvalid
	}
	project, err := s.projects.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.Project{}, err
	}
	s.audit(ctx, actor, "project.create", "project", &project.ID, nil, nil)
	return project, nil
}

// GetProject returns one project of the actor's organization.
func (s *Service) GetProject(ctx context.Context, actor domain.Actor, projectID uuid.UUID) (domain.Project, error) {
	if err := s.RequirePermission(actor, domain.PermissionProjectsRead); err != nil {
		return domain.Project{}, err
	}
	return s.projects.Get(ctx, actor.OrganizationID, projectID)
}

// ListProjects returns the actor's organization's projects.
func (s *Service) ListProjects(ctx context.Context, actor domain.Actor) ([]domain.Project, error) {
	if err := s.RequirePermission(actor, domain.PermissionProjectsRead); err != nil {
		return nil, err
	}
	return s.projects.List(ctx, actor.OrganizationID)
}

// CreateTeam creates a team inside the actor's organization.
func (s *Service) CreateTeam(ctx context.Context, actor domain.Actor, input domain.NewTeam) (domain.Team, error) {
	if err := s.RequirePermission(actor, domain.PermissionTeamsManage); err != nil {
		return domain.Team{}, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return domain.Team{}, repositories.ErrInvalid
	}
	team, err := s.teams.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.Team{}, err
	}
	s.audit(ctx, actor, "team.create", "team", &team.ID, nil, nil)
	return team, nil
}

// GetTeam returns one team of the actor's organization.
func (s *Service) GetTeam(ctx context.Context, actor domain.Actor, teamID uuid.UUID) (domain.Team, error) {
	if err := s.RequirePermission(actor, domain.PermissionTeamsRead); err != nil {
		return domain.Team{}, err
	}
	return s.teams.Get(ctx, actor.OrganizationID, teamID)
}

// ListTeams returns the actor's organization's teams.
func (s *Service) ListTeams(ctx context.Context, actor domain.Actor) ([]domain.Team, error) {
	if err := s.RequirePermission(actor, domain.PermissionTeamsRead); err != nil {
		return nil, err
	}
	return s.teams.List(ctx, actor.OrganizationID)
}

// AddTeamMember adds a user to a team with an admin or member role.
func (s *Service) AddTeamMember(ctx context.Context, actor domain.Actor, teamID, userID uuid.UUID, role string) (domain.TeamMember, error) {
	if err := s.RequirePermission(actor, domain.PermissionTeamsManage); err != nil {
		return domain.TeamMember{}, err
	}
	if !isJoinRole(role) {
		return domain.TeamMember{}, repositories.ErrInvalid
	}
	member, err := s.teamMembers.Add(ctx, actor.OrganizationID, teamID, userID, role)
	if err != nil {
		return domain.TeamMember{}, err
	}
	s.audit(ctx, actor, "team_member.add", "user", &userID, nil, nil)
	return member, nil
}

// RemoveTeamMember removes a user from a team.
func (s *Service) RemoveTeamMember(ctx context.Context, actor domain.Actor, teamID, userID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionTeamsManage); err != nil {
		return err
	}
	if err := s.teamMembers.Remove(ctx, actor.OrganizationID, teamID, userID); err != nil {
		return err
	}
	s.audit(ctx, actor, "team_member.remove", "user", &userID, nil, nil)
	return nil
}

// ListTeamMembers returns the members of one team.
func (s *Service) ListTeamMembers(ctx context.Context, actor domain.Actor, teamID uuid.UUID) ([]domain.TeamMemberView, error) {
	if err := s.RequirePermission(actor, domain.PermissionTeamsRead); err != nil {
		return nil, err
	}
	return s.teamMembers.List(ctx, actor.OrganizationID, teamID)
}

// AddProjectMember adds a user to a project with an admin or member role.
func (s *Service) AddProjectMember(ctx context.Context, actor domain.Actor, projectID, userID uuid.UUID, role string) (domain.ProjectMember, error) {
	if err := s.RequirePermission(actor, domain.PermissionProjectsManage); err != nil {
		return domain.ProjectMember{}, err
	}
	if !isJoinRole(role) {
		return domain.ProjectMember{}, repositories.ErrInvalid
	}
	member, err := s.projMembers.Add(ctx, actor.OrganizationID, projectID, userID, role)
	if err != nil {
		return domain.ProjectMember{}, err
	}
	s.audit(ctx, actor, "project_member.add", "user", &userID, nil, nil)
	return member, nil
}

// RemoveProjectMember removes a user from a project.
func (s *Service) RemoveProjectMember(ctx context.Context, actor domain.Actor, projectID, userID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionProjectsManage); err != nil {
		return err
	}
	if err := s.projMembers.Remove(ctx, actor.OrganizationID, projectID, userID); err != nil {
		return err
	}
	s.audit(ctx, actor, "project_member.remove", "user", &userID, nil, nil)
	return nil
}

// ListProjectMembers returns the members of one project.
func (s *Service) ListProjectMembers(ctx context.Context, actor domain.Actor, projectID uuid.UUID) ([]domain.ProjectMemberView, error) {
	if err := s.RequirePermission(actor, domain.PermissionProjectsRead); err != nil {
		return nil, err
	}
	return s.projMembers.List(ctx, actor.OrganizationID, projectID)
}

// CreateServiceAccount creates a machine identity for automation.
func (s *Service) CreateServiceAccount(ctx context.Context, actor domain.Actor, input domain.NewServiceAccount) (domain.ServiceAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.ServiceAccount{}, err
	}
	if strings.TrimSpace(input.Name) == "" {
		return domain.ServiceAccount{}, repositories.ErrInvalid
	}
	account, err := s.accounts.Create(ctx, actor.OrganizationID, input)
	if err != nil {
		return domain.ServiceAccount{}, err
	}
	s.audit(ctx, actor, "service_account.create", "service_account", &account.ID, nil, nil)
	return account, nil
}

// ListServiceAccounts returns the organization's machine identities.
func (s *Service) ListServiceAccounts(ctx context.Context, actor domain.Actor) ([]domain.ServiceAccount, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return nil, err
	}
	return s.accounts.List(ctx, actor.OrganizationID)
}

// RevokeServiceAccount disables a machine identity.
func (s *Service) RevokeServiceAccount(ctx context.Context, actor domain.Actor, serviceAccountID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	if _, err := s.accounts.Revoke(ctx, actor.OrganizationID, serviceAccountID); err != nil {
		return err
	}
	s.audit(ctx, actor, "service_account.revoke", "service_account", &serviceAccountID, nil, nil)
	return nil
}

// CreateCredential creates an API key. Gateway-scoped keys authenticate the
// organization's gateway traffic; control/all keys authenticate a machine
// identity and therefore require an active service account. The raw key is
// returned exactly once, alongside the stored credential.
func (s *Service) CreateCredential(ctx context.Context, actor domain.Actor, name, scope string, serviceAccountID *uuid.UUID) (domain.APICredential, string, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.APICredential{}, "", err
	}
	if strings.TrimSpace(name) == "" {
		return domain.APICredential{}, "", repositories.ErrInvalid
	}
	switch scope {
	case string(domain.APICredentialScopeGateway):
		serviceAccountID = nil
	case string(domain.APICredentialScopeControl), string(domain.APICredentialScopeAll):
		if serviceAccountID == nil {
			return domain.APICredential{}, "", repositories.ErrInvalid
		}
		account, err := s.accounts.Get(ctx, actor.OrganizationID, *serviceAccountID)
		if err != nil {
			return domain.APICredential{}, "", err
		}
		if account.Status != domain.ServiceAccountStatusActive {
			return domain.APICredential{}, "", repositories.ErrInvalid
		}
	default:
		return domain.APICredential{}, "", repositories.ErrInvalid
	}
	raw, err := randomHex(16)
	if err != nil {
		return domain.APICredential{}, "", err
	}
	rawKey := apiKeyPrefix + raw
	hash, err := s.hasher.Hash(ctx, rawKey)
	if err != nil {
		return domain.APICredential{}, "", err
	}
	credential, err := s.credentials.Create(ctx, actor.OrganizationID, domain.NewAPICredential{
		Name:             name,
		KeyPrefix:        rawKey[:len(apiKeyPrefix)+7],
		KeyHash:          hash,
		Scope:            scope,
		ServiceAccountID: serviceAccountID,
	})
	if err != nil {
		return domain.APICredential{}, "", err
	}
	s.audit(ctx, actor, "api_credential.create", "api_credential", &credential.ID, nil, nil)
	return credential, rawKey, nil
}

// ListCredentials returns the organization's API credentials.
func (s *Service) ListCredentials(ctx context.Context, actor domain.Actor) ([]domain.APICredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return nil, err
	}
	return s.credentials.List(ctx, actor.OrganizationID)
}

// RevokeCredential disables an API key.
func (s *Service) RevokeCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	if _, err := s.credentials.Revoke(ctx, actor.OrganizationID, credentialID); err != nil {
		return err
	}
	s.audit(ctx, actor, "api_credential.revoke", "api_credential", &credentialID, nil, nil)
	return nil
}

// AuthenticateToken verifies a JWT and materializes its actor. Every rejected
// token — malformed, expired, tampered, or from a foreign issuer — maps to
// ErrUnauthorized so callers cannot distinguish failure modes.
func (s *Service) AuthenticateToken(ctx context.Context, rawToken string) (domain.Actor, error) {
	claims, err := s.VerifyToken(ctx, rawToken)
	if err != nil {
		return domain.Actor{}, err
	}
	return claims.Actor(), nil
}

// VerifyToken verifies a JWT and returns the full claim set. Authentication
// itself only needs the actor projection, so AuthenticateToken is the narrower
// form; callers that must report session identity (email, expiry) without a
// second lookup use this. It performs exactly the same verification and maps
// every rejection to ErrUnauthorized.
func (s *Service) VerifyToken(ctx context.Context, rawToken string) (domain.TokenClaims, error) {
	claims, err := s.verifier.Verify(ctx, rawToken)
	if err != nil {
		return domain.TokenClaims{}, ErrUnauthorized
	}
	return claims, nil
}

// AuthenticateAPIKey resolves a raw API key to a service actor. Only active,
// non-gateway keys bound to an active service account authenticate.
func (s *Service) AuthenticateAPIKey(ctx context.Context, rawKey string) (domain.Actor, error) {
	hash, err := s.hasher.Hash(ctx, rawKey)
	if err != nil {
		return domain.Actor{}, err
	}
	credential, err := s.credentials.ResolveByKeyHash(ctx, hash)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Actor{}, ErrUnauthorized
		}
		return domain.Actor{}, err
	}
	if credential.Status != domain.APICredentialStatusActive ||
		credential.Scope == string(domain.APICredentialScopeGateway) ||
		credential.ServiceAccountID == nil {
		return domain.Actor{}, ErrUnauthorized
	}
	account, err := s.accounts.Get(ctx, credential.OrganizationID, *credential.ServiceAccountID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.Actor{}, ErrUnauthorized
		}
		return domain.Actor{}, err
	}
	if account.Status != domain.ServiceAccountStatusActive {
		return domain.Actor{}, ErrUnauthorized
	}
	return domain.ServiceActor(credential.OrganizationID, account.ID, account.Name), nil
}

// getOrCreateUser returns the user for email, creating it on first sight. A
// lost create race adopts the winner's row via the ErrConflict retry.
func (s *Service) getOrCreateUser(ctx context.Context, email, displayName string) (domain.User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, repositories.ErrNotFound) {
		return domain.User{}, err
	}
	user, err = s.users.Create(ctx, domain.NewUser{Email: email, DisplayName: displayName})
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, repositories.ErrConflict) {
		return domain.User{}, err
	}
	return s.users.GetByEmail(ctx, email)
}

// provisionOrganization creates a user-owned organization and installs the
// user as its owner. The slug derives from the display name (falling back to
// the email local part) with a random suffix on collision.
func (s *Service) provisionOrganization(ctx context.Context, user domain.User, displayName string) (domain.Organization, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = emailLocalPart(user.Email)
	}
	slug := slugify(name)
	if slug == "" {
		slug = "org"
	}
	var org domain.Organization
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		candidate := slug
		if attempt > 0 {
			suffix, err := randomHex(2)
			if err != nil {
				return domain.Organization{}, err
			}
			candidate = slug + "-" + suffix
		}
		org, err = s.orgs.Create(ctx, domain.NewOrganization{Name: name, Slug: candidate})
		if err == nil {
			break
		}
		if !errors.Is(err, repositories.ErrConflict) {
			return domain.Organization{}, err
		}
	}
	if err != nil {
		return domain.Organization{}, err
	}
	if _, err := s.members.Add(ctx, org.ID, user.ID, domain.RoleOwner); err != nil {
		return domain.Organization{}, err
	}
	owner := domain.Actor{
		Kind:           domain.ActorKindUser,
		UserID:         user.ID,
		OrganizationID: org.ID,
		Role:           domain.RoleOwner,
		DisplayName:    user.DisplayName,
	}
	s.audit(ctx, owner, "organization.create", "organization", &org.ID, nil, nil)
	return org, nil
}

// emailLocalPart returns the portion of an email before the @.
func emailLocalPart(email string) string {
	if i := strings.IndexByte(email, '@'); i > 0 {
		return email[:i]
	}
	return email
}

// isJoinRole reports whether role is valid for a team or project membership.
// Both join-table CHECKs allow exactly admin and member.
func isJoinRole(role string) bool {
	return role == domain.RoleAdmin || role == domain.RoleMember
}

// audit appends an audit record on a best-effort basis: a failed write is
// logged, never returned, so the governing mutation still succeeds.
func (s *Service) audit(ctx context.Context, actor domain.Actor, action, resourceType string, resourceID *uuid.UUID, before, after []byte) {
	entry := domain.NewAuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Before:       before,
		After:        after,
	}
	if actor.Kind == domain.ActorKindService {
		entry.ActorServiceAccountID = &actor.UserID
	} else {
		entry.ActorUserID = &actor.UserID
	}
	if _, err := s.audits.Append(ctx, actor.OrganizationID, entry); err != nil {
		s.log.WarnContext(ctx, "identity: audit append failed",
			"action", action, "resource_type", resourceType, "error", err)
	}
}

// CreateGatewayCredential creates a new gateway credential for a user or service account.
// The raw key is returned exactly once, alongside the stored credential.
func (s *Service) CreateGatewayCredential(ctx context.Context, actor domain.Actor, input domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.GatewayCredential{}, "", err
	}
	if strings.TrimSpace(input.Name) == "" {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}
	if !isValidSubjectType(input.SubjectType) {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}
	for _, scope := range input.Scopes {
		if !isValidGatewayCredentialScope(scope) {
			return domain.GatewayCredential{}, "", repositories.ErrInvalid
		}
	}
	if input.TeamID != nil {
		// Verify team belongs to organization
		team, err := s.teams.Get(ctx, actor.OrganizationID, *input.TeamID)
		if err != nil {
			return domain.GatewayCredential{}, "", err
		}
		if team.OrganizationID != actor.OrganizationID {
			return domain.GatewayCredential{}, "", repositories.ErrInvalid
		}
	}
	if input.ProjectID != nil {
		// Verify project belongs to organization
		project, err := s.projects.Get(ctx, actor.OrganizationID, *input.ProjectID)
		if err != nil {
			return domain.GatewayCredential{}, "", err
		}
		if project.OrganizationID != actor.OrganizationID {
			return domain.GatewayCredential{}, "", repositories.ErrInvalid
		}
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(s.now()) {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}

	// Generate raw key
	raw, err := randomHex(16)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}
	rawKey := apiKeyPrefix + raw
	hash, err := s.hasher.Hash(ctx, rawKey)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}

	credential, err := s.gatewayCredentials.Create(ctx, actor.OrganizationID, domain.NewGatewayCredential{
		Name:        input.Name,
		KeyPrefix:   rawKey[:len(apiKeyPrefix)+7],
		KeyHash:     hash,
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		TeamID:      input.TeamID,
		ProjectID:   input.ProjectID,
		Scopes:      input.Scopes,
		ExpiresAt:   input.ExpiresAt,
	})
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}
	s.audit(ctx, actor, "gateway_credential.create", "gateway_credential", &credential.ID, nil, nil)
	return credential, rawKey, nil
}

// ListGatewayCredentials returns the organization's gateway credentials.
func (s *Service) ListGatewayCredentials(ctx context.Context, actor domain.Actor) ([]domain.GatewayCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return nil, err
	}
	return s.gatewayCredentials.List(ctx, actor.OrganizationID)
}

// GetGatewayCredential returns one gateway credential of the actor's organization.
func (s *Service) GetGatewayCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) (domain.GatewayCredential, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsRead); err != nil {
		return domain.GatewayCredential{}, err
	}
	return s.gatewayCredentials.Get(ctx, actor.OrganizationID, credentialID)
}

// RevokeGatewayCredential disables a gateway credential.
func (s *Service) RevokeGatewayCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	credential, err := s.gatewayCredentials.Revoke(ctx, actor.OrganizationID, credentialID)
	if err != nil {
		return err
	}
	if s.gatewayCredentialCache != nil {
		s.gatewayCredentialCache.Invalidate(ctx, s.toCachedCredential(&credential))
	}
	s.audit(ctx, actor, "gateway_credential.revoke", "gateway_credential", &credentialID, nil, nil)
	return nil
}

// RotateGatewayCredential replaces an active credential with a new key.
// The old credential is revoked and its cache entry invalidated.
func (s *Service) RotateGatewayCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID, input domain.NewGatewayCredential) (domain.GatewayCredential, string, error) {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return domain.GatewayCredential{}, "", err
	}
	if strings.TrimSpace(input.Name) == "" {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}
	for _, scope := range input.Scopes {
		if !domain.IsValidGatewayCredentialScope(scope) {
			return domain.GatewayCredential{}, "", repositories.ErrInvalid
		}
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(s.now()) {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}

	// Get existing credential to verify it exists and is active
	existing, err := s.gatewayCredentials.Get(ctx, actor.OrganizationID, credentialID)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}
	if existing.Status != domain.GatewayCredentialStatusActive {
		return domain.GatewayCredential{}, "", repositories.ErrInvalid
	}

	// Generate new raw key
	raw, err := randomHex(16)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}
	rawKey := apiKeyPrefix + raw
	hash, err := s.hasher.Hash(ctx, rawKey)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}

	// Create new credential with rotation input (preserves subject/team/project if not overridden)
	newInput := domain.NewGatewayCredential{
		Name:        input.Name,
		KeyPrefix:   rawKey[:len(apiKeyPrefix)+7],
		KeyHash:     hash,
		SubjectType: existing.SubjectType,
		SubjectID:   existing.SubjectID,
		TeamID:      existing.TeamID,
		ProjectID:   existing.ProjectID,
		Scopes:      existing.Scopes,
		ExpiresAt:   existing.ExpiresAt,
	}
	if len(input.Scopes) > 0 {
		newInput.Scopes = input.Scopes
	}
	if input.ExpiresAt != nil {
		newInput.ExpiresAt = input.ExpiresAt
	}
	if input.TeamID != nil {
		newInput.TeamID = input.TeamID
	}
	if input.ProjectID != nil {
		newInput.ProjectID = input.ProjectID
	}

	// Use transaction to revoke old and create new
	newCred, _, err := s.gatewayCredentials.Rotate(ctx, actor.OrganizationID, credentialID, newInput)
	if err != nil {
		return domain.GatewayCredential{}, "", err
	}

	// Invalidate old credential from cache
	if s.gatewayCredentialCache != nil {
		s.gatewayCredentialCache.Invalidate(ctx, s.toCachedCredential(&existing))
		s.gatewayCredentialCache.Set(ctx, s.toCachedCredential(&newCred))
	}
	s.audit(ctx, actor, "gateway_credential.rotate", "gateway_credential", &newCred.ID, nil, nil)
	return newCred, rawKey, nil
}

// ExpireGatewayCredential marks a credential as expired.
func (s *Service) ExpireGatewayCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	credential, err := s.gatewayCredentials.UpdateStatus(ctx, actor.OrganizationID, credentialID, domain.GatewayCredentialStatusExpired)
	if err != nil {
		return err
	}
	if s.gatewayCredentialCache != nil {
		s.gatewayCredentialCache.Invalidate(ctx, s.toCachedCredential(&credential))
	}
	s.audit(ctx, actor, "gateway_credential.expire", "gateway_credential", &credentialID, nil, nil)
	return nil
}

// RestoreGatewayCredential restores a revoked credential to active status.
// Only revoked credentials can be restored (not expired).
func (s *Service) RestoreGatewayCredential(ctx context.Context, actor domain.Actor, credentialID uuid.UUID) error {
	if err := s.RequirePermission(actor, domain.PermissionCredentialsManage); err != nil {
		return err
	}
	credential, err := s.gatewayCredentials.Get(ctx, actor.OrganizationID, credentialID)
	if err != nil {
		return err
	}
	if credential.Status != domain.GatewayCredentialStatusRevoked {
		return repositories.ErrInvalid
	}
	if credential.ExpiresAt != nil && !credential.ExpiresAt.After(s.now()) {
		return repositories.ErrInvalid // expired credentials cannot be restored
	}
	restored, err := s.gatewayCredentials.UpdateStatus(ctx, actor.OrganizationID, credentialID, domain.GatewayCredentialStatusActive)
	if err != nil {
		return err
	}
	if s.gatewayCredentialCache != nil {
		s.gatewayCredentialCache.Set(ctx, s.toCachedCredential(&restored))
	}
	s.audit(ctx, actor, "gateway_credential.restore", "gateway_credential", &credentialID, nil, nil)
	return nil
}

// ResolveGatewayCredential is the hot-path resolver used by the gateway proxy.
// It checks cache first, then DB, then caches the result.
func (s *Service) ResolveGatewayCredential(ctx context.Context, rawKey string) (domain.GatewayCredential, error) {
	hash, err := s.hasher.Hash(ctx, rawKey)
	if err != nil {
		return domain.GatewayCredential{}, err
	}

	// Try cache first
	if s.gatewayCredentialCache != nil {
		if cred := s.gatewayCredentialCache.GetByHash(ctx, hash); cred != nil {
			return s.fromCachedCredential(cred), nil
		}
	}

	// Fallback to DB
	credential, err := s.gatewayCredentials.ResolveByKeyHash(ctx, hash)
	if err != nil {
		return domain.GatewayCredential{}, err
	}

	// Cache the result
	if s.gatewayCredentialCache != nil {
		s.gatewayCredentialCache.Set(ctx, s.toCachedCredential(&credential))
	}
	return credential, nil
}

// toCachedCredential converts a domain.GatewayCredential to a platform credentialcache.CachedCredential.
func (s *Service) toCachedCredential(c *domain.GatewayCredential) *credentialcache.CachedCredential {
	scopes := make([]string, len(c.Scopes))
	for i, s := range c.Scopes {
		scopes[i] = string(s)
	}
	return &credentialcache.CachedCredential{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		KeyPrefix:      c.KeyPrefix,
		KeyHash:        c.KeyHash,
		SubjectType:    string(c.SubjectType),
		SubjectID:      c.SubjectID,
		TeamID:         c.TeamID,
		ProjectID:      c.ProjectID,
		Scopes:         scopes,
		Status:         string(c.Status),
		ExpiresAt:      c.ExpiresAt,
		LastUsedAt:     c.LastUsedAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// fromCachedCredential converts a platform credentialcache.CachedCredential to a domain.GatewayCredential.
func (s *Service) fromCachedCredential(c *credentialcache.CachedCredential) domain.GatewayCredential {
	scopes := make([]domain.GatewayCredentialScope, len(c.Scopes))
	for i, s := range c.Scopes {
		scopes[i] = domain.GatewayCredentialScope(s)
	}
	return domain.GatewayCredential{
		ID:             c.ID,
		OrganizationID: c.OrganizationID,
		Name:           c.Name,
		KeyPrefix:      c.KeyPrefix,
		KeyHash:        c.KeyHash,
		SubjectType:    domain.SubjectType(c.SubjectType),
		SubjectID:      c.SubjectID,
		TeamID:         c.TeamID,
		ProjectID:      c.ProjectID,
		Scopes:         scopes,
		Status:         domain.GatewayCredentialStatus(c.Status),
		ExpiresAt:      c.ExpiresAt,
		LastUsedAt:     c.LastUsedAt,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

// isValidSubjectType reports whether the subject type is valid.
func isValidSubjectType(t domain.SubjectType) bool {
	return t == domain.SubjectTypeUser || t == domain.SubjectTypeServiceAccount
}

// isValidGatewayCredentialScope reports whether the scope is valid.
func isValidGatewayCredentialScope(s domain.GatewayCredentialScope) bool {
	return s == domain.GatewayCredentialScopeGateway || s == domain.GatewayCredentialScopeControl || s == domain.GatewayCredentialScopeAll
}
