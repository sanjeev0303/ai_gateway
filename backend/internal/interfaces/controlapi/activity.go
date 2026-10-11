package controlapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ai-governance-saas/backend/internal/domain"
)

func registerActivityRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/activity", a.requireAuth(a.handleActivity))
	mux.Handle("GET /v1/activity/active-users", a.requireAuth(a.handleActivityActiveUsers))
	mux.Handle("GET /v1/activity/active-requests", a.requireAuth(a.handleActivityActiveRequests))
	mux.Handle("GET /v1/activity/stream", a.requireAuth(a.handleStream))
}

func (a *api) handleActivity(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	events, err := a.activity.ActiveRequests(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	if events == nil {
		events = []domain.ActivityEvent{}
	}

	// If no in-flight streaming requests in Redis, include recent completed requests
	// so the activity feed displays recent gateway requests seamlessly.
	if len(events) == 0 && a.usage != nil {
		if recents, err := a.usage.GetRecent(r.Context(), actor); err == nil {
			for _, req := range recents {
				state := domain.ActivityStateCompleted
				if req.Status == "failed" {
					state = domain.ActivityStateFailed
				}
				events = append(events, domain.ActivityEvent{
					RequestID:      req.RequestID,
					OrganizationID: req.OrganizationID,
					UserID:         req.UserID,
					TeamID:         req.TeamID,
					ProjectID:      req.ProjectID,
					Provider:       req.ProviderKind,
					Model:          req.Model,
					Status:         req.Status,
					StartedAt:      req.CreatedAt,
					CurrentState:   state,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"data": events})
}

func (a *api) handleActivityActiveUsers(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	events, err := a.activity.ActiveRequests(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	users := make(map[string]bool)
	for _, ev := range events {
		if ev.UserID != nil {
			users[ev.UserID.String()] = true
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"count": len(users)})
}

func (a *api) handleActivityActiveRequests(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	events, err := a.activity.ActiveRequests(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"count": len(events)})
}

func (a *api) handleStream(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	events, err := a.activity.Subscribe(r.Context(), actor.OrganizationID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial ping to establish connection
	fmt.Fprintf(w, "event: ping\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			data, _ := json.Marshal(ev)
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}
