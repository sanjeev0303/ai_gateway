package controlapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/platform/httpx"
)

// maxConversationPage caps limit_count so one request cannot ask for the whole
// table.
const maxConversationPage = 200

func registerConversationRoutes(mux *http.ServeMux, deps Deps) {
	a := newAPI(deps)
	mux.Handle("GET /v1/conversations", a.requireAuth(a.listConversations))
	mux.Handle("POST /v1/conversations", a.requireAuth(a.createConversation))
	mux.Handle("GET /v1/conversations/{id}", a.requireAuth(a.getConversation))
	mux.Handle("GET /v1/conversations/{id}/messages", a.requireAuth(a.listConversationMessages))
	mux.Handle("POST /v1/conversations/{id}/messages", a.requireAuth(a.createConversationMessage))
}

// conversationSummaryDTO is the wire shape of one conversation row. Cost is
// always null: conversation_messages carries no link to llm_requests, so
// per-conversation spend cannot be derived yet.
type conversationSummaryDTO struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	UserID         string  `json:"user_id"`
	ProjectID      *string `json:"project_id"`
	Provider       *string `json:"provider"`
	Model          *string `json:"model"`
	StartedAt      string  `json:"started_at"`
	LastActivityAt *string `json:"last_activity_at"`
	RequestCount   int32   `json:"request_count"`
	MessageCount   int32   `json:"message_count"`
	TotalTokens    int32   `json:"total_tokens"`
	CostMicros     *int64  `json:"cost_micros"`
	PrivacyState   string  `json:"privacy_state"`
}

// conversationMessageDTO omits content when privacy withheld it, so a
// redacted turn carries no content key at all rather than an empty string.
type conversationMessageDTO struct {
	ID                string  `json:"id"`
	ConversationID    string  `json:"conversation_id"`
	Role              string  `json:"role"`
	Content           string  `json:"content,omitempty"`
	Tokens            int32   `json:"tokens"`
	ToolName          *string `json:"tool_name"`
	ProviderMessageID *string `json:"provider_message_id"`
	CreatedAt         string  `json:"created_at"`
}

func (a *api) listConversations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	filter, err := parseConversationFilter(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	limit, offset, err := parseConversationPage(r)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	summaries, hasMore, err := a.conversations.List(ctx, actor, filter, limit, offset)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]conversationSummaryDTO, 0, len(summaries))
	for _, s := range summaries {
		items = append(items, toConversationSummaryDTO(s))
	}
	httpx.WriteJSON(w, http.StatusOK, listDTO{Items: items, HasMore: &hasMore})
}

func (a *api) getConversation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	summary, err := a.conversations.Get(ctx, actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toConversationSummaryDTO(summary))
}

func (a *api) listConversationMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	actor, _ := actorFrom(ctx)

	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	messages, err := a.conversations.ListMessages(ctx, actor, id)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]conversationMessageDTO, 0, len(messages))
	for _, m := range messages {
		items = append(items, toConversationMessageDTO(m))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

// parseConversationFilter reads the optional list filters. It deliberately has
// no privacy parameter: the caller cannot influence how much content is
// returned.
func parseConversationFilter(r *http.Request) (domain.ConversationFilter, error) {
	var filter domain.ConversationFilter
	var err error

	if filter.From, err = parseOptionalTimeQuery(r, "from"); err != nil {
		return filter, err
	}
	if filter.To, err = parseOptionalTimeQuery(r, "to"); err != nil {
		return filter, err
	}
	if filter.UserID, err = parseOptionalUUIDParam(r, "user_id"); err != nil {
		return filter, err
	}
	if filter.TeamID, err = parseOptionalUUIDParam(r, "team_id"); err != nil {
		return filter, err
	}
	if filter.ProjectID, err = parseOptionalUUIDParam(r, "project_id"); err != nil {
		return filter, err
	}
	filter.Provider = parseOptionalString(r, "provider")
	filter.Model = parseOptionalString(r, "model")
	return filter, nil
}

func parseConversationPage(r *http.Request) (int32, int32, error) {
	limit, err := parseIntQuery(r, "limit", 50)
	if err != nil {
		return 0, 0, err
	}
	if limit < 1 || limit > maxConversationPage {
		return 0, 0, invalidQuery("limit must be between 1 and " + strconv.Itoa(maxConversationPage))
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

// parseOptionalTimeQuery leaves the zero time when absent, which the
// repository maps to SQL NULL.
func parseOptionalTimeQuery(r *http.Request, key string) (time.Time, error) {
	return parseTimeQuery(r, key, time.Time{})
}

func toConversationSummaryDTO(s domain.ConversationSummary) conversationSummaryDTO {
	dto := conversationSummaryDTO{
		ID:           s.ID.String(),
		Title:        s.Title,
		UserID:       s.UserID.String(),
		ProjectID:    uuidPtrString(s.ProjectID),
		Provider:     s.Provider,
		Model:        s.Model,
		StartedAt:    s.StartedAt.UTC().Format(time.RFC3339),
		RequestCount: s.RequestCount,
		MessageCount: s.MessageCount,
		TotalTokens:  s.TotalTokens,
	}
	if s.LastActivityAt != nil {
		last := s.LastActivityAt.UTC().Format(time.RFC3339)
		dto.LastActivityAt = &last
	}
	if s.PrivacyState != nil {
		dto.PrivacyState = *s.PrivacyState
	}
	return dto
}

func toConversationMessageDTO(m domain.ConversationMessage) conversationMessageDTO {
	return conversationMessageDTO{
		ID:                m.ID.String(),
		ConversationID:    m.ConversationID.String(),
		Role:              m.Role,
		Content:           m.Content,
		Tokens:            m.Tokens,
		ToolName:          m.ToolName,
		ProviderMessageID: m.ProviderMessageID,
		CreatedAt:         m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

type createConversationRequest struct {
	Title string `json:"title"`
}

func (a *api) createConversation(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	var req createConversationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	conv, err := a.conversations.Create(r.Context(), actor, req.Title)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":         conv.ID.String(),
		"title":      conv.Title,
		"user_id":    conv.UserID.String(),
		"created_at": conv.CreatedAt.Format(time.RFC3339),
	})
}

type createMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Tokens  int32  `json:"tokens,omitempty"`
}

func (a *api) createConversationMessage(w http.ResponseWriter, r *http.Request) {
	actor, _ := actorFrom(r.Context())
	id, err := pathUUID(r, "id")
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	var req createMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeServiceError(w, r, err)
		return
	}
	msg, err := a.conversations.CreateMessage(r.Context(), actor, id, req.Role, req.Content, req.Tokens)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"id":              msg.ID.String(),
		"conversation_id": msg.ConversationID.String(),
		"role":            msg.Role,
		"content":         msg.Content,
		"created_at":      msg.CreatedAt.Format(time.RFC3339),
	})
}

