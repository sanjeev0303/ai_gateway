package controlapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// maxSecurityEventPage caps limit so one request cannot ask for the whole feed.
const maxSecurityEventPage = 200

func registerSecurityEventRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/security-events", a.requireAuth(a.listSecurityEvents))
}

// securityEventDTO is the wire shape of one append-only platform event.
type securityEventDTO struct {
	ID        string          `json:"id"`
	Category  string          `json:"category"`
	Severity  string          `json:"severity"`
	Title     string          `json:"title"`
	Detail    json.RawMessage `json:"detail"`
	Source    string          `json:"source"`
	CreatedAt string          `json:"created_at"`
}

func (a *api) listSecurityEvents(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	filter, err := parseSecurityEventFilter(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	limit, offset, err := parseSecurityEventPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	events, err := a.securityEvents.List(r.Context(), actor, filter, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]securityEventDTO, 0, len(events))
	for _, event := range events {
		items = append(items, toSecurityEventDTO(event))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func toSecurityEventDTO(event domain.SecurityEvent) securityEventDTO {
	return securityEventDTO{
		ID:        event.ID.String(),
		Category:  string(event.Category),
		Severity:  string(event.Severity),
		Title:     event.Title,
		Detail:    jsonObjectOrEmpty(event.Detail),
		Source:    event.Source,
		CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339),
	}
}

// parseSecurityEventFilter rejects an unknown category or severity rather than
// passing it through, because a typo in a filter would otherwise look like a
// request that legitimately matched no events.
func parseSecurityEventFilter(r *http.Request) (domain.SecurityEventFilter, error) {
	var filter domain.SecurityEventFilter
	var err error

	if filter.Category, err = parseSecurityEventCategory(r); err != nil {
		return filter, err
	}
	if filter.Severity, err = parseSecurityEventSeverity(r); err != nil {
		return filter, err
	}
	if filter.From, err = parseOptionalTimeQuery(r, "from"); err != nil {
		return filter, err
	}
	if filter.To, err = parseOptionalTimeQuery(r, "to"); err != nil {
		return filter, err
	}
	return filter, nil
}

func parseSecurityEventCategory(r *http.Request) (*domain.SecurityEventCategory, error) {
	val := r.URL.Query().Get("category")
	if val == "" {
		return nil, nil
	}
	category := domain.SecurityEventCategory(val)
	if !category.Valid() {
		return nil, invalidQuery("category must be provider_incident or pipeline_failure")
	}
	return &category, nil
}

func parseSecurityEventSeverity(r *http.Request) (*domain.SecurityEventSeverity, error) {
	val := r.URL.Query().Get("severity")
	if val == "" {
		return nil, nil
	}
	severity := domain.SecurityEventSeverity(val)
	if !severity.Valid() {
		return nil, invalidQuery("severity must be info, warning, or critical")
	}
	return &severity, nil
}

func parseSecurityEventPage(r *http.Request) (int32, int32, error) {
	limit, err := parseIntQuery(r, "limit", 50)
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > maxSecurityEventPage {
		return 0, 0, invalidQuery("limit must be between 1 and " + strconv.Itoa(maxSecurityEventPage))
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
