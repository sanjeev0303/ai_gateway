package controlapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/application/identity"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// fakeConversationRepo returns canned rows and records the filter it was
// handed, so tests can assert what the handler parsed.
type fakeConversationRepo struct {
	summaries []domain.ConversationSummary
	detail    domain.ConversationSummary
	detailErr error

	gotFilter domain.ConversationFilter
	gotLimit  int32
	gotOffset int32
}

func (f *fakeConversationRepo) Create(context.Context, uuid.UUID, domain.NewConversation) (domain.Conversation, error) {
	return domain.Conversation{}, nil
}

func (f *fakeConversationRepo) Get(context.Context, uuid.UUID, uuid.UUID) (domain.Conversation, error) {
	return domain.Conversation{}, nil
}

func (f *fakeConversationRepo) Detail(context.Context, uuid.UUID, uuid.UUID) (domain.ConversationSummary, error) {
	if f.detailErr != nil {
		return domain.ConversationSummary{}, f.detailErr
	}
	return f.detail, nil
}

func (f *fakeConversationRepo) List(_ context.Context, _ uuid.UUID, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, error) {
	f.gotFilter, f.gotLimit, f.gotOffset = filter, limit, offset
	return f.summaries, nil
}

type fakeMessageRepo struct{ messages []domain.ConversationMessage }

func (f *fakeMessageRepo) Create(context.Context, uuid.UUID, domain.NewConversationMessage) (domain.ConversationMessage, error) {
	return domain.ConversationMessage{}, nil
}

func (f *fakeMessageRepo) List(context.Context, uuid.UUID, uuid.UUID) ([]domain.ConversationMessage, error) {
	return f.messages, nil
}

func newConversationServer(t *testing.T, st *controlStubs, conv *fakeConversationRepo, msg *fakeMessageRepo, mode domain.PrivacyMode) *httptest.Server {
	t.Helper()
	svc := identity.NewService(st.deps())
	csvc := conversation.NewService(conversation.ServiceDeps{
		Conversations:  conv,
		Messages:       msg,
		DefaultPrivacy: mode,
	})
	return buildServer(t, Deps{Identity: svc, Conversations: csvc, DevLoginEnabled: true})
}

// getConv performs an authenticated GET and returns the response with its body
// already drained.
func getConv(t *testing.T, st *controlStubs, conv *fakeConversationRepo, msg *fakeMessageRepo, mode domain.PrivacyMode, path string) (*http.Response, string) {
	t.Helper()
	srv := newConversationServer(t, st, conv, msg, mode)
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range bearerHeader(st, "member") {
		req.Header.Set(k, v)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp, string(raw)
}

func TestListConversationsParsesFilters(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{}

	getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent,
		"/v1/conversations?user_id="+testUserID.String()+
			"&project_id="+testProjID.String()+
			"&provider=anthropic&model=claude-opus-4"+
			"&from=2026-09-01T00:00:00Z&to=2026-09-30T00:00:00Z"+
			"&limit=10&offset=5")

	if conv.gotFilter.UserID == nil || *conv.gotFilter.UserID != testUserID {
		t.Error("user_id not parsed")
	}
	if conv.gotFilter.ProjectID == nil || *conv.gotFilter.ProjectID != testProjID {
		t.Error("project_id not parsed")
	}
	if conv.gotFilter.Provider == nil || *conv.gotFilter.Provider != "anthropic" {
		t.Error("provider not parsed")
	}
	if conv.gotFilter.Model == nil || *conv.gotFilter.Model != "claude-opus-4" {
		t.Error("model not parsed")
	}
	if conv.gotFilter.From.IsZero() || conv.gotFilter.To.IsZero() {
		t.Error("from/to not parsed")
	}
	if conv.gotLimit != 10 || conv.gotOffset != 5 {
		t.Errorf("pagination = (%d, %d), want (10, 5)", conv.gotLimit, conv.gotOffset)
	}
}

func TestListConversationsDefaultsPagination(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{}
	getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent, "/v1/conversations")
	if conv.gotLimit != 50 || conv.gotOffset != 0 {
		t.Errorf("default pagination = (%d, %d), want (50, 0)", conv.gotLimit, conv.gotOffset)
	}
}

// TestListConversationsHasNoPrivacyOverride proves a caller cannot widen
// access: no query parameter selects a privacy mode.
func TestListConversationsHasNoPrivacyOverride(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{summaries: []domain.ConversationSummary{
		{ID: uuid.New(), StartedAt: testNow},
	}}
	_, body := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyMetadataOnly,
		"/v1/conversations?privacy_state=FULL_CONTENT&privacy=FULL_CONTENT&mode=FULL_CONTENT")
	if !strings.Contains(body, string(domain.PrivacyMetadataOnly)) {
		t.Errorf("privacy query parameter widened access: %s", body)
	}
}

