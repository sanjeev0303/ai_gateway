// Package transcripts reads the conversation text gateway-v2 keeps in memory.
package transcripts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
)

// Client calls the gateway's private transcript listener.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns a client for base (e.g. http://gateway-v2:8091).
func New(base, token string) *Client {
	return &Client{base: strings.TrimRight(base, "/"), token: token, http: &http.Client{Timeout: 3 * time.Second}}
}

// Transcripts returns the live transcripts among ids for orgID. Conversations
// the gateway no longer holds are simply absent from the result.
func (c *Client) Transcripts(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID, withMessages bool) (map[uuid.UUID]domain.Transcript, error) {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = id.String()
	}
	q := url.Values{"org": {orgID.String()}, "ids": {strings.Join(parts, ",")}}
	if withMessages {
		q.Set("messages", "1")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/transcripts?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcripts: gateway answered %s", resp.Status)
	}
	var body struct {
		Items map[uuid.UUID]domain.Transcript `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("transcripts: %w", err)
	}
	return body.Items, nil
}

// Delete forgets one conversation's text in gateway memory.
func (c *Client) Delete(ctx context.Context, orgID, conversationID uuid.UUID) error {
	u := c.base + "/transcripts/" + conversationID.String() + "?" + url.Values{"org": {orgID.String()}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("transcripts: gateway answered %s", resp.Status)
	}
	return nil
}
