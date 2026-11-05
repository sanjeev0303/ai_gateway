package gatewayv2

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// conversationNamespace seeds the ids of conversations the gateway derives
// itself, so the same credential and opening message map to the same id.
var conversationNamespace = uuid.MustParse("6f1c2a54-9d3e-4c1b-8a57-1b7e2d9c0f31")

const (
	maxRecordedPrompt  = 64 << 10
	conversationTitle  = 80
	recordTurnDeadline = 5 * time.Second
)

// promptTurn returns the first and the latest user message of an OpenAI,
// Anthropic or Gemini request body. The first identifies the conversation; the
// latest is the new turn.
func promptTurn(body []byte) (first, latest string) {
	var req struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}
	if json.Unmarshal(body, &req) != nil {
		return "", ""
	}
	var users []string
	for _, m := range req.Messages {
		if m.Role == "user" {
			users = append(users, contentText(m.Content))
		}
	}
	for _, c := range req.Contents {
		if c.Role == "" || c.Role == "user" {
			var b strings.Builder
			for _, p := range c.Parts {
				b.WriteString(p.Text)
			}
			users = append(users, b.String())
		}
	}
	if len(users) == 0 {
		return "", ""
	}
	return users[0], users[len(users)-1]
}

// conversationIDFor picks the conversation a request belongs to: the id the
// client sent (body or x-conversation-id header), else one derived from the
// credential and the opening user message, so a client that resends its
// history keeps landing in the same conversation.
func conversationIDFor(r *http.Request, hints InspectorHints, credentialID uuid.UUID, firstUser string) uuid.UUID {
	if hints.ConversationID != nil {
		return *hints.ConversationID
	}
	if id, err := uuid.Parse(strings.TrimSpace(r.Header.Get("x-conversation-id"))); err == nil {
		return id
	}
	return uuid.NewSHA1(conversationNamespace, []byte(credentialID.String()+"\x00"+firstUser))
}

// truncateRunes cuts s to at most n runes without splitting a character.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}

// recordConversation stores one successful turn's metadata, then keeps its
// text in memory under the new message ids. It runs after the response has
// been sent, on its own short deadline, and a failure is only logged: the
// request has already succeeded and must not be affected.
func (g *Gateway) recordConversation(turn domain.GatewayConversationTurn, orgID uuid.UUID) {
	if g.conversations == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), recordTurnDeadline)
	defer cancel()
	userMsg, assistantMsg, err := g.conversations.RecordTurn(ctx, orgID, turn)
	if err != nil {
		g.logger.Warn("conversation recording failed", "conversation_id", turn.ConversationID, "error", err)
		return
	}
	if userMsg != uuid.Nil {
		g.transcripts.put(orgID, turn.ConversationID, userMsg, turn.Title, turn.UserText)
	}
	if assistantMsg != uuid.Nil {
		g.transcripts.put(orgID, turn.ConversationID, assistantMsg, turn.Title, turn.AssistantText)
	}
}
