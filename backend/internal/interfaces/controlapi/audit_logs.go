package controlapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// maxAuditLogPage caps limit so one request cannot ask for the whole ledger.
const maxAuditLogPage = 200

func registerAuditLogRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/audit-logs", a.requireAuth(a.listAuditLogs))
	mux.Handle("GET /v1/audit-logs/{id}", a.requireAuth(a.getAuditLog))
	mux.Handle("GET /v1/audit/logs", a.requireAuth(a.listAuditLogs))
}

// auditLogDTO is the wire shape of one immutable audit record.
type auditLogDTO struct {
	ID                    string          `json:"id"`
	ActorType             string          `json:"actor_type"`
	ActorUserID           *string         `json:"actor_user_id"`
	ActorServiceAccountID *string         `json:"actor_service_account_id"`
	Action                string          `json:"action"`
	ResourceType          string          `json:"resource_type"`
	ResourceID            *string         `json:"resource_id"`
	Before                json.RawMessage `json:"before"`
	After                 json.RawMessage `json:"after"`
	IPAddress             *string         `json:"ip_address"`
	RequestID             *string         `json:"request_id"`
	CreatedAt             string          `json:"created_at"`
}

func (a *api) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	filter, err := parseAuditLogFilter(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	limit, offset, err := parseAuditLogPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	entries, hasMore, err := a.auditLogs.List(r.Context(), actor, filter, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]auditLogDTO, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toAuditLogDTO(entry))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items, HasMore: &hasMore})
}

func (a *api) getAuditLog(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	entry, err := a.auditLogs.Get(r.Context(), actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAuditLogDTO(entry))
}

func toAuditLogDTO(entry domain.AuditLog) auditLogDTO {
	dto := auditLogDTO{
		ID:                    entry.ID.String(),
		ActorUserID:           uuidPtrString(entry.ActorUserID),
		ActorServiceAccountID: uuidPtrString(entry.ActorServiceAccountID),
		Action:                entry.Action,
		ResourceType:          entry.ResourceType,
		ResourceID:            uuidPtrString(entry.ResourceID),
		Before:                jsonObjectOrEmpty(entry.Before),
		After:                 jsonObjectOrEmpty(entry.After),
		IPAddress:             entry.IPAddress,
		RequestID:             entry.RequestID,
		CreatedAt:             entry.CreatedAt.UTC().Format(time.RFC3339),
	}
	switch {
	case entry.ActorUserID != nil:
		dto.ActorType = "user"
	case entry.ActorServiceAccountID != nil:
		dto.ActorType = "service_account"
	}
	return dto
}

// jsonObjectOrEmpty falls back to an empty object so one unparseable JSON
// column cannot fail the whole response.
func jsonObjectOrEmpty(raw []byte) json.RawMessage {
	if !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func parseAuditLogFilter(r *http.Request) (domain.AuditLogFilter, error) {
	var filter domain.AuditLogFilter
	var err error

	if filter.ActorID, err = parseOptionalUUIDParam(r, "actor_id"); err != nil {
		return filter, err
	}
	if filter.ResourceID, err = parseOptionalUUIDParam(r, "resource_id"); err != nil {
		return filter, err
	}
	if filter.ActorType, err = parseAuditActorType(r); err != nil {
		return filter, err
	}
	filter.Action = parseOptionalString(r, "action")
	filter.ResourceType = parseOptionalString(r, "resource_type")
	if filter.From, err = parseOptionalTimeQuery(r, "from"); err != nil {
		return filter, err
	}
	if filter.To, err = parseOptionalTimeQuery(r, "to"); err != nil {
		return filter, err
	}
	return filter, nil
}

func parseAuditActorType(r *http.Request) (*string, error) {
	val := r.URL.Query().Get("actor_type")
	if val == "" {
		return nil, nil
	}
	if val != "user" && val != "service_account" {
		return nil, invalidQuery("actor_type must be user or service_account")
	}
	return &val, nil
}

func parseAuditLogPage(r *http.Request) (int32, int32, error) {
	limit, err := parseIntQuery(r, "limit", 50)
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > maxAuditLogPage {
		return 0, 0, invalidQuery("limit must be between 1 and " + strconv.Itoa(maxAuditLogPage))
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
