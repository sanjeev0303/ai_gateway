package controlapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/platformadmin"
	"ai-governance-saas/backend/internal/domain"
	perrors "ai-governance-saas/backend/internal/platform/errors"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// The /v1/admin surface is the super-admin console. Every route answers 404
// to anyone who is not a platform admin, signed in or not.

const maxAdminPage = 100

func registerAdminRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/admin/me", a.requirePlatformAdmin(a.adminMe))
	mux.Handle("GET /v1/admin/overview", a.requirePlatformAdmin(a.adminOverview))

	mux.Handle("GET /v1/admin/organizations", a.requirePlatformAdmin(a.adminListOrganizations))
	mux.Handle("GET /v1/admin/organizations/{id}", a.requirePlatformAdmin(a.adminGetOrganization))
	mux.Handle("POST /v1/admin/organizations/{id}/suspend", a.requirePlatformAdmin(a.adminSuspendOrganization))
	mux.Handle("POST /v1/admin/organizations/{id}/reactivate", a.requirePlatformAdmin(a.adminReactivateOrganization))
	mux.Handle("PUT /v1/admin/organizations/{id}/plan", a.requirePlatformAdmin(a.adminSetOrganizationPlan))
	mux.Handle("POST /v1/admin/organizations/{id}/revoke-keys", a.requirePlatformAdmin(a.adminRevokeOrganizationKeys))

	mux.Handle("GET /v1/admin/users", a.requirePlatformAdmin(a.adminListUsers))
	mux.Handle("POST /v1/admin/users/{id}/disable", a.requirePlatformAdmin(a.adminDisableUser))
	mux.Handle("POST /v1/admin/users/{id}/enable", a.requirePlatformAdmin(a.adminEnableUser))

	mux.Handle("GET /v1/admin/usage", a.requirePlatformAdmin(a.adminUsage))
	mux.Handle("GET /v1/admin/billing", a.requirePlatformAdmin(a.adminBilling))
	mux.Handle("GET /v1/admin/plans", a.requirePlatformAdmin(a.adminPlans))
	mux.Handle("GET /v1/admin/system", a.requirePlatformAdmin(a.adminSystem))
	mux.Handle("GET /v1/admin/models", a.requirePlatformAdmin(a.adminListModels))
	mux.Handle("PATCH /v1/admin/models/{id}", a.requirePlatformAdmin(a.adminSetModelActive))
	mux.Handle("GET /v1/admin/audit", a.requirePlatformAdmin(a.adminAudit))
	mux.Handle("GET /v1/admin/tenant-audit", a.requirePlatformAdmin(a.adminTenantAudit))
	mux.Handle("GET /v1/admin/security-events", a.requirePlatformAdmin(a.adminSecurityEvents))

	mux.Handle("GET /v1/admin/services", a.requirePlatformAdmin(a.adminServices))
	mux.Handle("GET /v1/admin/alerts", a.requirePlatformAdmin(a.adminAlerts))
	mux.Handle("POST /v1/admin/alerts/{id}/acknowledge", a.requirePlatformAdmin(a.adminAcknowledgeAlert))
	mux.Handle("GET /v1/admin/logs", a.requirePlatformAdmin(a.adminLogs))

	mux.Handle("GET /v1/admin/revenue", a.requirePlatformAdmin(a.adminRevenue))
	mux.Handle("GET /v1/admin/payments", a.requirePlatformAdmin(a.adminPayments))
}

type adminHandler func(w http.ResponseWriter, r *http.Request, admin domain.User)

// requirePlatformAdmin authenticates the caller like requireAuth, then
// requires a platform admin. It deliberately skips requireAuth's organization
// check: an admin whose own organization is suspended still needs the console.
func (a *api) requirePlatformAdmin(next adminHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, _, method, err := a.authenticate(r)
		if err != nil {
			writeServiceError(w, r, platformadmin.ErrNotPlatformAdmin)
			return
		}
		admin, err := a.platformAdmin.Authorize(r.Context(), actor)
		if err != nil {
			writeServiceError(w, r, err)
			return
		}
		if !a.csrfAllowed(r, method) {
			writeServiceError(w, r, perrors.New(perrors.KindPermissionDenied, "csrf_token_invalid", "missing or invalid CSRF token"))
			return
		}
		next(w, r.WithContext(withActor(r.Context(), actor)), admin)
	})
}

