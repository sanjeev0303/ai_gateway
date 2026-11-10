// Package platformadmin runs the super-admin console: cross-organization
// reads and the few platform actions (suspend an organization, disable a user,
// change a plan, revoke keys, retire a model). A platform admin is a user
// whose email is on the PLATFORM_ADMIN_EMAILS allowlist; nothing inside the
// application can grant that.
package platformadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/ports/repositories"
)

// ErrNotPlatformAdmin is returned for every caller that is not a platform
// admin, whatever the reason, so the console's existence leaks nothing.
var ErrNotPlatformAdmin = errors.New("platformadmin: not a platform admin")

const maxReasonLength = 500

// CredentialInvalidator evicts a gateway key from every gateway's cache, so a
// suspension or revocation takes effect at once instead of after the cache TTL.
type CredentialInvalidator interface {
	InvalidateCredential(orgID, credID uuid.UUID)
}

type ServiceDeps struct {
	Repo               repositories.PlatformAdminRepository
	Monitor            repositories.PlatformMonitorRepository
	Users              repositories.UserRepository
	Plans              repositories.PlanRepository
	Subscriptions      repositories.SubscriptionRepository
	GatewayCredentials repositories.GatewayCredentialRepository
	RefreshTokens      repositories.RefreshTokenRepository
	Models             repositories.ModelRepository
	ProviderHealth     repositories.ProviderHealthRepository
	Audits             repositories.AuditLogRepository
	Invalidator        CredentialInvalidator
	// AdminEmails is the PLATFORM_ADMIN_EMAILS allowlist.
	AdminEmails []string
	Log         *slog.Logger
}

type Service struct {
	repo               repositories.PlatformAdminRepository
	monitor            repositories.PlatformMonitorRepository
	users              repositories.UserRepository
	plans              repositories.PlanRepository
	subscriptions      repositories.SubscriptionRepository
	gatewayCredentials repositories.GatewayCredentialRepository
	refreshTokens      repositories.RefreshTokenRepository
	models             repositories.ModelRepository
	providerHealth     repositories.ProviderHealthRepository
	audits             repositories.AuditLogRepository
	invalidator        CredentialInvalidator
	admins             map[string]bool
	log                *slog.Logger
	now                func() time.Time
}

func NewService(deps ServiceDeps) *Service {
	admins := make(map[string]bool)
	for _, email := range deps.AdminEmails {
		email = normalizeEmail(email)
		if email != "" {
			admins[email] = true
		}
	}
	log := deps.Log
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo:               deps.Repo,
		monitor:            deps.Monitor,
		users:              deps.Users,
		plans:              deps.Plans,
		subscriptions:      deps.Subscriptions,
		gatewayCredentials: deps.GatewayCredentials,
		refreshTokens:      deps.RefreshTokens,
		models:             deps.Models,
		providerHealth:     deps.ProviderHealth,
		audits:             deps.Audits,
		invalidator:        deps.Invalidator,
		admins:             admins,
		log:                log,
		now:                time.Now,
	}
}

// AdminEmails returns the configured allowlist, for the startup check that
// warns about addresses nobody has registered yet.
func (s *Service) AdminEmails() []string {
	emails := make([]string, 0, len(s.admins))
	for email := range s.admins {
		emails = append(emails, email)
	}
	return emails
}

// Authorize returns the admin's own user record when actor is a platform
// admin. The email is read fresh from the user row, never from token claims,
// so a renamed or disabled account loses access on its next request.
func (s *Service) Authorize(ctx context.Context, actor domain.Actor) (domain.User, error) {
	if len(s.admins) == 0 || actor.Kind != domain.ActorKindUser || actor.UserID == uuid.Nil {
		return domain.User{}, ErrNotPlatformAdmin
	}
	user, err := s.users.GetByID(ctx, actor.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return domain.User{}, ErrNotPlatformAdmin
		}
		return domain.User{}, err
	}
	if user.Status != domain.UserStatusActive || !s.admins[normalizeEmail(user.Email)] {
		return domain.User{}, ErrNotPlatformAdmin
	}
	return user, nil
}

// --- Reads -------------------------------------------------------------

