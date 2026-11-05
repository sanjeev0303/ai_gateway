package gatewayv2

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

const (
	// maxTranscripts bounds memory. Text has no time limit: it stays until the
	// conversation is deleted, or until this process restarts. Past the cap the
	// least recently active conversation is dropped first.
	maxTranscripts = 10_000
	// maxTranscriptIDs bounds one read so a caller cannot ask for everything.
	maxTranscriptIDs = 200
)

type transcript struct {
	org      uuid.UUID
	title    string
	messages map[uuid.UUID]string
	last     time.Time
}

// transcriptCache holds conversation text in memory only. The database keeps
// roles, token counts and times; the words themselves are never written to
// Postgres, Redis, disk or object storage.
type transcriptCache struct {
	mu    sync.Mutex
	convs map[uuid.UUID]*transcript
	now   func() time.Time
}

func newTranscriptCache() *transcriptCache {
	return &transcriptCache{convs: make(map[uuid.UUID]*transcript), now: time.Now}
}

// put records one message's text. A conversation id seen under another
// organization is ignored, as the database upsert does.
func (c *transcriptCache) put(org, conv, msgID uuid.UUID, title, text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	t, ok := c.convs[conv]
	if ok && t.org != org {
		return
	}
	if !ok {
		c.makeRoom()
		t = &transcript{org: org, title: title, messages: make(map[uuid.UUID]string)}
		c.convs[conv] = t
	}
	t.messages[msgID] = text
	t.last = now
}

// makeRoom drops the least recently active conversation when the cache is full.
// ponytail: O(n) scan, only when full; use a heap if the cap grows by orders of
// magnitude.
func (c *transcriptCache) makeRoom() {
	if len(c.convs) < maxTranscripts {
		return
	}
	var oldest uuid.UUID
	var oldestAt time.Time
	for id, t := range c.convs {
		if oldestAt.IsZero() || t.last.Before(oldestAt) {
			oldest, oldestAt = id, t.last
		}
	}
	delete(c.convs, oldest)
}

// remove forgets a conversation's text, if it belongs to org.
func (c *transcriptCache) remove(org, conv uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.convs[conv]; ok && t.org == org {
		delete(c.convs, conv)
	}
}

// get returns copies of the live transcripts among ids that belong to org.
func (c *transcriptCache) get(org uuid.UUID, ids []uuid.UUID, withMessages bool) map[uuid.UUID]domain.Transcript {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[uuid.UUID]domain.Transcript)
	for _, id := range ids {
		t, ok := c.convs[id]
		if !ok || t.org != org {
			continue
		}
		item := domain.Transcript{Title: t.title}
		if withMessages {
			item.Messages = make(map[uuid.UUID]string, len(t.messages))
			for k, v := range t.messages {
				item.Messages[k] = v
			}
		}
		out[id] = item
	}
	return out
}

// TranscriptHandler serves the control API on a private listener:
//
//	GET    /transcripts?org=<uuid>&ids=<uuid,...>[&messages=1]
//	DELETE /transcripts/{id}?org=<uuid>
//
// Every call also needs the shared bearer token, compared in constant time.
func (g *Gateway) TranscriptHandler(token string) http.Handler {
	authorized := func(w http.ResponseWriter, r *http.Request) bool {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /transcripts/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(w, r) {
			return
		}
		org, errOrg := uuid.Parse(r.URL.Query().Get("org"))
		conv, errConv := uuid.Parse(r.PathValue("id"))
		if errOrg != nil || errConv != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		g.transcripts.remove(org, conv)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /transcripts", func(w http.ResponseWriter, r *http.Request) {
		if !authorized(w, r) {
			return
		}
		org, err := uuid.Parse(r.URL.Query().Get("org"))
		if err != nil {
			http.Error(w, "invalid org", http.StatusBadRequest)
			return
		}
		var ids []uuid.UUID
		for _, raw := range strings.Split(r.URL.Query().Get("ids"), ",") {
			if id, err := uuid.Parse(strings.TrimSpace(raw)); err == nil {
				ids = append(ids, id)
			}
		}
		if len(ids) > maxTranscriptIDs {
			http.Error(w, "too many ids", http.StatusBadRequest)
			return
		}
		items := g.transcripts.get(org, ids, r.URL.Query().Get("messages") == "1")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	})
	return mux
}