// ---- wire DTOs -----------------------------------------------------------

type adminOrganizationDTO struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Status             string `json:"status"`
	CreatedAt          string `json:"created_at"`
	PlanCode           string `json:"plan_code"`
	PlanName           string `json:"plan_name"`
	SubscriptionStatus string `json:"subscription_status"`
	Members            int64  `json:"members"`
	ActiveKeys         int64  `json:"active_keys"`
	Requests           int64  `json:"requests"`
	CostMicros         int64  `json:"cost_micros"`
	LastRequestAt      string `json:"last_request_at,omitempty"`
}

func toAdminOrganizationDTO(o domain.PlatformOrganization) adminOrganizationDTO {
	return adminOrganizationDTO{
		ID:                 o.ID.String(),
		Name:               o.Name,
		Slug:               o.Slug,
		Status:             o.Status,
		CreatedAt:          ts(o.CreatedAt),
		PlanCode:           o.PlanCode,
		PlanName:           o.PlanName,
		SubscriptionStatus: o.SubscriptionStatus,
		Members:            o.Members,
		ActiveKeys:         o.ActiveKeys,
		Requests:           o.Requests,
		CostMicros:         o.CostMicros,
		LastRequestAt:      tsPtr(o.LastRequestAt),
	}
}

type adminMemberDTO struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	JoinedAt    string `json:"joined_at"`
	LastLoginAt string `json:"last_login_at,omitempty"`
}

type adminUserDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	LastLoginAt string `json:"last_login_at,omitempty"`
	OrgCount    int64  `json:"org_count"`
	OrgNames    string `json:"org_names"`
}

type adminUsageDayDTO struct {
	Day            string `json:"day"`
	Requests       int64  `json:"requests"`
	FailedRequests int64  `json:"failed_requests"`
	Tokens         int64  `json:"tokens"`
	CostMicros     int64  `json:"cost_micros"`
}

type adminUsageRowDTO struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Requests     int64  `json:"requests"`
	Tokens       int64  `json:"tokens"`
	CostMicros   int64  `json:"cost_micros"`
	AvgLatencyMs int64  `json:"avg_latency_ms"`
}

type adminPlanSummaryDTO struct {
	PlanCode      string `json:"plan_code"`
	PlanName      string `json:"plan_name"`
	PriceMicros   int64  `json:"price_micros"`
	Currency      string `json:"currency"`
	Subscriptions int64  `json:"subscriptions"`
}