// Overview is the platform headline plus the background queue's health.
type Overview struct {
	Window   time.Duration
	Stats    domain.PlatformOverview
	Outbox   domain.PlatformOutbox
	Health   []domain.ProviderHealthState
	Daily    []domain.PlatformUsageDay
	TopOrgs  []domain.PlatformUsageRow
	Plans    []domain.PlatformPlanSummary
	Activity []domain.PlatformAuditEntry
	// LiveAlerts counts platform alerts not yet resolved.
	LiveAlerts int64
}

func (s *Service) Overview(ctx context.Context, window time.Duration) (Overview, error) {
	since := s.now().Add(-window)
	out := Overview{Window: window}
	var err error
	if out.Stats, err = s.repo.Overview(ctx, since); err != nil {
		return out, err
	}
	if out.Outbox, err = s.repo.Outbox(ctx); err != nil {
		return out, err
	}
	if out.Health, err = s.providerHealth.List(ctx); err != nil {
		return out, err
	}
	if out.Daily, err = s.repo.UsageDaily(ctx, since); err != nil {
		return out, err
	}
	if out.TopOrgs, err = s.repo.UsageBreakdown(ctx, since, "organization", 5); err != nil {
		return out, err
	}
	if out.Plans, err = s.repo.PlanSummary(ctx); err != nil {
		return out, err
	}
	if out.Activity, _, err = s.repo.ListAudit(ctx, 8, 0); err != nil {
		return out, err
	}
	if s.monitor != nil {
		if out.LiveAlerts, err = s.monitor.CountLiveAlerts(ctx); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (s *Service) ListOrganizations(ctx context.Context, search, status string, window time.Duration, limit, offset int32) ([]domain.PlatformOrganization, int64, error) {
	if status != "" && status != domain.OrgStatusActive && status != domain.OrgStatusSuspended {
		return nil, 0, invalid("invalid_status", "status must be active or suspended")
	}
	return s.repo.ListOrganizations(ctx, strings.TrimSpace(search), status, s.now().Add(-window), limit, offset)
}

// OrganizationDetail is one organization with its members and keys.
type OrganizationDetail struct {
	Organization domain.PlatformOrganization
	Members      []domain.PlatformOrgMember
	Keys         []domain.GatewayCredential
}

func (s *Service) GetOrganization(ctx context.Context, orgID uuid.UUID, window time.Duration) (OrganizationDetail, error) {
	var out OrganizationDetail
	var err error
	if out.Organization, err = s.repo.GetOrganization(ctx, orgID, s.now().Add(-window)); err != nil {
		return out, err
	}
	if out.Members, err = s.repo.ListOrganizationMembers(ctx, orgID); err != nil {
		return out, err
	}
	if out.Keys, err = s.gatewayCredentials.List(ctx, orgID); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) ListUsers(ctx context.Context, search, status string, limit, offset int32) ([]domain.PlatformUser, int64, error) {
	if status != "" && status != domain.UserStatusActive && status != domain.UserStatusDisabled {
		return nil, 0, invalid("invalid_status", "status must be active or disabled")
	}
	return s.repo.ListUsers(ctx, strings.TrimSpace(search), status, limit, offset)
}

// Usage is platform traffic over a window.
type Usage struct {
	Window     time.Duration
	Daily      []domain.PlatformUsageDay
	ByProvider []domain.PlatformUsageRow
	ByModel    []domain.PlatformUsageRow
	ByOrg      []domain.PlatformUsageRow
}

func (s *Service) Usage(ctx context.Context, window time.Duration) (Usage, error) {
	since := s.now().Add(-window)
	out := Usage{Window: window}
	var err error
	if out.Daily, err = s.repo.UsageDaily(ctx, since); err != nil {
		return out, err
	}
	if out.ByProvider, err = s.repo.UsageBreakdown(ctx, since, "provider", 20); err != nil {
		return out, err
	}
	if out.ByModel, err = s.repo.UsageBreakdown(ctx, since, "model", 20); err != nil {
		return out, err
	}
	if out.ByOrg, err = s.repo.UsageBreakdown(ctx, since, "organization", 20); err != nil {
		return out, err
	}
	return out, nil
}

// Billing is the platform's subscription and invoice picture.
type Billing struct {
	Plans    []domain.PlatformPlanSummary
	Invoices []domain.PlatformInvoice
}

func (s *Service) Billing(ctx context.Context) (Billing, error) {
	var out Billing
	var err error
	if out.Plans, err = s.repo.PlanSummary(ctx); err != nil {
		return out, err
	}
	if out.Invoices, err = s.repo.RecentInvoices(ctx, 50); err != nil {
		return out, err
	}
	return out, nil
}

// System is the health of the parts of the platform the control API can see.
type System struct {
	Outbox domain.PlatformOutbox
	Health []domain.ProviderHealthState
}

func (s *Service) System(ctx context.Context) (System, error) {
	var out System
	var err error
	if out.Outbox, err = s.repo.Outbox(ctx); err != nil {
		return out, err
	}
	if out.Health, err = s.providerHealth.List(ctx); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) ListModels(ctx context.Context) ([]domain.Model, error) {
	return s.models.ListAll(ctx)
}

func (s *Service) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx)
}

