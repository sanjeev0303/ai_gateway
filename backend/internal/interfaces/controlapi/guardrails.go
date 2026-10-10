package controlapi

import (
	"encoding/json"
	"net/http"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

func registerGuardrailRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/guardrails", a.requireAuth(a.listGuardrails))
	mux.Handle("POST /v1/guardrails", a.requireAuth(a.createGuardrail))
	mux.Handle("GET /v1/guardrails/events", a.requireAuth(a.listGuardrailEvents))
	mux.Handle("GET /v1/guardrails/{id}", a.requireAuth(a.getGuardrail))
	mux.Handle("PATCH /v1/guardrails/{id}", a.requireAuth(a.updateGuardrail))
	mux.Handle("DELETE /v1/guardrails/{id}", a.requireAuth(a.deleteGuardrail))
	mux.Handle("POST /v1/guardrails/{id}/test", a.requireAuth(a.testGuardrail))
}

type guardrailDTO struct {
	ID            string                     `json:"id"`
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	TargetType    domain.GuardrailTargetType `json:"target_type"`
	TargetID      *string                    `json:"target_id,omitempty"`
	Detector      domain.GuardrailDetector   `json:"detector"`
	Action        domain.GuardrailAction     `json:"action"`
	Configuration json.RawMessage            `json:"configuration"`
	Enabled       bool                       `json:"enabled"`
	CreatedAt     string                     `json:"created_at"`
	UpdatedAt     string                     `json:"updated_at"`
}

func toGuardrailDTO(g domain.Guardrail) guardrailDTO {
	return guardrailDTO{
		ID:            g.ID,
		Name:          g.Name,
		Description:   g.Description,
		TargetType:    g.TargetType,
		TargetID:      g.TargetID,
		Detector:      g.Detector,
		Action:        g.Action,
		Configuration: g.Configuration,
		Enabled:       g.Enabled,
		CreatedAt:     ts(g.CreatedAt),
		UpdatedAt:     ts(g.UpdatedAt),
	}
}

func (a *api) listGuardrails(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	list, err := a.guardrails.List(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	res := make([]guardrailDTO, len(list))
	for i, g := range list {
		res[i] = toGuardrailDTO(g)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": res})
}

type createGuardrailRequest struct {
	Name          string                     `json:"name"`
	Description   string                     `json:"description"`
	TargetType    domain.GuardrailTargetType `json:"target_type"`
	TargetID      *string                    `json:"target_id,omitempty"`
	Detector      domain.GuardrailDetector   `json:"detector"`
	Action        domain.GuardrailAction     `json:"action"`
	Configuration json.RawMessage            `json:"configuration"`
	Enabled       bool                       `json:"enabled"`
}

func (a *api) createGuardrail(w http.ResponseWriter, r *http.Request) {
	var req createGuardrailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	actor, _ := actorFrom(r.Context())
	
	// Normalize empty string target_id to nil for global guardrails
	targetID := req.TargetID
	if targetID != nil && *targetID == "" {
		targetID = nil
	}
	
	g := domain.Guardrail{
		Name:          req.Name,
		Description:   req.Description,
		TargetType:    req.TargetType,
		TargetID:      targetID,
		Detector:      req.Detector,
		Action:        req.Action,
		Configuration: req.Configuration,
		Enabled:       req.Enabled,
	}
	created, err := a.guardrails.Create(r.Context(), actor, g)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toGuardrailDTO(created))
}

func (a *api) getGuardrail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, _ := actorFrom(r.Context())
	g, err := a.guardrails.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGuardrailDTO(g))
}

type updateGuardrailRequest struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Action        domain.GuardrailAction `json:"action"`
	Configuration json.RawMessage        `json:"configuration"`
	Enabled       bool                   `json:"enabled"`
}

func (a *api) updateGuardrail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req updateGuardrailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	actor, _ := actorFrom(r.Context())
	g := domain.Guardrail{
		ID:            id,
		Name:          req.Name,
		Description:   req.Description,
		Action:        req.Action,
		Configuration: req.Configuration,
		Enabled:       req.Enabled,
	}
	updated, err := a.guardrails.Update(r.Context(), actor, g)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toGuardrailDTO(updated))
}

func (a *api) deleteGuardrail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, _ := actorFrom(r.Context())
	if err := a.guardrails.Delete(r.Context(), actor, id); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type guardrailEventDTO struct {
	ID              string `json:"id"`
	GuardrailID     string `json:"guardrail_id"`
	RequestID       string `json:"request_id"`
	Direction       string `json:"direction"`
	Detector        string `json:"detector"`
	Action          string `json:"action"`
	MatchedCategory string `json:"matched_category"`
	Timestamp       string `json:"timestamp"`
}

func (a *api) listGuardrailEvents(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	events, err := a.guardrails.ListEvents(r.Context(), actor)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	res := make([]guardrailEventDTO, len(events))
	for i, e := range events {
		res[i] = guardrailEventDTO{
			ID:              e.ID,
			GuardrailID:     e.GuardrailID,
			RequestID:       e.RequestID,
			Direction:       string(e.Direction),
			Detector:        string(e.Detector),
			Action:          string(e.Action),
			MatchedCategory: e.MatchedCategory,
			Timestamp:       ts(e.Timestamp),
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": res})
}

type testGuardrailRequest struct {
	Payload    string `json:"payload"`
	TokenCount int    `json:"token_count"`
}

type testGuardrailResponse struct {
	Triggered       bool   `json:"triggered"`
	Action          string `json:"action,omitempty"`
	MatchedCategory string `json:"matched_category,omitempty"`
}

func (a *api) testGuardrail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actor, _ := actorFrom(r.Context())

	var req testGuardrailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}

	g, err := a.guardrails.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	// For the control plane, if `evaluator` isn't injected, we can just use a local one.
	// But it's better to pass it in `Deps`. Assuming `a.evaluator` is available.
	if a.guardrailsEval == nil {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	result := a.guardrailsEval.Evaluate(r.Context(), req.Payload, req.TokenCount, []domain.Guardrail{g})
	if result != nil {
		httpx.WriteJSON(w, http.StatusOK, testGuardrailResponse{
			Triggered:       true,
			Action:          string(result.Action),
			MatchedCategory: result.MatchedCategory,
		})
	} else {
		httpx.WriteJSON(w, http.StatusOK, testGuardrailResponse{
			Triggered: false,
		})
	}
}