type adminInvoiceDTO struct {
	ID               string `json:"id"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	Number           string `json:"number"`
	Status           string `json:"status"`
	Currency         string `json:"currency"`
	TotalMicros      int64  `json:"total_micros"`
	IssuedAt         string `json:"issued_at"`
}

type adminAuditDTO struct {
	ID               string `json:"id"`
	ActorEmail       string `json:"actor_email"`
	Action           string `json:"action"`
	TargetType       string `json:"target_type"`
	TargetID         string `json:"target_id,omitempty"`
	OrganizationID   string `json:"organization_id,omitempty"`
	OrganizationName string `json:"organization_name,omitempty"`
	Reason           string `json:"reason"`
	Detail           any    `json:"detail"`
	CreatedAt        string `json:"created_at"`
}

type adminOutboxDTO struct {
	Pending          int64  `json:"pending"`
	Retrying         int64  `json:"retrying"`
	OldestPending    string `json:"oldest_pending,omitempty"`
	OldestAgeSeconds int64  `json:"oldest_age_seconds"`
	LastError        string `json:"last_error,omitempty"`
}

type adminOverviewStatsDTO struct {
	OrgsTotal         int64 `json:"orgs_total"`
	OrgsSuspended     int64 `json:"orgs_suspended"`
	UsersTotal        int64 `json:"users_total"`
	UsersDisabled     int64 `json:"users_disabled"`
	Requests          int64 `json:"requests"`
	FailedRequests    int64 `json:"failed_requests"`
	Tokens            int64 `json:"tokens"`
	CostMicros        int64 `json:"cost_micros"`
	ActiveKeys        int64 `json:"active_keys"`
	PaidSubscriptions int64 `json:"paid_subscriptions"`
	MRRMicros         int64 `json:"mrr_micros"`
}

func usageDays(days []domain.PlatformUsageDay) []adminUsageDayDTO {
	out := make([]adminUsageDayDTO, 0, len(days))
	for _, d := range days {
		out = append(out, adminUsageDayDTO{
			Day:            d.Day.Format("2006-01-02"),
			Requests:       d.Requests,
			FailedRequests: d.FailedRequests,
			Tokens:         d.Tokens,
			CostMicros:     d.CostMicros,
		})
	}
	return out
}

func usageRows(rows []domain.PlatformUsageRow) []adminUsageRowDTO {
	out := make([]adminUsageRowDTO, 0, len(rows))
	for _, u := range rows {
		out = append(out, adminUsageRowDTO(u))
	}
	return out
}

func planSummaries(plans []domain.PlatformPlanSummary) []adminPlanSummaryDTO {
	out := make([]adminPlanSummaryDTO, 0, len(plans))
	for _, p := range plans {
		out = append(out, adminPlanSummaryDTO(p))
	}
	return out
}

func healthStates(states []domain.ProviderHealthState) []providerHealthDTO {
	out := make([]providerHealthDTO, 0, len(states))
	for _, s := range states {
		out = append(out, toProviderHealthDTO(s))
	}
	return out
}

func auditEntries(entries []domain.PlatformAuditEntry) []adminAuditDTO {
	out := make([]adminAuditDTO, 0, len(entries))
	for _, e := range entries {
		dto := adminAuditDTO{
			ID:               e.ID.String(),
			ActorEmail:       e.ActorEmail,
			Action:           e.Action,
			TargetType:       e.TargetType,
			OrganizationName: e.OrganizationName,
			Reason:           e.Reason,
			Detail:           jsonObjectOrEmpty(e.Detail),
			CreatedAt:        ts(e.CreatedAt),
		}
		if e.TargetID != nil {
			dto.TargetID = e.TargetID.String()
		}
		if e.OrganizationID != nil {
			dto.OrganizationID = e.OrganizationID.String()
		}
		out = append(out, dto)
	}
	return out
}

func toAdminOutboxDTO(o domain.PlatformOutbox) adminOutboxDTO {
	return adminOutboxDTO{
		Pending:          o.Pending,
		Retrying:         o.Retrying,
		OldestPending:    tsPtr(o.OldestPending),
		OldestAgeSeconds: o.OldestAgeSeconds,
		LastError:        o.LastError,
	}
}

// ---- reads ---------------------------------------------------------------

func (a *api) adminMe(w http.ResponseWriter, r *http.Request, admin domain.User) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"id":           admin.ID.String(),
		"email":        admin.Email,
		"display_name": admin.DisplayName,
	})
}

func (a *api) adminOverview(w http.ResponseWriter, r *http.Request, _ domain.User) {
	window, err := parseAdminWindow(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	o, err := a.platformAdmin.Overview(r.Context(), window)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"window_hours":    int(window.Hours()),
		"stats":           adminOverviewStatsDTO(o.Stats),
		"outbox":          toAdminOutboxDTO(o.Outbox),
		"provider_health": healthStates(o.Health),
		"daily":           usageDays(o.Daily),
		"top_orgs":        usageRows(o.TopOrgs),
		"plans":           planSummaries(o.Plans),
		"recent_actions":  auditEntries(o.Activity),
		"live_alerts":     o.LiveAlerts,
	})
}

func (a *api) adminListOrganizations(w http.ResponseWriter, r *http.Request, _ domain.User) {
	window, err := parseAdminWindow(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	q := r.URL.Query()
	orgs, total, err := a.platformAdmin.ListOrganizations(r.Context(), q.Get("search"), q.Get("status"), window, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]adminOrganizationDTO, 0, len(orgs))
	for _, o := range orgs {
		items = append(items, toAdminOrganizationDTO(o))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminGetOrganization(w http.ResponseWriter, r *http.Request, _ domain.User) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	window, err := parseAdminWindow(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	detail, err := a.platformAdmin.GetOrganization(r.Context(), id, window)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	members := make([]adminMemberDTO, 0, len(detail.Members))
	for _, m := range detail.Members {
		members = append(members, adminMemberDTO{
			UserID:      m.UserID.String(),
			Email:       m.Email,
			DisplayName: m.DisplayName,
			Role:        m.Role,
			Status:      m.Status,
			JoinedAt:    ts(m.JoinedAt),
			LastLoginAt: tsPtr(m.LastLoginAt),
		})
	}
	keys := make([]gatewayCredentialDTO, 0, len(detail.Keys))
	for _, k := range detail.Keys {
		keys = append(keys, toGatewayCredentialDTO(k))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"organization": toAdminOrganizationDTO(detail.Organization),
		"members":      members,
		"keys":         keys,
	})
}

func (a *api) adminListUsers(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	q := r.URL.Query()
	users, total, err := a.platformAdmin.ListUsers(r.Context(), q.Get("search"), q.Get("status"), limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]adminUserDTO, 0, len(users))
	for _, u := range users {
		items = append(items, adminUserDTO{
			ID:          u.ID.String(),
			Email:       u.Email,
			DisplayName: u.DisplayName,
			Status:      u.Status,
			CreatedAt:   ts(u.CreatedAt),
			LastLoginAt: tsPtr(u.LastLoginAt),
			OrgCount:    u.OrgCount,
			OrgNames:    u.OrgNames,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminUsage(w http.ResponseWriter, r *http.Request, _ domain.User) {
	window, err := parseAdminWindow(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	u, err := a.platformAdmin.Usage(r.Context(), window)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"window_hours": int(window.Hours()),
		"daily":        usageDays(u.Daily),
		"by_provider":  usageRows(u.ByProvider),
		"by_model":     usageRows(u.ByModel),
		"by_org":       usageRows(u.ByOrg),
	})
}

func (a *api) adminBilling(w http.ResponseWriter, r *http.Request, _ domain.User) {
	b, err := a.platformAdmin.Billing(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	invoices := make([]adminInvoiceDTO, 0, len(b.Invoices))
	for _, i := range b.Invoices {
		invoices = append(invoices, adminInvoiceDTO{
			ID:               i.ID.String(),
			OrganizationID:   i.OrganizationID.String(),
			OrganizationName: i.OrganizationName,
			Number:           i.Number,
			Status:           i.Status,
			Currency:         i.Currency,
			TotalMicros:      i.TotalMicros,
			IssuedAt:         ts(i.IssuedAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"plans":    planSummaries(b.Plans),
		"invoices": invoices,
	})
}

func (a *api) adminPlans(w http.ResponseWriter, r *http.Request, _ domain.User) {
	plans, err := a.platformAdmin.ListPlans(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]planDTO, 0, len(plans))
	for _, p := range plans {
		items = append(items, toPlanDTO(p))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *api) adminSystem(w http.ResponseWriter, r *http.Request, _ domain.User) {
	s, err := a.platformAdmin.System(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"outbox":          toAdminOutboxDTO(s.Outbox),
		"provider_health": healthStates(s.Health),
	})
}

func (a *api) adminListModels(w http.ResponseWriter, r *http.Request, _ domain.User) {
	models, err := a.platformAdmin.ListModels(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]modelDTO, 0, len(models))
	for _, m := range models {
		items = append(items, toModelDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *api) adminAudit(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	entries, total, err := a.platformAdmin.ListAudit(r.Context(), limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": auditEntries(entries), "total": total})
}

// ---- actions -------------------------------------------------------------

type adminReasonRequest struct {
	Reason string `json:"reason"`
}

func (a *api) adminSuspendOrganization(w http.ResponseWriter, r *http.Request, admin domain.User) {
	a.adminIDAction(w, r, func(id uuid.UUID, reason string) error {
		return a.platformAdmin.SuspendOrganization(r.Context(), admin, id, reason)
	})
}

func (a *api) adminReactivateOrganization(w http.ResponseWriter, r *http.Request, admin domain.User) {
	a.adminIDAction(w, r, func(id uuid.UUID, reason string) error {
		return a.platformAdmin.ReactivateOrganization(r.Context(), admin, id, reason)
	})
}

func (a *api) adminRevokeOrganizationKeys(w http.ResponseWriter, r *http.Request, admin domain.User) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req adminReasonRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	revoked, err := a.platformAdmin.RevokeOrganizationKeys(r.Context(), admin, id, req.Reason)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]int{"revoked": revoked})
}

func (a *api) adminSetOrganizationPlan(w http.ResponseWriter, r *http.Request, admin domain.User) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req struct {
		PlanCode string `json:"plan_code"`
		Reason   string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.platformAdmin.SetOrganizationPlan(r.Context(), admin, id, strings.TrimSpace(req.PlanCode), req.Reason); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminDisableUser(w http.ResponseWriter, r *http.Request, admin domain.User) {
	a.adminIDAction(w, r, func(id uuid.UUID, reason string) error {
		return a.platformAdmin.DisableUser(r.Context(), admin, id, reason)
	})
}

func (a *api) adminEnableUser(w http.ResponseWriter, r *http.Request, admin domain.User) {
	a.adminIDAction(w, r, func(id uuid.UUID, reason string) error {
		return a.platformAdmin.EnableUser(r.Context(), admin, id, reason)
	})
}

func (a *api) adminSetModelActive(w http.ResponseWriter, r *http.Request, admin domain.User) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req struct {
		Active *bool  `json:"active"`
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if req.Active == nil {
		writeServiceError(w, r, perrors.New(perrors.KindInvalidArgument, "active_required", "active is required"))
		return
	}
	if err := a.platformAdmin.SetModelActive(r.Context(), admin, id, *req.Active, req.Reason); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// adminIDAction runs an action that takes the path ID and a reason, and
// answers 204.
func (a *api) adminIDAction(w http.ResponseWriter, r *http.Request, act func(id uuid.UUID, reason string) error) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req adminReasonRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := act(id, req.Reason); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- query parsing -------------------------------------------------------

var adminWindows = map[string]time.Duration{
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
	"90d": 90 * 24 * time.Hour,
}

func parseAdminWindow(r *http.Request) (time.Duration, error) {
	raw := r.URL.Query().Get("window")
	if raw == "" {
		return adminWindows["30d"], nil
	}
	window, ok := adminWindows[raw]
	if !ok {
		return 0, invalidQuery("window must be one of 24h, 7d, 30d, 90d")
	}
	return window, nil
}

func parseAdminPage(r *http.Request) (int32, int32, error) {
	limit, err := parseIntQuery(r, "limit", 25)
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > maxAdminPage {
		return 0, 0, invalidQuery("limit must be between 1 and " + strconv.Itoa(maxAdminPage))
	}
	offset, err := parseIntQuery(r, "offset", 0)
	if err != nil {
		return 0, 0, err
	}
	if offset < 0 {
		return 0, 0, invalidQuery("offset must be >= 0")
	}
	return int32(limit), int32(offset), nil
}