func (s *Service) ListAudit(ctx context.Context, limit, offset int32) ([]domain.PlatformAuditEntry, int64, error) {
	return s.repo.ListAudit(ctx, limit, offset)
}

// ServicesView is the latest reading of every service and the last hour of
// readings, for the health page's figures and charts.
type ServicesView struct {
	Latest []domain.ServiceSample
	Series []domain.ServiceSample
}

func (s *Service) Services(ctx context.Context) (ServicesView, error) {
	var out ServicesView
	var err error
	if out.Latest, err = s.monitor.LatestSamples(ctx); err != nil {
		return out, err
	}
	if out.Series, err = s.monitor.SampleSeries(ctx, s.now().Add(-time.Hour)); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) Alerts(ctx context.Context, status string, limit, offset int32) ([]domain.PlatformAlert, int64, error) {
	switch status {
	case "", "live", domain.PlatformAlertOpen, domain.PlatformAlertAcknowledged, domain.PlatformAlertResolved:
	default:
		return nil, 0, invalid("invalid_status", "status must be live, open, acknowledged or resolved")
	}
	return s.monitor.ListAlerts(ctx, status, limit, offset)
}

// AcknowledgeAlert marks an alert as seen. It stays listed until the monitor
// sees the problem clear and resolves it.
func (s *Service) AcknowledgeAlert(ctx context.Context, admin domain.User, alertID uuid.UUID) error {
	if err := s.monitor.AcknowledgeAlert(ctx, alertID, admin.Email); err != nil {
		return err
	}
	return s.record(ctx, admin, "platform.alert_acknowledged", "alert", &alertID, nil, "Acknowledged in the admin console", nil)
}

func (s *Service) Logs(ctx context.Context, filter domain.ServiceLogFilter, limit, offset int32) ([]domain.ServiceLog, int64, error) {
	filter.Level = strings.ToUpper(strings.TrimSpace(filter.Level))
	if filter.Level != "" && filter.Level != "WARN" && filter.Level != "ERROR" {
		return nil, 0, invalid("invalid_level", "level must be WARN or ERROR")
	}
	filter.Search = strings.TrimSpace(filter.Search)
	return s.monitor.ListLogs(ctx, filter, limit, offset)
}

func (s *Service) TenantAudit(ctx context.Context, orgID *uuid.UUID, actionPrefix string, limit, offset int32) ([]domain.TenantAuditEntry, int64, error) {
	return s.repo.TenantAudit(ctx, orgID, strings.TrimSpace(actionPrefix), limit, offset)
}

func (s *Service) SecurityEvents(ctx context.Context, severity string, limit, offset int32) ([]domain.PlatformSecurityEvent, int64, error) {
	return s.repo.SecurityEvents(ctx, strings.TrimSpace(severity), limit, offset)
}

// Revenue is money actually taken: paid invoices, by month and by customer.
type Revenue struct {
	Summary   domain.RevenueSummary
	Monthly   []domain.RevenueMonth
	Customers []domain.PayingCustomer
}

