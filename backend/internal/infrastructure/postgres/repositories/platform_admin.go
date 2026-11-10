package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type platformAdminRepo struct {
	store *postgres.Store
}

// NewPlatformAdminRepository returns the cross-tenant repository behind the
// super-admin console. Reads call the SECURITY DEFINER app.platform_*
// functions from migration 00043.
func NewPlatformAdminRepository(store *postgres.Store) ports.PlatformAdminRepository {
	return &platformAdminRepo{store: store}
}

func (r *platformAdminRepo) Overview(ctx context.Context, since time.Time) (domain.PlatformOverview, error) {
	var o domain.PlatformOverview
	err := r.store.Pool().QueryRow(ctx, `SELECT * FROM app.platform_overview($1)`, since).Scan(
		&o.OrgsTotal, &o.OrgsSuspended, &o.UsersTotal, &o.UsersDisabled,
		&o.Requests, &o.FailedRequests, &o.Tokens, &o.CostMicros,
		&o.ActiveKeys, &o.PaidSubscriptions, &o.MRRMicros,
	)
	if err != nil {
		return o, fmt.Errorf("platform overview: %w", err)
	}
	return o, nil
}

func (r *platformAdminRepo) ListOrganizations(ctx context.Context, search, status string, since time.Time, limit, offset int32) ([]domain.PlatformOrganization, int64, error) {
	rows, err := r.store.Pool().Query(ctx,
		`SELECT * FROM app.platform_organizations($1, $2, $3, $4, $5)`,
		search, status, since, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list platform organizations: %w", err)
	}
	defer rows.Close()

	orgs := []domain.PlatformOrganization{}
	var total int64
	for rows.Next() {
		var o domain.PlatformOrganization
		var planCode, planName, subStatus *string
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.Status, &o.CreatedAt,
			&planCode, &planName, &subStatus,
			&o.Members, &o.ActiveKeys, &o.Requests, &o.CostMicros, &o.LastRequestAt,
			&total); err != nil {
			return nil, 0, fmt.Errorf("scan platform organization: %w", err)
		}
		o.PlanCode = derefString(planCode)
		o.PlanName = derefString(planName)
		o.SubscriptionStatus = derefString(subStatus)
		orgs = append(orgs, o)
	}
	return orgs, total, rows.Err()
}

func (r *platformAdminRepo) GetOrganization(ctx context.Context, id uuid.UUID, since time.Time) (domain.PlatformOrganization, error) {
	// platform_organizations matches an exact ID in its search argument.
	orgs, _, err := r.ListOrganizations(ctx, id.String(), "", since, 1, 0)
	if err != nil {
		return domain.PlatformOrganization{}, err
	}
	if len(orgs) == 0 || orgs[0].ID != id {
		return domain.PlatformOrganization{}, ports.ErrNotFound
	}
	return orgs[0], nil
}

