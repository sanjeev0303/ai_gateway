package domain

import (
	"time"

	"github.com/google/uuid"
)

// PrivacyMode controls how much conversation content the read API may return.
// The server resolves the effective mode and strips content before the value
// reaches a handler, so a caller cannot widen it: there is no request
// parameter that selects a mode.
type PrivacyMode string

const (
	// PrivacyOff returns content with no redaction.
	PrivacyOff PrivacyMode = "OFF"
	// PrivacyMetadataOnly returns no message content at all.
	PrivacyMetadataOnly PrivacyMode = "METADATA_ONLY"
	// PrivacyPromptOnly returns the prompt side (system and user turns) and
	// withholds assistant and tool output.
	PrivacyPromptOnly PrivacyMode = "PROMPT_ONLY"
	// PrivacyFullContent returns every turn verbatim.
	PrivacyFullContent PrivacyMode = "FULL_CONTENT"
)

// DefaultPrivacyMode fails closed: an unset or unrecognised deployment
// withholds content rather than exposing it.
const DefaultPrivacyMode = PrivacyMetadataOnly

// Valid reports whether m is one of the four supported modes.
func (m PrivacyMode) Valid() bool {
	switch m {
	case PrivacyOff, PrivacyMetadataOnly, PrivacyPromptOnly, PrivacyFullContent:
		return true
	default:
		return false
	}
}

// ParsePrivacyMode converts s to a PrivacyMode. An unrecognised value is
// rejected rather than silently downgraded, so a typo cannot quietly widen
// access.
func ParsePrivacyMode(s string) (PrivacyMode, bool) {
	m := PrivacyMode(s)
	return m, m.Valid()
}

// ResolvePrivacyMode returns the effective mode for a conversation. A
// per-conversation override wins; nil means "inherit the server default".
func ResolvePrivacyMode(override *string, serverDefault PrivacyMode) PrivacyMode {
	if override == nil {
		if serverDefault.Valid() {
			return serverDefault
		}
		return DefaultPrivacyMode
	}
	m, ok := ParsePrivacyMode(*override)
	if !ok {
		return DefaultPrivacyMode
	}
	return m
}

// ContentVisible reports whether a turn's content may be returned under m. An
// unrecognised mode is treated as METADATA_ONLY so a new or corrupted value can
// only ever withhold content.
func (m PrivacyMode) ContentVisible(role string) bool {
	switch m {
	case PrivacyOff, PrivacyFullContent:
		return true
	case PrivacyPromptOnly:
		return role == "system" || role == "user"
	default:
		return false
	}
}

// Conversation is a user's persisted chat session.
//
// ProjectID, Model, Provider and PrivacyState are nullable because the write
// path does not populate them yet; nil means "not recorded", never a zero
// value. PrivacyState is an override, not the effective mode — resolve it with
// ResolvePrivacyMode.
type Conversation struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	ProjectID      *uuid.UUID
	Title          string
	Status         string
	Model          *string
	Provider       *string
	PrivacyState   *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ConversationMessage is one turn within a conversation.
type ConversationMessage struct {
	ID                uuid.UUID
	ConversationID    uuid.UUID
	OrganizationID    uuid.UUID
	UserID            uuid.UUID
	Role              string
	Content           string
	Tokens            int32
	ToolName          *string
	ProviderMessageID *string
	CreatedAt         time.Time
}

// Redacted returns a copy whose Content is blanked unless mode permits the
// turn's role. This is the single chokepoint every content-returning path must
// pass through; the handler omits the blanked field from the response entirely
// rather than sending an empty string.
func (m ConversationMessage) Redacted(mode PrivacyMode) ConversationMessage {
	if !mode.ContentVisible(m.Role) {
		m.Content = ""
	}
	return m
}

// ConversationSummary is one row of the conversation list: identity plus the
// aggregates derived from its messages.
type ConversationSummary struct {
	ID        uuid.UUID
	Title     string
	UserID    uuid.UUID
	ProjectID *uuid.UUID
	Model     *string
	Provider  *string
	StartedAt time.Time
	// LastActivityAt is the newest message timestamp, or nil when the
	// conversation has no messages.
	LastActivityAt *time.Time
	// RequestCount counts assistant turns. llm_requests has no conversation
	// key, so an assistant turn is the only evidence of a provider request.
	RequestCount int32
	MessageCount int32
	TotalTokens  int32
	// Raw override: the repository cannot know the server default, so the
	// service resolves it.
	PrivacyState *string
}

// ConversationFilter narrows a conversation listing. Nil members are ignored,
// so it composes with any subset of the filters.
type ConversationFilter struct {
	ID        *uuid.UUID
	From      time.Time
	To        time.Time
	UserID    *uuid.UUID
	TeamID    *uuid.UUID
	ProjectID *uuid.UUID
	Provider  *string
	Model     *string
}

// NewConversation carries the caller-supplied fields for creating a
// conversation. Status is defaulted by the database to 'active'.
type NewConversation struct {
	UserID uuid.UUID
	Title  string
}

// NewConversationMessage carries the caller-supplied fields for appending a
// message to a conversation.
type NewConversationMessage struct {
	ConversationID    uuid.UUID
	UserID            uuid.UUID
	Role              string
	Content           string
	Tokens            int32
	ToolName          *string
	ProviderMessageID *string
}
