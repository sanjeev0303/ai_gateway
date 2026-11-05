package conversation

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

// ServiceDeps carries the repositories and the server-side privacy fallback.
type ServiceDeps struct {
	Organizations repositories.OrganizationRepository
	Conversations repositories.ConversationRepository
	Messages      repositories.ConversationMessageRepository
	// Transcripts reads conversation text from gateway memory. Optional: when
	// nil, conversations show metadata only.
	Transcripts TranscriptSource
	Log         *slog.Logger
}

// TranscriptSource reads the conversation text the gateway holds in memory.
// The text is never stored: it stays until deleted or the gateway restarts.
type TranscriptSource interface {
	Transcripts(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID, withMessages bool) (map[uuid.UUID]domain.Transcript, error)
	Delete(ctx context.Context, orgID, conversationID uuid.UUID) error
}

// TranscriptState tells the console why message text is or is not shown.
type TranscriptState string

const (
	TranscriptAvailable TranscriptState = "available"
	// TranscriptExpired: the gateway no longer holds the text (it restarted).
	TranscriptExpired TranscriptState = "expired"
)

// Service reads conversations for authenticated actors and applies privacy
// redaction on the way out. Callers cannot influence the resolved mode.
type Service struct {
	organizations repositories.OrganizationRepository
	conversations repositories.ConversationRepository
	messages      repositories.ConversationMessageRepository
	transcripts   TranscriptSource
	log           *slog.Logger
}

// NewService returns a conversation read service. Log is optional.
func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Service{
		organizations: deps.Organizations,
		conversations: deps.Conversations,
		messages:      deps.Messages,
		transcripts:   deps.Transcripts,
		log:           deps.Log,
	}
}

func isPrivilegedRole(role string) bool {
	return role == domain.RoleAdmin || role == domain.RoleOwner || role == domain.RoleAuditor
}

func (s *Service) List(ctx context.Context, actor domain.Actor, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, bool, error) {
	if err := s.require(actor); err != nil {
		return nil, false, err
	}
	// Subject access control: non-privileged members only view their own conversations
	if !isPrivilegedRole(actor.Role) {
		filter.UserID = &actor.UserID
	}
	org, err := s.organizations.GetByID(ctx, actor.OrganizationID)
	if err != nil {
		return nil, false, err
	}
	defaultPrivacy := org.PrivacyDefault

	summaries, err := s.conversations.List(ctx, actor.OrganizationID, filter, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := false
	if len(summaries) > int(limit) {
		hasMore = true
		summaries = summaries[:limit]
	}
	for i := range summaries {
		summaries[i].PrivacyState = s.effective(summaries[i].PrivacyState, defaultPrivacy)
	}
	s.fillTitles(ctx, actor.OrganizationID, summaries)
	return summaries, hasMore, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.ConversationSummary, error) {
	if err := s.require(actor); err != nil {
		return domain.ConversationSummary{}, err
	}
	summary, err := s.conversations.Detail(ctx, actor.OrganizationID, id)
	if err != nil {
		return domain.ConversationSummary{}, err
	}
	// Subject access control: non-privileged members can only access their own conversations
	if !isPrivilegedRole(actor.Role) && summary.UserID != actor.UserID {
		return domain.ConversationSummary{}, repositories.ErrForbidden
	}

	org, err := s.organizations.GetByID(ctx, actor.OrganizationID)
	if err != nil {
		return domain.ConversationSummary{}, err
	}
	defaultPrivacy := org.PrivacyDefault

	summary.PrivacyState = s.effective(summary.PrivacyState, defaultPrivacy)
	one := []domain.ConversationSummary{summary}
	s.fillTitles(ctx, actor.OrganizationID, one)
	return one[0], nil
}

// ListMessages reads one conversation's turns with their text from gateway
// memory. The text is never stored, so it is shown whenever the gateway still
// holds it; the privacy mode does not hide it.
func (s *Service) ListMessages(ctx context.Context, actor domain.Actor, id uuid.UUID) ([]domain.ConversationMessage, TranscriptState, error) {
	if err := s.require(actor); err != nil {
		return nil, "", err
	}
	conversation, err := s.conversations.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, "", err
	}
	// Subject access control: non-privileged members can only access their own conversations
	if !isPrivilegedRole(actor.Role) && conversation.UserID != actor.UserID {
		return nil, "", repositories.ErrForbidden
	}

	messages, err := s.messages.List(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, "", err
	}

	// The database holds no text; fill each turn from gateway memory.
	live := s.transcriptFor(ctx, actor.OrganizationID, []uuid.UUID{id}, true)[id]
	state := TranscriptExpired
	for i := range messages {
		if text, ok := live.Messages[messages[i].ID]; ok && messages[i].Content == "" {
			messages[i].Content = text
		}
		if messages[i].Content != "" {
			state = TranscriptAvailable
		}
	}
	return messages, state, nil
}