func (s *Service) Revenue(ctx context.Context) (Revenue, error) {
	var out Revenue
	var err error
	if out.Summary, err = s.repo.RevenueSummary(ctx); err != nil {
		return out, err
	}
	if out.Monthly, err = s.repo.RevenueMonthly(ctx, 12); err != nil {
		return out, err
	}
	if out.Customers, err = s.repo.PayingCustomers(ctx, 100); err != nil {
		return out, err
	}
	return out, nil
}

func (s *Service) Payments(ctx context.Context, orgID *uuid.UUID, limit, offset int32) ([]domain.PlatformPayment, int64, error) {
	return s.repo.Payments(ctx, orgID, limit, offset)
}

// --- Actions -------------------------------------------------------------
//
// Every action takes the admin returned by Authorize and a reason, and
// records itself in platform_audit_log (and in the organization's own audit
// log when it targets one).

// SuspendOrganization stops an organization: its gateway keys stop resolving
// and its members' console requests are refused. Data is untouched.
func (s *Service) SuspendOrganization(ctx context.Context, admin domain.User, orgID uuid.UUID, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	if err := s.repo.SetOrganizationStatus(ctx, orgID, domain.OrgStatusSuspended); err != nil {
		return err
	}
	// The keys stay active in the table (reactivation restores them); the
	// gateway's resolver now refuses them, but cached copies must go.
	keys, err := s.gatewayCredentials.List(ctx, orgID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if key.Status == domain.GatewayCredentialStatusActive {
			s.evict(orgID, key.ID)
		}
	}
	return s.record(ctx, admin, "platform.organization_suspended", "organization", &orgID, &orgID, reason, nil)
}

func (s *Service) ReactivateOrganization(ctx context.Context, admin domain.User, orgID uuid.UUID, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	if err := s.repo.SetOrganizationStatus(ctx, orgID, domain.OrgStatusActive); err != nil {
		return err
	}
	return s.record(ctx, admin, "platform.organization_reactivated", "organization", &orgID, &orgID, reason, nil)
}

func (s *Service) SetOrganizationPlan(ctx context.Context, admin domain.User, orgID uuid.UUID, planCode, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	plans, err := s.plans.List(ctx)
	if err != nil {
		return err
	}
	var plan *domain.Plan
	for i := range plans {
		if plans[i].Code == planCode {
			plan = &plans[i]
			break
		}
	}
	if plan == nil {
		return invalid("unknown_plan", "no plan with that code")
	}
	// GetOrganization first so an unknown org is a 404, not a subscription
	// row inserted for an organization that does not exist.
	if _, err := s.repo.GetOrganization(ctx, orgID, s.now()); err != nil {
		return err
	}
	if err := s.subscriptions.SetPlan(ctx, orgID, plan.ID); err != nil {
		return err
	}
	return s.record(ctx, admin, "platform.organization_plan_changed", "organization", &orgID, &orgID, reason,
		map[string]any{"plan_code": plan.Code})
}

// RevokeOrganizationKeys revokes every active gateway key the organization
// holds. Unlike suspension this is permanent: keys must be reissued.
func (s *Service) RevokeOrganizationKeys(ctx context.Context, admin domain.User, orgID uuid.UUID, reason string) (int, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return 0, err
	}
	keys, err := s.gatewayCredentials.List(ctx, orgID)
	if err != nil {
		return 0, err
	}
	revoked := 0
	for _, key := range keys {
		if key.Status != domain.GatewayCredentialStatusActive {
			continue
		}
		if _, err := s.gatewayCredentials.Revoke(ctx, orgID, key.ID); err != nil {
			return revoked, fmt.Errorf("revoke key %s: %w", key.ID, err)
		}
		s.evict(orgID, key.ID)
		revoked++
	}
	return revoked, s.record(ctx, admin, "platform.organization_keys_revoked", "organization", &orgID, &orgID, reason,
		map[string]any{"revoked": revoked})
}

// DisableUser signs the user out everywhere and blocks them: sign-in and
// refresh already refuse disabled users, the control API now checks status
// on every request, and their own gateway keys stop resolving.
func (s *Service) DisableUser(ctx context.Context, admin domain.User, userID uuid.UUID, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	if userID == admin.ID {
		return invalid("cannot_disable_self", "you cannot disable your own account")
	}
	if err := s.repo.SetUserStatus(ctx, userID, domain.UserStatusDisabled); err != nil {
		return err
	}
	if err := s.refreshTokens.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	keys, err := s.repo.UserGatewayCredentials(ctx, userID)
	if err != nil {
		return err
	}
	for _, key := range keys {
		s.evict(key.OrganizationID, key.ID)
	}
	return s.recordForUser(ctx, admin, "platform.user_disabled", userID, reason)
}