func TestConversationBadQueryParams(t *testing.T) {
	for _, tc := range []struct{ name, qs string }{
		{"limit zero", "limit=0"},
		{"limit negative", "limit=-1"},
		{"limit too large", "limit=201"},
		{"offset negative", "offset=-1"},
		{"bad uuid", "user_id=not-a-uuid"},
		{"bad timestamp", "from=yesterday"},
		{"bad limit", "limit=abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newControlStubs()
			resp, _ := getConv(t, st, &fakeConversationRepo{}, &fakeMessageRepo{},
				domain.PrivacyFullContent, "/v1/conversations?"+tc.qs)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestConversationInvalidPathUUID(t *testing.T) {
	st := newControlStubs()
	resp, _ := getConv(t, st, &fakeConversationRepo{}, &fakeMessageRepo{},
		domain.PrivacyFullContent, "/v1/conversations/not-a-uuid")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestMetadataOnlyOmitsContent is the core privacy guarantee: under the
// fail-closed default no message content may appear in the payload, and the
// content key is absent rather than blank.
func TestMetadataOnlyOmitsContent(t *testing.T) {
	st := newControlStubs()
	msg := &fakeMessageRepo{messages: []domain.ConversationMessage{
		{ID: uuid.New(), ConversationID: uuid.New(), Role: "user", Content: "secret prompt"},
		{ID: uuid.New(), ConversationID: uuid.New(), Role: "assistant", Content: "secret answer"},
	}}
	_, body := getConv(t, st, &fakeConversationRepo{}, msg, domain.PrivacyMetadataOnly,
		"/v1/conversations/"+uuid.New().String()+"/messages")

	if strings.Contains(body, "secret prompt") || strings.Contains(body, "secret answer") {
		t.Errorf("METADATA_ONLY leaked content: %s", body)
	}
	if strings.Contains(body, `"content"`) {
		t.Errorf("METADATA_ONLY included a content key: %s", body)
	}
}

func TestPromptOnlyRedactsAssistant(t *testing.T) {
	st := newControlStubs()
	msg := &fakeMessageRepo{messages: []domain.ConversationMessage{
		{ID: uuid.New(), ConversationID: uuid.New(), Role: "user", Content: "visible prompt"},
		{ID: uuid.New(), ConversationID: uuid.New(), Role: "assistant", Content: "hidden answer"},
	}}
	_, body := getConv(t, st, &fakeConversationRepo{}, msg, domain.PrivacyPromptOnly,
		"/v1/conversations/"+uuid.New().String()+"/messages")

	if !strings.Contains(body, "visible prompt") {
		t.Errorf("PROMPT_ONLY should keep user content: %s", body)
	}
	if strings.Contains(body, "hidden answer") {
		t.Errorf("PROMPT_ONLY leaked assistant content: %s", body)
	}
}

func TestFullContentReturnsContent(t *testing.T) {
	st := newControlStubs()
	msg := &fakeMessageRepo{messages: []domain.ConversationMessage{
		{ID: uuid.New(), ConversationID: uuid.New(), Role: "assistant", Content: "shown"},
	}}
	_, body := getConv(t, st, &fakeConversationRepo{}, msg, domain.PrivacyFullContent,
		"/v1/conversations/"+uuid.New().String()+"/messages")
	if !strings.Contains(body, "shown") {
		t.Errorf("FULL_CONTENT should return content: %s", body)
	}
}

// TestUnknownStoredPrivacyFailsClosed guards a corrupted DB value: it must not
// widen access.
func TestUnknownStoredPrivacyFailsClosed(t *testing.T) {
	st := newControlStubs()
	bad := "SOMETHING_ELSE"
	conv := &fakeConversationRepo{detail: domain.ConversationSummary{ID: uuid.New(), PrivacyState: &bad}}
	_, body := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent,
		"/v1/conversations/"+uuid.New().String())
	if !strings.Contains(body, string(domain.PrivacyMetadataOnly)) {
		t.Errorf("unknown privacy_state should resolve to METADATA_ONLY, got: %s", body)
	}
}

func TestSummaryReportsEffectivePrivacyState(t *testing.T) {
	st := newControlStubs()
	override := string(domain.PrivacyPromptOnly)
	conv := &fakeConversationRepo{summaries: []domain.ConversationSummary{
		{ID: uuid.New(), PrivacyState: &override, StartedAt: testNow},
	}}
	_, body := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyMetadataOnly, "/v1/conversations")
	if !strings.Contains(body, override) {
		t.Errorf("summary should report the conversation override, got: %s", body)
	}
}

// TestSummaryNullsCost documents that cost_micros is present but always null:
// conversation_messages has no link to llm_requests.
func TestSummaryNullsCost(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{summaries: []domain.ConversationSummary{
		{ID: uuid.New(), StartedAt: testNow},
	}}
	_, body := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent, "/v1/conversations")
	if !strings.Contains(body, `"cost_micros":null`) {
		t.Errorf("expected a null cost_micros, got: %s", body)
	}
}

func TestConversationDetailUsesSnakeCaseKeys(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{detail: domain.ConversationSummary{ID: uuid.New(), StartedAt: testNow}}
	_, body := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent,
		"/v1/conversations/"+uuid.New().String())

	var got map[string]any
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	for _, key := range []string{"id", "title", "user_id", "started_at", "request_count", "message_count", "total_tokens", "privacy_state", "cost_micros"} {
		if _, ok := got[key]; !ok {
			t.Errorf("response is missing key %q: %s", key, body)
		}
	}
	if _, ok := got["ID"]; ok {
		t.Errorf("response leaked a Go field name: %s", body)
	}
}

func TestConversationNotFound(t *testing.T) {
	st := newControlStubs()
	conv := &fakeConversationRepo{detailErr: repositories.ErrNotFound}
	resp, _ := getConv(t, st, conv, &fakeMessageRepo{}, domain.PrivacyFullContent,
		"/v1/conversations/"+uuid.New().String())
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestConversationRoutesRequireAuth(t *testing.T) {
	st := newControlStubs()
	srv := newConversationServer(t, st, &fakeConversationRepo{}, &fakeMessageRepo{}, domain.PrivacyFullContent)

	resp, err := srv.Client().Get(srv.URL + "/v1/conversations")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want 401", resp.StatusCode)
	}
}
