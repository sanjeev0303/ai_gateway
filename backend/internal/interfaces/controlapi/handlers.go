package controlapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/activity"
	"ai-governance-saas/backend/internal/application/alert"
	"ai-governance-saas/backend/internal/application/auditlog"
	"ai-governance-saas/backend/internal/application/billing"
	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/application/provider"
	"ai-governance-saas/backend/internal/application/securityevent"
	"ai-governance-saas/backend/internal/application/usage"
	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/platform/httpx"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// maxBodyBytes caps request bodies at 1 MiB to reject oversized payloads early.
const maxBodyBytes = 1 << 20

// api carries the services every v1 handler needs plus the session/CSRF
// settings that drive the browser cookie flow.
type api struct {
	svc            *identity.Service
	providers      *provider.Service
	usage          *usage.Service
	activity       *activity.Service
	conversations  *conversation.Service
	auditLogs      *auditlog.Service
	alerts         *alert.Service
	securityEvents *securityevent.Service
	billing        *billing.Service

	sessionCookieName   string
	sessionCookieSecure bool
	csrfEnabled         bool

	signInLimiter *signInThrottle
}

func newAPI(deps Deps) *api {
	return &api{
		svc:                 deps.Identity,
		providers:           deps.Providers,
		usage:               deps.Usage,
		activity:            deps.Activity,
		conversations:       deps.Conversations,
		auditLogs:           deps.AuditLogs,
		alerts:              deps.Alerts,
		securityEvents:      deps.SecurityEvents,
		billing:             deps.Billing,
		sessionCookieName:   deps.SessionCookieName,
		sessionCookieSecure: deps.SessionCookieSecure,
		csrfEnabled:         deps.CSRFEnabled,
		signInLimiter:       newSignInThrottle(),
	}
}

// ---- wire DTOs -----------------------------------------------------------

type userDTO struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
}

type orgMemberDTO struct {
	User        userDTO `json:"user"`
	Role        string  `json:"role"`
	MemberSince string  `json:"member_since"`
}