func (s *Service) EnableUser(ctx context.Context, admin domain.User, userID uuid.UUID, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	if err := s.repo.SetUserStatus(ctx, userID, domain.UserStatusActive); err != nil {
		return err
	}
	return s.recordForUser(ctx, admin, "platform.user_enabled", userID, reason)
}

// SetModelActive switches a model on or off in the shared catalog, for every
// organization. This is the operator path tenant syncs no longer have.
func (s *Service) SetModelActive(ctx context.Context, admin domain.User, modelID uuid.UUID, active bool, reason string) error {
	reason, err := cleanReason(reason)
	if err != nil {
		return err
	}
	if err := s.repo.SetModelActive(ctx, modelID, active); err != nil {
		return err
	}
	action := "platform.model_deactivated"
	if active {
		action = "platform.model_activated"
	}
	return s.record(ctx, admin, action, "model", &modelID, nil, reason, nil)
}

// --- helpers ---------------------------------------------------------

func (s *Service) evict(orgID, credID uuid.UUID) {
	if s.invalidator != nil {
		s.invalidator.InvalidateCredential(orgID, credID)
	}
}

// recordForUser audits a user action into the platform log and into each of
// the user's organizations, so their owners can see it.
func (s *Service) recordForUser(ctx context.Context, admin domain.User, action string, userID uuid.UUID, reason string) error {
	if err := s.record(ctx, admin, action, "user", &userID, nil, reason, nil); err != nil {
		return err
	}
	orgs, err := s.users.ListOrganizationsForUser(ctx, userID)
	if err != nil {
		return err
	}
	for _, org := range orgs {
		s.tenantAudit(ctx, admin, org.Organization.ID, action, "user", &userID, reason, nil)
	}
	return nil
}

// record writes the platform audit row, which must succeed: an action the
// platform cannot account for is reported as failed. The tenant copy is best
// effort.
func (s *Service) record(ctx context.Context, admin domain.User, action, targetType string, targetID, orgID *uuid.UUID, reason string, detail map[string]any) error {
	var detailJSON []byte
	if detail != nil {
		detailJSON, _ = json.Marshal(detail)
	}
	err := s.repo.AppendAudit(ctx, domain.NewPlatformAuditEntry{
		ActorUserID:    admin.ID,
		ActorEmail:     admin.Email,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		OrganizationID: orgID,
		Reason:         reason,
		Detail:         detailJSON,
	})
	if err != nil {
		return fmt.Errorf("record platform audit: %w", err)
	}
	if orgID != nil {
		s.tenantAudit(ctx, admin, *orgID, action, targetType, targetID, reason, detail)
	}
	return nil
}

func (s *Service) tenantAudit(ctx context.Context, admin domain.User, orgID uuid.UUID, action, targetType string, targetID *uuid.UUID, reason string, detail map[string]any) {
	if s.audits == nil {
		return
	}
	after := map[string]any{"reason": reason, "platform_admin": admin.Email}
	for k, v := range detail {
		after[k] = v
	}
	afterJSON, _ := json.Marshal(after)
	adminID := admin.ID
	if _, err := s.audits.Append(ctx, orgID, domain.NewAuditLog{
		ActorUserID:  &adminID,
		Action:       action,
		ResourceType: targetType,
		ResourceID:   targetID,
		After:        afterJSON,
	}); err != nil {
		s.log.WarnContext(ctx, "platformadmin: tenant audit copy failed", "org_id", orgID, "action", action, "error", err)
	}
}

func cleanReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", invalid("reason_required", "a reason is required")
	}
	if len(reason) > maxReasonLength {
		return "", invalid("reason_too_long", fmt.Sprintf("reason must be at most %d characters", maxReasonLength))
	}
	return reason, nil
}

func invalid(code, msg string) error {
	return perrors.New(perrors.KindInvalidArgument, code, msg)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
