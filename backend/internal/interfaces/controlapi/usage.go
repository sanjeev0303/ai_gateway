package controlapi

import (
	"errors"
	"net/http"
	"time"

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
	mux.Handle("GET /v1/usage/by-team", a.requireAuth(a.getUsageByTeam))
	mux.Handle("GET /v1/usage/recent", a.requireAuth(a.getUsageRecent))
	mux.Handle("GET /v1/usage/requests", a.requireAuth(a.getUsageRequests))
}

func (a *api) getUsageSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	projectID, err := parseOptionalUUIDParam(r, "project_id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	model := parseOptionalString(r, "model")

	summary, err := a.usage.GetSummary(ctx, actor, from, to, projectID, model)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (a *api) getUsageTimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-7*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	projectID, err := parseOptionalUUIDParam(r, "project_id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	points, err := a.usage.GetTimeSeries(ctx, actor, from, to, projectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": points})
}

func (a *api) getUsageByProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-30*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByProject(ctx, actor, from, to)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-30*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByModel(ctx, actor, from, to)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByProvider(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-30*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByProvider(ctx, actor, from, to)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-30*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByUser(ctx, actor, from, to)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func (a *api) getUsageByTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	from, err := parseTimeQuery(r, "from", time.Now().Add(-30*24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	results, err := a.usage.GetByTeam(ctx, actor, from, to)
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

	from, err := parseTimeQuery(r, "from", time.Now().Add(-24*time.Hour))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	to, err := parseTimeQuery(r, "to", time.Now())
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	
	limit := int32(50)
	offset := int32(0)

	results, err := a.usage.GetRequests(ctx, actor, from, to, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": results})
}

func parseTimeQuery(r *http.Request, key string, def time.Time) (time.Time, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return def, nil
	}
	return time.Parse(time.RFC3339, val)
}

func parseOptionalUUIDParam(r *http.Request, key string) (*uuid.UUID, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil, nil
	}
	id, err := uuid.Parse(val)
	if err != nil {
		return nil, errors.New("invalid uuid format for " + key)
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
