package controlapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// The admin console's monitoring, audit and earnings views.

type adminSampleDTO struct {
	Service    string   `json:"service"`
	SampledAt  string   `json:"sampled_at"`
	Up         bool     `json:"up"`
	Error      string   `json:"error,omitempty"`
	CPUPercent *float64 `json:"cpu_percent,omitempty"`
	RSSBytes   *int64   `json:"rss_bytes,omitempty"`
	HeapBytes  *int64   `json:"heap_bytes,omitempty"`
	Goroutines *int     `json:"goroutines,omitempty"`
	Detail     any      `json:"detail"`
}

func toAdminSampleDTO(s domain.ServiceSample) adminSampleDTO {
	return adminSampleDTO{
		Service:    s.Service,
		SampledAt:  ts(s.SampledAt),
		Up:         s.Up,
		Error:      s.Error,
		CPUPercent: s.CPUPercent,
		RSSBytes:   s.RSSBytes,
		HeapBytes:  s.HeapBytes,
		Goroutines: s.Goroutines,
		Detail:     jsonObjectOrEmpty(s.Detail),
	}
}

type adminAlertDTO struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	Service        string `json:"service"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Detail         string `json:"detail"`
	Status         string `json:"status"`
	OpenedAt       string `json:"opened_at"`
	LastSeenAt     string `json:"last_seen_at"`
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
	Emailed        bool   `json:"emailed"`
}

func (a *api) adminServices(w http.ResponseWriter, r *http.Request, _ domain.User) {
	view, err := a.platformAdmin.Services(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	latest := make([]adminSampleDTO, 0, len(view.Latest))
	for _, s := range view.Latest {
		latest = append(latest, toAdminSampleDTO(s))
	}
	series := make([]adminSampleDTO, 0, len(view.Series))
	for _, s := range view.Series {
		series = append(series, toAdminSampleDTO(s))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"latest": latest, "series": series})
}

func (a *api) adminAlerts(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	alerts, total, err := a.platformAdmin.Alerts(r.Context(), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]adminAlertDTO, 0, len(alerts))
	for _, al := range alerts {
		items = append(items, adminAlertDTO{
			ID:             al.ID.String(),
			Kind:           al.Kind,
			Service:        al.Service,
			Severity:       al.Severity,
			Title:          al.Title,
			Detail:         al.Detail,
			Status:         al.Status,
			OpenedAt:       ts(al.OpenedAt),
			LastSeenAt:     ts(al.LastSeenAt),
			AcknowledgedAt: tsPtr(al.AcknowledgedAt),
			AcknowledgedBy: al.AcknowledgedBy,
			ResolvedAt:     tsPtr(al.ResolvedAt),
			Emailed:        al.NotifiedAt != nil,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminAcknowledgeAlert(w http.ResponseWriter, r *http.Request, admin domain.User) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.platformAdmin.AcknowledgeAlert(r.Context(), admin, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) adminLogs(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	q := r.URL.Query()
	logs, total, err := a.platformAdmin.Logs(r.Context(), domain.ServiceLogFilter{
		Service: q.Get("service"),
		Level:   q.Get("level"),
		Search:  q.Get("search"),
	}, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		items = append(items, map[string]any{
			"id":        l.ID,
			"service":   l.Service,
			"level":     l.Level,
			"message":   l.Message,
			"attrs":     jsonObjectOrEmpty(l.Attrs),
			"logged_at": l.LoggedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminTenantAudit(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	orgID, err := optionalUUIDQuery(r, "organization_id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	entries, total, err := a.platformAdmin.TenantAudit(r.Context(), orgID, r.URL.Query().Get("action"), limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		item := map[string]any{
			"id":                e.ID.String(),
			"organization_id":   e.OrganizationID.String(),
			"organization_name": e.OrganizationName,
			"actor_email":       e.ActorEmail,
			"action":            e.Action,
			"resource_type":     e.ResourceType,
			"after":             jsonObjectOrEmpty(e.After),
			"created_at":        ts(e.CreatedAt),
		}
		if e.ResourceID != nil {
			item["resource_id"] = e.ResourceID.String()
		}
		if e.IPAddress != nil {
			item["ip_address"] = *e.IPAddress
		}
		items = append(items, item)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminSecurityEvents(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	events, total, err := a.platformAdmin.SecurityEvents(r.Context(), r.URL.Query().Get("severity"), limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		items = append(items, map[string]any{
			"id":                e.ID.String(),
			"organization_id":   e.OrganizationID.String(),
			"organization_name": e.OrganizationName,
			"category":          e.Category,
			"severity":          e.Severity,
			"title":             e.Title,
			"detail":            jsonObjectOrEmpty(e.Detail),
			"source":            e.Source,
			"created_at":        ts(e.CreatedAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

func (a *api) adminRevenue(w http.ResponseWriter, r *http.Request, _ domain.User) {
	rev, err := a.platformAdmin.Revenue(r.Context())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	monthly := make([]map[string]any, 0, len(rev.Monthly))
	for _, m := range rev.Monthly {
		monthly = append(monthly, map[string]any{
			"month":          m.Month.Format("2006-01"),
			"revenue_micros": m.RevenueMicros,
			"payments":       m.Payments,
		})
	}
	customers := make([]map[string]any, 0, len(rev.Customers))
	for _, c := range rev.Customers {
		customers = append(customers, map[string]any{
			"organization_id":   c.OrganizationID.String(),
			"organization_name": c.OrganizationName,
			"plan_name":         c.PlanName,
			"lifetime_micros":   c.LifetimeMicros,
			"payments":          c.Payments,
			"last_paid_at":      ts(c.LastPaidAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]any{
			"lifetime_micros":   rev.Summary.LifetimeMicros,
			"last_30d_micros":   rev.Summary.Last30DMicros,
			"this_month_micros": rev.Summary.ThisMonthMicros,
			"payments":          rev.Summary.Payments,
			"paying_orgs":       rev.Summary.PayingOrgs,
		},
		"monthly":   monthly,
		"customers": customers,
	})
}

func (a *api) adminPayments(w http.ResponseWriter, r *http.Request, _ domain.User) {
	limit, offset, err := parseAdminPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	orgID, err := optionalUUIDQuery(r, "organization_id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	payments, total, err := a.platformAdmin.Payments(r.Context(), orgID, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(payments))
	for _, p := range payments {
		items = append(items, map[string]any{
			"id":                p.ID.String(),
			"organization_id":   p.OrganizationID.String(),
			"organization_name": p.OrganizationName,
			"number":            p.Number,
			"total_micros":      p.TotalMicros,
			"currency":          p.Currency,
			"provider":          p.Provider,
			"payment_id":        p.PaymentID,
			"paid_at":           ts(p.PaidAt),
		})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}

// optionalUUIDQuery reads a UUID query parameter; absent means nil.
func optionalUUIDQuery(r *http.Request, key string) (*uuid.UUID, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, invalidQuery(key + " must be a UUID")
	}
	return &id, nil
}