type invitationDTO struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	ExpiresAt      string  `json:"expires_at"`
	InvitedBy      *string `json:"invited_by,omitempty"`
	AcceptedBy     *string `json:"accepted_by,omitempty"`
	AcceptedAt     *string `json:"accepted_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type organizationDTO struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Slug      string                  `json:"slug"`
	Status    string                  `json:"status"`
	CreatedAt string                  `json:"created_at"`
	UpdatedAt string                  `json:"updated_at"`
	Plan      *organizationPlanDTO    `json:"plan"`
	Billing   *organizationBillingDTO `json:"billing"`
}

type organizationPlanDTO struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type organizationBillingDTO struct {
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
}

type projectDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type teamDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type membershipDTO struct {
	User        userDTO `json:"user"`
	Role        string  `json:"role"`
	MemberSince string  `json:"member_since"`
}

type projectMemberRowDTO struct {
	ProjectID   string `json:"project_id"`
	UserID      string `json:"user_id"`
	Role        string `json:"role"`
	MemberSince string `json:"member_since"`
}

type teamMemberRowDTO struct {
	TeamID      string `json:"team_id"`
	UserID      string `json:"user_id"`
	Role        string `json:"role"`
	MemberSince string `json:"member_since"`
}

type serviceAccountDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// APICredentialDTO never exposes the key hash; the raw key is returned exactly
// once, in the create response.
type credentialDTO struct {
	ID               string  `json:"id"`
	OrganizationID   string  `json:"organization_id"`
	Name             string  `json:"name"`
	KeyPrefix        string  `json:"key_prefix"`
	Scope            string  `json:"scope"`
	Status           string  `json:"status"`
	ServiceAccountID *string `json:"service_account_id,omitempty"`
	LastUsedAt       *string `json:"last_used_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type claimsDTO struct {
	Subject        string `json:"subject"`
	Email          string `json:"email"`
	DisplayName    string `json:"display_name"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	Kind           string `json:"kind"`
	IssuedAt       string `json:"issued_at"`
	ExpiresAt      string `json:"expires_at"`
}

// sessionDTO reports who the caller is, never how to authenticate as them. The
// access and refresh tokens travel only in HttpOnly cookies; CSRFToken is
// returned because the browser must echo it in a header, which requires
// JavaScript to be able to read that one value.
type sessionDTO struct {
	Claims    claimsDTO `json:"claims"`
	CSRFToken string    `json:"csrf_token,omitempty"`
}

type inviteResponse struct {
	Invitation invitationDTO `json:"invitation"`
	Token      string        `json:"token"`
}

type createCredentialResponse struct {
	Credential credentialDTO `json:"credential"`
	APIKey     string        `json:"api_key"`
}

type listDTO struct {
	Items any `json:"items"`
}

// ---- request bodies ------------------------------------------------------

type devLoginRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

type setRoleRequest struct {
	Role string `json:"role"`
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type createTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type createServiceAccountRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createCredentialRequest struct {
	Name             string  `json:"name"`
	Scope            string  `json:"scope"`
	ServiceAccountID *string `json:"service_account_id,omitempty"`
}

// ---- DTO conversions -----------------------------------------------------

func toUserDTO(u domain.User) userDTO {
	dto := userDTO{
		ID:          u.ID.String(),
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Status:      u.Status,
		CreatedAt:   ts(u.CreatedAt),
		UpdatedAt:   ts(u.UpdatedAt),
	}
	if u.LastLoginAt != nil {
		s := ts(*u.LastLoginAt)
		dto.LastLoginAt = &s
	}
	return dto
}

func toOrgMemberDTO(m domain.OrgMember) orgMemberDTO {
	return orgMemberDTO{
		User:        toUserDTO(m.User),
		Role:        m.Role,
		MemberSince: ts(m.MemberSince),
	}
}

func toInvitationDTO(i domain.Invitation) invitationDTO {
	dto := invitationDTO{
		ID:             i.ID.String(),
		OrganizationID: i.OrganizationID.String(),
		Email:          i.Email,
		Role:           i.Role,
		Status:         i.Status,
		ExpiresAt:      ts(i.ExpiresAt),
		CreatedAt:      ts(i.CreatedAt),
		UpdatedAt:      ts(i.UpdatedAt),
	}
	if i.InvitedBy != nil {
		s := i.InvitedBy.String()
		dto.InvitedBy = &s
	}
	if i.AcceptedBy != nil {
		s := i.AcceptedBy.String()
		dto.AcceptedBy = &s
	}
	if i.AcceptedAt != nil {
		s := ts(*i.AcceptedAt)
		dto.AcceptedAt = &s
	}
	return dto
}

func toProjectDTO(p domain.Project) projectDTO {
	return projectDTO{
		ID:             p.ID.String(),
		OrganizationID: p.OrganizationID.String(),
		Name:           p.Name,
		Slug:           p.Slug,
		Description:    p.Description,
		Status:         p.Status,
		CreatedAt:      ts(p.CreatedAt),
		UpdatedAt:      ts(p.UpdatedAt),
	}
}

func toOrganizationDTO(o domain.Organization, plan *domain.OrganizationPlan) organizationDTO {
	dto := organizationDTO{
		ID:        o.ID.String(),
		Name:      o.Name,
		Slug:      o.Slug,
		Status:    o.Status,
		CreatedAt: ts(o.CreatedAt),
		UpdatedAt: ts(o.UpdatedAt),
	}
	if plan == nil {
		return dto
	}
	dto.Plan = &organizationPlanDTO{Code: plan.PlanCode, Name: plan.PlanName}
	dto.Billing = &organizationBillingDTO{
		Status:             plan.Status,
		CurrentPeriodStart: tsPtr(plan.CurrentPeriodStart),
		CurrentPeriodEnd:   tsPtr(plan.CurrentPeriodEnd),
		CancelAtPeriodEnd:  plan.CancelAtPeriodEnd,
	}
	return dto
}

func toTeamDTO(t domain.Team) teamDTO {
	return teamDTO{
		ID:             t.ID.String(),
		OrganizationID: t.OrganizationID.String(),
		Name:           t.Name,
		Description:    t.Description,
		CreatedAt:      ts(t.CreatedAt),
		UpdatedAt:      ts(t.UpdatedAt),
	}
}

func toProjectMemberRowDTO(m domain.ProjectMember) projectMemberRowDTO {
	return projectMemberRowDTO{
		ProjectID:   m.ProjectID.String(),
		UserID:      m.UserID.String(),
		Role:        m.Role,
		MemberSince: ts(m.CreatedAt),
	}
}

func toTeamMemberRowDTO(m domain.TeamMember) teamMemberRowDTO {
	return teamMemberRowDTO{
		TeamID:      m.TeamID.String(),
		UserID:      m.UserID.String(),
		Role:        m.Role,
		MemberSince: ts(m.CreatedAt),
	}
}

func toServiceAccountDTO(sa domain.ServiceAccount) serviceAccountDTO {
	return serviceAccountDTO{
		ID:             sa.ID.String(),
		OrganizationID: sa.OrganizationID.String(),
		Name:           sa.Name,
		Description:    sa.Description,
		Status:         sa.Status,
		CreatedAt:      ts(sa.CreatedAt),
		UpdatedAt:      ts(sa.UpdatedAt),
	}
}

func toCredentialDTO(c domain.APICredential) credentialDTO {
	dto := credentialDTO{
		ID:             c.ID.String(),
		OrganizationID: c.OrganizationID.String(),
		Name:           c.Name,
		KeyPrefix:      c.KeyPrefix,
		Scope:          c.Scope,
		Status:         c.Status,
		CreatedAt:      ts(c.CreatedAt),
		UpdatedAt:      ts(c.UpdatedAt),
	}
	if c.ServiceAccountID != nil {
		s := c.ServiceAccountID.String()
		dto.ServiceAccountID = &s
	}
	if c.LastUsedAt != nil {
		s := ts(*c.LastUsedAt)
		dto.LastUsedAt = &s
	}
	return dto
}

func toClaimsDTO(c domain.TokenClaims) claimsDTO {
	return claimsDTO{
		Subject:        c.Subject,
		Email:          c.Email,
		DisplayName:    c.DisplayName,
		OrganizationID: c.OrganizationID.String(),
		Role:           c.Role,
		Kind:           string(c.Kind),
		IssuedAt:       ts(c.IssuedAt),
		ExpiresAt:      ts(c.ExpiresAt),
	}
}

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func tsPtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return ts(*t)
}

// ---- helpers -------------------------------------------------------------

// writeServiceError maps service/repository errors to typed application errors
// and emits them with the correct status code. Unexpected errors are logged and
// degraded to a generic 500.
func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *perrors.Error
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		appErr = perrors.New(perrors.KindUnauthenticated, "unauthenticated", "missing or invalid credentials")
	case errors.Is(err, identity.ErrInvalidInvitation):
		appErr = perrors.New(perrors.KindInvalidArgument, "invalid_invitation", identity.ErrInvalidInvitation.Error())
	case errors.Is(err, identity.ErrInvitationExpired):
		appErr = perrors.New(perrors.KindInvalidArgument, "invitation_expired", identity.ErrInvitationExpired.Error())
	case errors.Is(err, identity.ErrInvitationEmailMismatch):
		appErr = perrors.New(perrors.KindInvalidArgument, "invitation_email_mismatch", identity.ErrInvitationEmailMismatch.Error())
	case errors.Is(err, provider.ErrInvalidProviderCredential):
		appErr = perrors.New(perrors.KindInvalidArgument, "invalid_provider_credential", provider.ErrInvalidProviderCredential.Error())
	case errors.Is(err, repositories.ErrInvalid):
		appErr = perrors.New(perrors.KindInvalidArgument, "invalid_argument", repositories.ErrInvalid.Error())
	case errors.Is(err, repositories.ErrNotFound):
		appErr = perrors.New(perrors.KindNotFound, "not_found", repositories.ErrNotFound.Error())
	case errors.Is(err, repositories.ErrEmailTaken):
		appErr = perrors.New(perrors.KindConflict, "email_taken", "an account with this email already exists")
	case errors.Is(err, repositories.ErrConflict):
		appErr = perrors.New(perrors.KindConflict, "conflict", repositories.ErrConflict.Error())
	case errors.Is(err, repositories.ErrForbidden):
		appErr = perrors.New(perrors.KindPermissionDenied, "permission_denied", repositories.ErrForbidden.Error())
	case errors.As(err, &appErr):
		// Contextual errors (decodeJSON, pathUUID, parseOptionalUUID) already
		// carry their own kind and wire code; pass them through unchanged.
	default:
		httpx.LoggerFrom(r.Context()).Error("controlapi: internal error", "error", err)
		appErr = perrors.New(perrors.KindInternal, "internal", "internal error")
	}
	httpx.WriteError(w, httpx.StatusFor(appErr.Kind), httpx.RequestIDFrom(r.Context()), appErr)
}

// decodeJSON parses a request body with a 1 MiB cap and rejects unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return perrors.New(perrors.KindInvalidArgument, "invalid_json", "malformed request body")
	}
	return nil
}

// pathUUID parses a named path segment as a UUID.
func pathUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, perrors.New(perrors.KindInvalidArgument, "invalid_id", "invalid "+name+" in path")
	}
	return id, nil
}

// invalidQuery marks a malformed query parameter with ErrInvalid so
// writeServiceError reports 400. A bare error falls through to 500, which
// tells the caller the gateway is broken when really they sent a bad value.
func invalidQuery(msg string) error {
	return fmt.Errorf("%w: %s", repositories.ErrInvalid, msg)
}

// parseOptionalUUID converts an optional JSON UUID string. An empty string is
// treated as absent.
func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, perrors.New(perrors.KindInvalidArgument, "invalid_id", "invalid id")
	}
	return &id, nil
}

// ---- auth handler --------------------------------------------------------

func (a *api) devLogin(w http.ResponseWriter, r *http.Request) {
	var req devLoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	session, err := a.svc.DevLogin(r.Context(), req.Email, req.DisplayName)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	dto := sessionDTO{Claims: toClaimsDTO(session.Claims)}
	if a.sessionCookieName != "" {
		csrf := newCSRFToken()
		a.setSessionCookies(w, session, csrf)
		dto.CSRFToken = csrf
	}
	httpx.WriteJSON(w, http.StatusOK, dto)
}

// logout retires the refresh token and clears every session cookie. Retiring
// the token is what makes logout server-side: clearing the cookies alone only
// removes the browser's copy, leaving a stolen refresh token usable until it
// expires.
func (a *api) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.refreshCookieName()); err == nil {
		if err := a.svc.RevokeRefresh(r.Context(), c.Value); err != nil {
			writeServiceError(w, r, err)
			return
		}
	}
	a.clearSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// ---- organization handlers ----------------------------------------------

func (a *api) getOrganization(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	org, plan, err := a.svc.GetOrganization(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrganizationDTO(org, plan))
}

type updateOrganizationRequest struct {
	Name *string `json:"name"`
	Slug *string `json:"slug"`
}

func (a *api) updateOrganization(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req updateOrganizationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	org, err := a.svc.UpdateOrganization(r.Context(), actor, domain.UpdateOrganization{
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrganizationDTO(org, nil))
}

// ---- member handlers -----------------------------------------------------

func (a *api) listMembers(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	members, err := a.svc.ListMembers(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]orgMemberDTO, 0, len(members))
	for _, m := range members {
		items = append(items, toOrgMemberDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) setMemberRole(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	userID, err := pathUUID(r, "userID")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req setRoleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	member, err := a.svc.SetMemberRole(r.Context(), actor, userID, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrgMemberDTO(member))
}

func (a *api) removeMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	userID, err := pathUUID(r, "userID")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RemoveMember(r.Context(), actor, userID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- invitation handlers -------------------------------------------------

func (a *api) invite(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req inviteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	inv, rawToken, err := a.svc.Invite(r.Context(), actor, req.Email, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, inviteResponse{
		Invitation: toInvitationDTO(inv),
		Token:      rawToken,
	})
}

func (a *api) acceptInvitation(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req acceptInvitationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	member, err := a.svc.AcceptInvitation(r.Context(), actor, req.Token)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toOrgMemberDTO(member))
}

func (a *api) listInvitations(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	invitations, err := a.svc.ListInvitations(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]invitationDTO, 0, len(invitations))
	for _, i := range invitations {
		items = append(items, toInvitationDTO(i))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) revokeInvitation(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RevokeInvitation(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- project handlers ----------------------------------------------------

func (a *api) createProject(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	project, err := a.svc.CreateProject(r.Context(), actor, domain.NewProject{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProjectDTO(project))
}

func (a *api) listProjects(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	projects, err := a.svc.ListProjects(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]projectDTO, 0, len(projects))
	for _, p := range projects {
		items = append(items, toProjectDTO(p))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) getProject(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	project, err := a.svc.GetProject(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProjectDTO(project))
}

func (a *api) addProjectMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	projectID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req addMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_id", "invalid user_id"))
		return
	}
	member, err := a.svc.AddProjectMember(r.Context(), actor, projectID, userID, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProjectMemberRowDTO(member))
}

func (a *api) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	projectID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	members, err := a.svc.ListProjectMembers(r.Context(), actor, projectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]membershipDTO, 0, len(members))
	for _, m := range members {
		items = append(items, membershipDTO{
			User:        toUserDTO(m.User),
			Role:        m.Role,
			MemberSince: ts(m.MemberSince),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	projectID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	userID, err := pathUUID(r, "userID")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RemoveProjectMember(r.Context(), actor, projectID, userID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- team handlers -------------------------------------------------------

func (a *api) createTeam(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createTeamRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	team, err := a.svc.CreateTeam(r.Context(), actor, domain.NewTeam{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toTeamDTO(team))
}

func (a *api) listTeams(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	teams, err := a.svc.ListTeams(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]teamDTO, 0, len(teams))
	for _, t := range teams {
		items = append(items, toTeamDTO(t))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) getTeam(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	team, err := a.svc.GetTeam(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTeamDTO(team))
}

func (a *api) addTeamMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	teamID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req addMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "invalid_id", "invalid user_id"))
		return
	}
	member, err := a.svc.AddTeamMember(r.Context(), actor, teamID, userID, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toTeamMemberRowDTO(member))
}

func (a *api) listTeamMembers(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	teamID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	members, err := a.svc.ListTeamMembers(r.Context(), actor, teamID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]membershipDTO, 0, len(members))
	for _, m := range members {
		items = append(items, membershipDTO{
			User:        toUserDTO(m.User),
			Role:        m.Role,
			MemberSince: ts(m.MemberSince),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) removeTeamMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	teamID, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	userID, err := pathUUID(r, "userID")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RemoveTeamMember(r.Context(), actor, teamID, userID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- service account handlers --------------------------------------------

func (a *api) createServiceAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createServiceAccountRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	sa, err := a.svc.CreateServiceAccount(r.Context(), actor, domain.NewServiceAccount{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toServiceAccountDTO(sa))
}

func (a *api) listServiceAccounts(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	accounts, err := a.svc.ListServiceAccounts(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]serviceAccountDTO, 0, len(accounts))
	for _, sa := range accounts {
		items = append(items, toServiceAccountDTO(sa))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) revokeServiceAccount(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RevokeServiceAccount(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- credential handlers -------------------------------------------------

func (a *api) createCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createCredentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	saID, err := parseOptionalUUID(req.ServiceAccountID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	cred, rawKey, err := a.svc.CreateCredential(r.Context(), actor, req.Name, req.Scope, saID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createCredentialResponse{
		Credential: toCredentialDTO(cred),
		APIKey:     rawKey,
	})
}

func (a *api) listCredentials(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	credentials, err := a.svc.ListCredentials(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]credentialDTO, 0, len(credentials))
	for _, c := range credentials {
		items = append(items, toCredentialDTO(c))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items})
}

func (a *api) revokeCredential(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.svc.RevokeCredential(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
