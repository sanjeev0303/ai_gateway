package gatewayv2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/transcripts"
)

func TestTranscriptCache(t *testing.T) {
	now := time.Now()
	c := newTranscriptCache()
	c.now = func() time.Time { return now }
	org, other, conv, msg := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	c.put(org, conv, msg, "opening", "hello")
	c.put(other, conv, uuid.New(), "x", "must not land in another org's conversation")

	got := c.get(org, []uuid.UUID{conv}, true)[conv]
	if got.Title != "opening" || got.Messages[msg] != "hello" || len(got.Messages) != 1 {
		t.Fatalf("live read = %+v", got)
	}
	if len(c.get(other, []uuid.UUID{conv}, true)) != 0 {
		t.Fatal("another organization read the transcript")
	}
	if c.get(org, []uuid.UUID{conv}, false)[conv].Messages != nil {
		t.Fatal("title-only read returned messages")
	}

	// No time limit: a month later the text is still there, until it is deleted.
	now = now.Add(30 * 24 * time.Hour)
	if len(c.get(org, []uuid.UUID{conv}, true)) != 1 {
		t.Fatal("transcript expired on its own")
	}
	c.remove(other, conv)
	if len(c.get(org, []uuid.UUID{conv}, true)) != 1 {
		t.Fatal("another organization deleted the transcript")
	}
	c.remove(org, conv)
	if len(c.get(org, []uuid.UUID{conv}, true)) != 0 {
		t.Fatal("transcript survived its deletion")
	}
}

func TestTranscriptHandlerNeedsToken(t *testing.T) {
	g := &Gateway{transcripts: newTranscriptCache()}
	h := g.TranscriptHandler("secret")
	url := "/transcripts?org=" + uuid.NewString() + "&ids=" + uuid.NewString()

	for token, want := range map[string]int{"": http.StatusUnauthorized, "wrong": http.StatusUnauthorized, "secret": http.StatusOK} {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("token %q: status %d, want %d", token, rec.Code, want)
		}
	}
}

// fakeTurnStore stands in for Postgres: it hands out message ids and keeps
// nothing, as the real store keeps no text.
type fakeTurnStore struct{ user, assistant uuid.UUID }

func (f fakeTurnStore) RecordTurn(context.Context, uuid.UUID, domain.GatewayConversationTurn) (uuid.UUID, uuid.UUID, error) {
	return f.user, f.assistant, nil
}

// The whole path: a recorded turn's text is readable by the control API's
// client over the private listener, and only for its own organization.
func TestRecordedTurnReadableByControlAPI(t *testing.T) {
	store := fakeTurnStore{user: uuid.New(), assistant: uuid.New()}
	g := &Gateway{logger: discardLogger(), conversations: store, transcripts: newTranscriptCache()}
	org, conv := uuid.New(), uuid.New()
	g.recordConversation(domain.GatewayConversationTurn{
		ConversationID: conv, Title: "In one sentence, what is accrual accounting?",
		UserText: "In one sentence, what is accrual accounting?", AssistantText: "Revenue and costs are booked when earned or incurred.",
	}, org)

	srv := httptest.NewServer(g.TranscriptHandler("token"))
	defer srv.Close()

	got, err := transcripts.New(srv.URL, "token").Transcripts(context.Background(), org, []uuid.UUID{conv}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got[conv].Messages[store.assistant] != "Revenue and costs are booked when earned or incurred." || got[conv].Title == "" {
		t.Fatalf("read back %+v", got[conv])
	}
	if other, _ := transcripts.New(srv.URL, "token").Transcripts(context.Background(), uuid.New(), []uuid.UUID{conv}, true); len(other) != 0 {
		t.Fatal("another organization read the transcript")
	}
	if _, err := transcripts.New(srv.URL, "wrong").Transcripts(context.Background(), org, []uuid.UUID{conv}, true); err == nil {
		t.Fatal("a wrong token was accepted")
	}
	if err := transcripts.New(srv.URL, "wrong").Delete(context.Background(), org, conv); err == nil {
		t.Fatal("a wrong token deleted the transcript")
	}
	if err := transcripts.New(srv.URL, "token").Delete(context.Background(), org, conv); err != nil {
		t.Fatal(err)
	}
	if left, _ := transcripts.New(srv.URL, "token").Transcripts(context.Background(), org, []uuid.UUID{conv}, true); len(left) != 0 {
		t.Fatal("the transcript survived its deletion")
	}
}
