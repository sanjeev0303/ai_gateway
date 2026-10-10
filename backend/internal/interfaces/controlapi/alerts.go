package controlapi

import (
	"encoding/json"
	"net/http"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

func registerAlertRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/alerts", a.requireAuth(a.listAlerts))
	mux.Handle("POST /v1/alerts", a.requireAuth(a.createAlert))
	mux.Handle("GET /v1/alerts/{id}", a.requireAuth(a.getAlert))
	mux.Handle("PUT /v1/alerts/{id}", a.requireAuth(a.updateAlert))
	mux.Handle("DELETE /v1/alerts/{id}", a.requireAuth(a.deleteAlert))
	mux.Handle("POST /v1/alerts/{id}/test", a.requireAuth(a.testAlert))
}

// alertDTO is the wire shape of one configured alert rule.
type alertDTO struct {
	ID              string          `json:"id"`
	AlertType       string          `json:"alert_type"`
	Name            string          `json:"name"`
	Severity        string          `json:"severity"`
	Configuration   json.RawMessage `json:"configuration"`
	Status          string          `json:"status"`
	LastTriggeredAt *string         `json:"last_triggered_at"`
	CreatedAt       string          `json:"created_at"`
	UpdatedAt       string          `json:"updated_at"`
}

type updateAlertRequest struct {
	Name          string          `json:"name"`
	Severity      string          `json:"severity"`
	Configuration json.RawMessage `json:"configuration"`
	Status        string          `json:"status"`
}

type createAlertRequest struct {
	AlertType     string          `json:"alert_type"`
	Name          string          `json:"name"`
	Severity      string          `json:"severity"`
	Configuration json.RawMessage `json:"configuration"`
}

func (a *api) listAlerts(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	alerts, err := a.alerts.List(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]alertDTO, 0, len(alerts))
	for _, entry := range alerts {
		items = append(items, toAlertDTO(entry))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *api) getAlert(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	entry, err := a.alerts.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAlertDTO(entry))
}

func (a *api) createAlert(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	var req createAlertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	
	entry, err := a.alerts.Create(r.Context(), actor, domain.NewAlert{
		AlertType:     req.AlertType,
		Name:          req.Name,
		Severity:      req.Severity,
		Configuration: req.Configuration,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toAlertDTO(entry))
}

func (a *api) updateAlert(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req updateAlertRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	entry, err := a.alerts.Update(r.Context(), actor, id, domain.AlertUpdate{
		Name:          req.Name,
		Severity:      req.Severity,
		Configuration: req.Configuration,
		Status:        req.Status,
	})
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAlertDTO(entry))
}

func (a *api) deleteAlert(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if err := a.alerts.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) testAlert(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	result, err := a.alerts.Test(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"alert_id":          result.AlertID.String(),
		"alert_type":        result.AlertType,
		"severity":          result.Severity,
		"status":            result.Status,
		"configuration":     result.Configuration,
		"status_is_muted":   result.StatusIsMuted,
		"notification_sent": result.NotificationSent,
		"message":           result.Message,
	})
}

func toAlertDTO(entry domain.Alert) alertDTO {
	dto := alertDTO{
		ID:            entry.ID.String(),
		AlertType:     entry.AlertType,
		Name:          entry.Name,
		Severity:      entry.Severity,
		Configuration: jsonObjectOrEmpty(entry.Configuration),
		Status:        entry.Status,
		CreatedAt:     entry.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     entry.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if entry.LastTriggeredAt != nil {
		triggered := entry.LastTriggeredAt.UTC().Format(time.RFC3339)
		dto.LastTriggeredAt = &triggered
	}
	return dto
}