func (r *platformAdminRepo) ListOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]domain.PlatformOrgMember, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_organization_members($1)`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list platform organization members: %w", err)
	}
	defer rows.Close()

	members := []domain.PlatformOrgMember{}
	for rows.Next() {
		var m domain.PlatformOrgMember
		if err := rows.Scan(&m.UserID, &m.Email, &m.DisplayName, &m.Role, &m.Status, &m.JoinedAt, &m.LastLoginAt); err != nil {
			return nil, fmt.Errorf("scan platform organization member: %w", err)
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *platformAdminRepo) ListUsers(ctx context.Context, search, status string, limit, offset int32) ([]domain.PlatformUser, int64, error) {
	rows, err := r.store.Pool().Query(ctx,
		`SELECT * FROM app.platform_users($1, $2, $3, $4)`,
		search, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list platform users: %w", err)
	}
	defer rows.Close()

	users := []domain.PlatformUser{}
	var total int64
	for rows.Next() {
		var u domain.PlatformUser
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.LastLoginAt,
			&u.OrgCount, &u.OrgNames, &total); err != nil {
			return nil, 0, fmt.Errorf("scan platform user: %w", err)
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *platformAdminRepo) UsageDaily(ctx context.Context, since time.Time) ([]domain.PlatformUsageDay, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_usage_daily($1)`, since)
	if err != nil {
		return nil, fmt.Errorf("platform usage daily: %w", err)
	}
	defer rows.Close()

	days := []domain.PlatformUsageDay{}
	for rows.Next() {
		var d domain.PlatformUsageDay
		if err := rows.Scan(&d.Day, &d.Requests, &d.FailedRequests, &d.Tokens, &d.CostMicros); err != nil {
			return nil, fmt.Errorf("scan platform usage day: %w", err)
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

func (r *platformAdminRepo) UsageBreakdown(ctx context.Context, since time.Time, dimension string, limit int32) ([]domain.PlatformUsageRow, error) {
	rows, err := r.store.Pool().Query(ctx,
		`SELECT * FROM app.platform_usage_breakdown($1, $2, $3)`, since, dimension, limit)
	if err != nil {
		return nil, fmt.Errorf("platform usage breakdown: %w", err)
	}
	defer rows.Close()

	out := []domain.PlatformUsageRow{}
	for rows.Next() {
		var u domain.PlatformUsageRow
		if err := rows.Scan(&u.Key, &u.Label, &u.Requests, &u.Tokens, &u.CostMicros, &u.AvgLatencyMs); err != nil {
			return nil, fmt.Errorf("scan platform usage row: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *platformAdminRepo) PlanSummary(ctx context.Context) ([]domain.PlatformPlanSummary, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_plan_summary()`)
	if err != nil {
		return nil, fmt.Errorf("platform plan summary: %w", err)
	}
	defer rows.Close()

	plans := []domain.PlatformPlanSummary{}
	for rows.Next() {
		var p domain.PlatformPlanSummary
		if err := rows.Scan(&p.PlanCode, &p.PlanName, &p.PriceMicros, &p.Currency, &p.Subscriptions); err != nil {
			return nil, fmt.Errorf("scan platform plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func (r *platformAdminRepo) RecentInvoices(ctx context.Context, limit int32) ([]domain.PlatformInvoice, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_invoices($1)`, limit)
	if err != nil {
		return nil, fmt.Errorf("platform invoices: %w", err)
	}
	defer rows.Close()

	invoices := []domain.PlatformInvoice{}
	for rows.Next() {
		var i domain.PlatformInvoice
		if err := rows.Scan(&i.ID, &i.OrganizationID, &i.OrganizationName, &i.Number, &i.Status,
			&i.Currency, &i.TotalMicros, &i.IssuedAt); err != nil {
			return nil, fmt.Errorf("scan platform invoice: %w", err)
		}
		invoices = append(invoices, i)
	}
	return invoices, rows.Err()
}

func (r *platformAdminRepo) AppendAudit(ctx context.Context, e domain.NewPlatformAuditEntry) error {
	detail := e.Detail
	if len(detail) == 0 {
		detail = []byte("{}")
	}
	_, err := r.store.Pool().Exec(ctx, `
		INSERT INTO platform_audit_log (actor_user_id, actor_email, action, target_type, target_id, organization_id, reason, detail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ActorUserID, e.ActorEmail, e.Action, e.TargetType, e.TargetID, e.OrganizationID, e.Reason, detail)
	if err != nil {
		return fmt.Errorf("append platform audit: %w", err)
	}
	return nil
}

// ListAudit reads platform_audit_log directly: it is a platform table with no
// RLS. The organization name is joined for display and may be gone.
func (r *platformAdminRepo) ListAudit(ctx context.Context, limit, offset int32) ([]domain.PlatformAuditEntry, int64, error) {
	rows, err := r.store.Pool().Query(ctx, `
		SELECT a.id, a.actor_user_id, a.actor_email, a.action, a.target_type, a.target_id,
		       a.organization_id, coalesce(o.name, ''), a.reason, a.detail, a.created_at,
		       count(*) OVER ()
		FROM platform_audit_log a
		LEFT JOIN organizations o ON o.id = a.organization_id
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("platform audit: %w", err)
	}
	defer rows.Close()

	entries := []domain.PlatformAuditEntry{}
	var total int64
	for rows.Next() {
		var e domain.PlatformAuditEntry
		if err := rows.Scan(&e.ID, &e.ActorUserID, &e.ActorEmail, &e.Action, &e.TargetType, &e.TargetID,
			&e.OrganizationID, &e.OrganizationName, &e.Reason, &e.Detail, &e.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan platform audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// Outbox reads event_outbox directly: it is a global queue with no tenant
// column, so it sits outside RLS.
func (r *platformAdminRepo) Outbox(ctx context.Context) (domain.PlatformOutbox, error) {
	var o domain.PlatformOutbox
	var lastError *string
	err := r.store.Pool().QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE attempts > 0),
		       min(created_at),
		       coalesce(extract(epoch FROM now() - min(created_at)), 0)::bigint,
		       (SELECT last_error FROM event_outbox WHERE last_error IS NOT NULL ORDER BY next_attempt_at DESC LIMIT 1)
		FROM event_outbox`).Scan(&o.Pending, &o.Retrying, &o.OldestPending, &o.OldestAgeSeconds, &lastError)
	if err != nil {
		return o, fmt.Errorf("platform outbox: %w", err)
	}
	o.LastError = derefString(lastError)
	return o, nil
}

func (r *platformAdminRepo) UserGatewayCredentials(ctx context.Context, userID uuid.UUID) ([]domain.PlatformCredentialRef, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_user_gateway_credentials($1)`, userID)
	if err != nil {
		return nil, fmt.Errorf("platform user gateway credentials: %w", err)
	}
	defer rows.Close()

	refs := []domain.PlatformCredentialRef{}
	for rows.Next() {
		var ref domain.PlatformCredentialRef
		if err := rows.Scan(&ref.ID, &ref.OrganizationID); err != nil {
			return nil, fmt.Errorf("scan platform credential ref: %w", err)
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

func (r *platformAdminRepo) SetOrganizationStatus(ctx context.Context, orgID uuid.UUID, status string) error {
	return r.execOne(ctx, `UPDATE organizations SET status = $2, updated_at = now() WHERE id = $1`, orgID, status)
}

func (r *platformAdminRepo) SetUserStatus(ctx context.Context, userID uuid.UUID, status string) error {
	return r.execOne(ctx, `UPDATE users SET status = $2, updated_at = now() WHERE id = $1`, userID, status)
}

func (r *platformAdminRepo) SetModelActive(ctx context.Context, modelID uuid.UUID, active bool) error {
	status := domain.ModelStatusInactive
	if active {
		status = domain.ModelStatusActive
	}
	return r.execOne(ctx, `UPDATE models SET is_active = $2, status = $3, updated_at = now() WHERE id = $1`, modelID, active, status)
}

// execOne runs an UPDATE that must touch exactly one row.
func (r *platformAdminRepo) execOne(ctx context.Context, sql string, args ...any) error {
	tag, err := r.store.Pool().Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %v", ports.ErrNotFound, pgx.ErrNoRows)
	}
	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *platformAdminRepo) TenantAudit(ctx context.Context, orgID *uuid.UUID, actionPrefix string, limit, offset int32) ([]domain.TenantAuditEntry, int64, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_tenant_audit($1, $2, $3, $4)`, orgID, actionPrefix, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("platform tenant audit: %w", err)
	}
	defer rows.Close()

	entries := []domain.TenantAuditEntry{}
	var total int64
	for rows.Next() {
		var e domain.TenantAuditEntry
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.OrganizationName, &e.ActorEmail, &e.Action,
			&e.ResourceType, &e.ResourceID, &e.After, &e.IPAddress, &e.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan tenant audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (r *platformAdminRepo) SecurityEvents(ctx context.Context, severity string, limit, offset int32) ([]domain.PlatformSecurityEvent, int64, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_security_events($1, $2, $3)`, severity, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("platform security events: %w", err)
	}
	defer rows.Close()

	events := []domain.PlatformSecurityEvent{}
	var total int64
	for rows.Next() {
		var e domain.PlatformSecurityEvent
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.OrganizationName, &e.Category, &e.Severity,
			&e.Title, &e.Detail, &e.Source, &e.CreatedAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan platform security event: %w", err)
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *platformAdminRepo) RevenueSummary(ctx context.Context) (domain.RevenueSummary, error) {
	var s domain.RevenueSummary
	err := r.store.Pool().QueryRow(ctx, `SELECT * FROM app.platform_revenue_summary()`).Scan(
		&s.LifetimeMicros, &s.Last30DMicros, &s.ThisMonthMicros, &s.Payments, &s.PayingOrgs)
	if err != nil {
		return s, fmt.Errorf("platform revenue summary: %w", err)
	}
	return s, nil
}

func (r *platformAdminRepo) RevenueMonthly(ctx context.Context, months int32) ([]domain.RevenueMonth, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_revenue_monthly($1)`, months)
	if err != nil {
		return nil, fmt.Errorf("platform revenue monthly: %w", err)
	}
	defer rows.Close()

	out := []domain.RevenueMonth{}
	for rows.Next() {
		var m domain.RevenueMonth
		if err := rows.Scan(&m.Month, &m.RevenueMicros, &m.Payments); err != nil {
			return nil, fmt.Errorf("scan revenue month: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *platformAdminRepo) Payments(ctx context.Context, orgID *uuid.UUID, limit, offset int32) ([]domain.PlatformPayment, int64, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_payments($1, $2, $3)`, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("platform payments: %w", err)
	}
	defer rows.Close()

	payments := []domain.PlatformPayment{}
	var total int64
	for rows.Next() {
		var p domain.PlatformPayment
		if err := rows.Scan(&p.ID, &p.OrganizationID, &p.OrganizationName, &p.Number, &p.TotalMicros,
			&p.Currency, &p.Provider, &p.PaymentID, &p.PaidAt, &total); err != nil {
			return nil, 0, fmt.Errorf("scan platform payment: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, total, rows.Err()
}

func (r *platformAdminRepo) PayingCustomers(ctx context.Context, limit int32) ([]domain.PayingCustomer, error) {
	rows, err := r.store.Pool().Query(ctx, `SELECT * FROM app.platform_paying_customers($1)`, limit)
	if err != nil {
		return nil, fmt.Errorf("platform paying customers: %w", err)
	}
	defer rows.Close()

	customers := []domain.PayingCustomer{}
	for rows.Next() {
		var c domain.PayingCustomer
		if err := rows.Scan(&c.OrganizationID, &c.OrganizationName, &c.PlanName, &c.LifetimeMicros,
			&c.Payments, &c.LastPaidAt); err != nil {
			return nil, fmt.Errorf("scan paying customer: %w", err)
		}
		customers = append(customers, c)
	}
	return customers, rows.Err()
}
