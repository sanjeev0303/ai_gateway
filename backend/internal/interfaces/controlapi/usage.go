package controlapi

import (
	"net/http"
	"strconv"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
	"github.com/google/uuid"
)

func registerUsageRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/usage/summary", a.requireAuth(a.getUsageSummary))
	mux.Handle("GET /v1/usage/timeseries", a.requireAuth(a.getUsageTimeSeries))
	mux.Handle("GET /v1/usage/by-project", a.requireAuth(a.getUsageByProject))
	mux.Handle("GET /v1/usage/by-model", a.requireAuth(a.getUsageByModel))
	mux.Handle("GET /v1/usage/by-provider", a.requireAuth(a.getUsageByProvider))
	mux.Handle("GET /v1/usage/by-user", a.requireAuth(a.getUsageByUser))
	mux.Handle("GET /v1/usage/by-service-account", a.requireAuth(a.getUsageByServiceAccount))
	mux.Handle("GET /v1/usage/by-team", a.requireAuth(a.getUsageByTeam))
	mux.Handle("GET /v1/usage/latency", a.requireAuth(a.getUsageLatency))
	mux.Handle("GET /v1/usage/recent", a.requireAuth(a.getUsageRecent))
	mux.Handle("GET /v1/usage/requests", a.requireAuth(a.getUsageRequests))
}

func (a *api) getUsageSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	summary, err := a.usage.GetSummary(ctx, actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (a *api) getUsageTimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 7*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	points, err := a.usage.GetTimeSeries(ctx, actor, filter.From, filter.To, filter.ProjectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": points})
}

func (a *api) getUsageByProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByProject(ctx, actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByModel(ctx, actor, filter.From, filter.To)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByProvider(ctx, actor, filter.From, filter.To)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByUser(ctx, actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByServiceAccount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByServiceAccount(ctx, actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 30*24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByTeam(ctx, actor, filter)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

// getUsageLatency returns p50/p95/p99 timings computed by Postgres. Pass
// group_by=provider_model to break them down per provider and model.
func (a *api) getUsageLatency(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	groupBy := r.URL.Query().Get("group_by") == "provider_model"

	results, err := a.usage.GetLatency(ctx, actor, filter, groupBy)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageRecent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	results, err := a.usage.GetRecent(ctx, actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageRequests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseUsageFilter(r, 24*time.Hour)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	limit, err := parseIntQuery(r, "limit", 50)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	offset, err := parseIntQuery(r, "offset", 0)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, hasMore, err := a.usage.GetRequests(ctx, actor, filter, int32(limit), int32(offset))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: results, HasMore: &hasMore})
}

// parseUsageFilter reads the filter every usage endpoint shares. An unknown
// status is rejected rather than silently matching no rows, which would read
// as "zero usage" instead of a bad query.
func parseUsageFilter(r *http.Request, defaultWindow time.Duration) (domain.UsageFilter, error) {
	from, err := parseTimeQuery(r, "from", time.Now().Add(-defaultWindow))
	if err != nil {
		return domain.UsageFilter{}, err
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		return domain.UsageFilter{}, err
	}
	if from.After(to) {
		return domain.UsageFilter{}, invalidQuery("'from' timestamp must not be after 'to' timestamp")
	}
	projectID, err := parseOptionalUUIDParam(r, "project_id")
	if err != nil {
		return domain.UsageFilter{}, err
	}
	userID, err := parseOptionalUUIDParam(r, "user_id")
	if err != nil {
		return domain.UsageFilter{}, err
	}
	serviceAccountID, err := parseOptionalUUIDParam(r, "service_account_id")
	if err != nil {
		return domain.UsageFilter{}, err
	}
	teamID, err := parseOptionalUUIDParam(r, "team_id")
	if err != nil {
		return domain.UsageFilter{}, err
	}
	status := parseOptionalString(r, "status")
	if status != nil && !validUsageStatuses[*status] {
		return domain.UsageFilter{}, invalidQuery("invalid value for status")
	}
	return domain.UsageFilter{
		From:             from,
		To:               to,
		ProjectID:        projectID,
		UserID:           userID,
		ServiceAccountID: serviceAccountID,
		TeamID:           teamID,
		Provider:         parseOptionalString(r, "provider"),
		Model:            parseOptionalString(r, "model"),
		Status:           status,
	}, nil
}

// validUsageStatuses mirrors the CHECK constraint on llm_requests.status.
var validUsageStatuses = map[string]bool{
	"succeeded": true,
	"failed":    true,
	"cancelled": true,
}

func parseTimeQuery(r *http.Request, key string, def time.Time) (time.Time, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return def, nil
	}
	parsed, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return time.Time{}, invalidQuery("invalid RFC3339 timestamp for " + key)
	}
	return parsed, nil
}

func parseOptionalUUIDParam(r *http.Request, key string) (*uuid.UUID, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil, nil
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return nil, invalidQuery("invalid uuid format for " + key)
	}
	return &id, nil
}

func parseOptionalString(r *http.Request, key string) *string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}
	return &val
}

func parseIntQuery(r *http.Request, key string, def int) (int, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return def, nil
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, invalidQuery("invalid integer for " + key)
	}
	return n, nil
}