// fillTitles names each conversation after its opening prompt while the
// gateway still holds it.
func (s *Service) fillTitles(ctx context.Context, orgID uuid.UUID, summaries []domain.ConversationSummary) {
	if len(summaries) == 0 {
		return
	}
	ids := make([]uuid.UUID, len(summaries))
	for i, sm := range summaries {
		ids[i] = sm.ID
	}
	live := s.transcriptFor(ctx, orgID, ids, false)
	for i := range summaries {
		if t, ok := live[summaries[i].ID]; ok && t.Title != "" {
			summaries[i].Title = t.Title
		}
	}
}

// transcriptFor reads from gateway memory. A failure only means the text is
// not shown, so it is logged rather than returned.
func (s *Service) transcriptFor(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID, withMessages bool) map[uuid.UUID]domain.Transcript {
	if s.transcripts == nil {
		return nil
	}
	live, err := s.transcripts.Transcripts(ctx, orgID, ids, withMessages)
	if err != nil {
		s.log.Warn("reading transcripts from the gateway failed", "error", err)
		return nil
	}
	return live
}

// CountsByKey counts conversations per employee and gateway key. Members see
// only their own.
func (s *Service) CountsByKey(ctx context.Context, actor domain.Actor) ([]domain.ConversationKeyCount, error) {
	if err := s.require(actor); err != nil {
		return nil, err
	}
	var only *uuid.UUID
	if !isPrivilegedRole(actor.Role) {
		only = &actor.UserID
	}
	return s.conversations.CountsByKey(ctx, actor.OrganizationID, only)
}

// Delete removes a whole conversation: its text in gateway memory, then its
// record and messages. Only owners and admins may delete.
func (s *Service) Delete(ctx context.Context, actor domain.Actor, id uuid.UUID) error {
	if err := s.require(actor); err != nil {
		return err
	}
	if actor.Role != domain.RoleOwner && actor.Role != domain.RoleAdmin {
		return repositories.ErrForbidden
	}
	// Loading it first confirms it belongs to the caller's organization.
	if _, err := s.conversations.Get(ctx, actor.OrganizationID, id); err != nil {
		return err
	}
	if s.transcripts != nil {
		if err := s.transcripts.Delete(ctx, actor.OrganizationID, id); err != nil {
			// The text is unreachable once the record is gone, and lost on the
			// gateway's next restart; the deletion still goes ahead.
			s.log.Warn("deleting transcript from the gateway failed", "conversation_id", id, "error", err)
		}
	}
	return s.conversations.Delete(ctx, actor.OrganizationID, id)
}

func (s *Service) Create(ctx context.Context, actor domain.Actor, title string) (domain.Conversation, error) {
	if err := s.require(actor); err != nil {
		return domain.Conversation{}, err
	}
	if title == "" {
		title = "New Conversation"
	}
	return s.conversations.Create(ctx, actor.OrganizationID, domain.NewConversation{
		UserID: actor.UserID,
		Title:  title,
	})
}

// CreateMessage records a turn's metadata. Its text is discarded: conversation
// content is never stored (a database constraint enforces it).
func (s *Service) CreateMessage(ctx context.Context, actor domain.Actor, convID uuid.UUID, role, _ string, tokens int32) (domain.ConversationMessage, error) {
	if err := s.require(actor); err != nil {
		return domain.ConversationMessage{}, err
	}
	conv, err := s.conversations.Get(ctx, actor.OrganizationID, convID)
	if err != nil {
		return domain.ConversationMessage{}, err
	}
	if !isPrivilegedRole(actor.Role) && conv.UserID != actor.UserID {
		return domain.ConversationMessage{}, repositories.ErrForbidden
	}

	return s.messages.Create(ctx, actor.OrganizationID, domain.NewConversationMessage{
		ConversationID: convID,
		UserID:         actor.UserID,
		Role:           role,
		Tokens:         tokens,
	})
}

func (s *Service) require(actor domain.Actor) error {
	if !actor.Can(domain.PermissionUsageRead) {
		return repositories.ErrForbidden
	}
	return nil
}

// effective reports the mode actually applied rather than the raw stored
// override, which may be NULL.
func (s *Service) effective(override *string, defaultPrivacy domain.PrivacyMode) *string {
	mode := string(domain.ResolvePrivacyMode(override, defaultPrivacy))
	return &mode
}

func (s *Service) resolve(override *string, defaultPrivacy domain.PrivacyMode) domain.PrivacyMode {
	return domain.ResolvePrivacyMode(override, defaultPrivacy)
}
