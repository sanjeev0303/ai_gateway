package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// PlatformAdminRepository reads and writes across every organization. Reads go
// through SECURITY DEFINER functions (migration 00043) because tenant tables
// are FORCE RLS. Only the platform admin service may use it.
type PlatformAdminRepository interface {
	Overview(ctx context.Context, since time.Time) (domain.PlatformOverview, error)
	// ListOrganizations returns one page plus the total number of matches.
	ListOrganizations(ctx context.Context, search, status string, since time.Time, limit, offset int32) ([]domain.PlatformOrganization, int64, error)
	GetOrganization(ctx context.Context, id uuid.UUID, since time.Time) (domain.PlatformOrganization, error)
	ListOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]domain.PlatformOrgMember, error)
	ListUsers(ctx context.Context, search, status string, limit, offset int32) ([]domain.PlatformUser, int64, error)
	UsageDaily(ctx context.Context, since time.Time) ([]domain.PlatformUsageDay, error)
	// UsageBreakdown groups traffic by "provider", "model" or "organization".
	UsageBreakdown(ctx context.Context, since time.Time, dimension string, limit int32) ([]domain.PlatformUsageRow, error)
	PlanSummary(ctx context.Context) ([]domain.PlatformPlanSummary, error)
	RecentInvoices(ctx context.Context, limit int32) ([]domain.PlatformInvoice, error)
	AppendAudit(ctx context.Context, entry domain.NewPlatformAuditEntry) error
	ListAudit(ctx context.Context, limit, offset int32) ([]domain.PlatformAuditEntry, int64, error)
	Outbox(ctx context.Context) (domain.PlatformOutbox, error)
	// TenantAudit reads organizations' own audit logs; orgID nil means all.
	TenantAudit(ctx context.Context, orgID *uuid.UUID, actionPrefix string, limit, offset int32) ([]domain.TenantAuditEntry, int64, error)
	SecurityEvents(ctx context.Context, severity string, limit, offset int32) ([]domain.PlatformSecurityEvent, int64, error)
	RevenueSummary(ctx context.Context) (domain.RevenueSummary, error)
	RevenueMonthly(ctx context.Context, months int32) ([]domain.RevenueMonth, error)
	Payments(ctx context.Context, orgID *uuid.UUID, limit, offset int32) ([]domain.PlatformPayment, int64, error)
	PayingCustomers(ctx context.Context, limit int32) ([]domain.PayingCustomer, error)
	UserGatewayCredentials(ctx context.Context, userID uuid.UUID) ([]domain.PlatformCredentialRef, error)

	// SetOrganizationStatus and SetUserStatus write global tables (no RLS).
	// They return ErrNotFound for an unknown ID.
	SetOrganizationStatus(ctx context.Context, orgID uuid.UUID, status string) error
	SetUserStatus(ctx context.Context, userID uuid.UUID, status string) error
	SetModelActive(ctx context.Context, modelID uuid.UUID, active bool) error
}
