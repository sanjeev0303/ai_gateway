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
	Log           *slog.Logger
}

// Service reads conversations for authenticated actors and applies privacy
// redaction on the way out. Callers cannot influence the resolved mode.
type Service struct {
	organizations repositories.OrganizationRepository
	conversations repositories.ConversationRepository
	messages      repositories.ConversationMessageRepository
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
	return summary, nil
}

// ListMessages reads one conversation's turns, redacted to the mode that
// resolves for that conversation. The conversation is loaded first because
// its privacy state is what governs its messages.
func (s *Service) ListMessages(ctx context.Context, actor domain.Actor, id uuid.UUID) ([]domain.ConversationMessage, error) {
	if err := s.require(actor); err != nil {
		return nil, err
	}
	conversation, err := s.conversations.Get(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	// Subject access control: non-privileged members can only access their own conversations
	if !isPrivilegedRole(actor.Role) && conversation.UserID != actor.UserID {
		return nil, repositories.ErrForbidden
	}

	org, err := s.organizations.GetByID(ctx, actor.OrganizationID)
	if err != nil {
		return nil, err
	}
	defaultPrivacy := org.PrivacyDefault

	mode := s.resolve(conversation.PrivacyState, defaultPrivacy)
	messages, err := s.messages.List(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		messages[i] = messages[i].Redacted(mode)
	}
	return messages, nil
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

func (s *Service) CreateMessage(ctx context.Context, actor domain.Actor, convID uuid.UUID, role, content string, tokens int32) (domain.ConversationMessage, error) {
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

	org, err := s.organizations.GetByID(ctx, actor.OrganizationID)
	if err != nil {
		return domain.ConversationMessage{}, err
	}
	mode := s.resolve(conv.PrivacyState, org.PrivacyDefault)
	if !mode.ContentVisible(role) {
		content = ""
	}

	return s.messages.Create(ctx, actor.OrganizationID, domain.NewConversationMessage{
		ConversationID: convID,
		UserID:         actor.UserID,
		Role:           role,
		Content:        content,
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
