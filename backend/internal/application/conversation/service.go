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
	Conversations  repositories.ConversationRepository
	Messages       repositories.ConversationMessageRepository
	DefaultPrivacy domain.PrivacyMode
	Log            *slog.Logger
}

// Service reads conversations for authenticated actors and applies privacy
// redaction on the way out. Callers cannot influence the resolved mode.
type Service struct {
	conversations  repositories.ConversationRepository
	messages       repositories.ConversationMessageRepository
	defaultPrivacy domain.PrivacyMode
	log            *slog.Logger
}

// NewService returns a conversation read service. Log is optional.
func NewService(deps ServiceDeps) *Service {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	if !deps.DefaultPrivacy.Valid() {
		deps.DefaultPrivacy = domain.DefaultPrivacyMode
	}
	return &Service{
		conversations:  deps.Conversations,
		messages:       deps.Messages,
		defaultPrivacy: deps.DefaultPrivacy,
		log:            deps.Log,
	}
}

func (s *Service) List(ctx context.Context, actor domain.Actor, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, error) {
	if err := s.require(actor); err != nil {
		return nil, err
	}
	summaries, err := s.conversations.List(ctx, actor.OrganizationID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		summaries[i].PrivacyState = s.effective(summaries[i].PrivacyState)
	}
	return summaries, nil
}

func (s *Service) Get(ctx context.Context, actor domain.Actor, id uuid.UUID) (domain.ConversationSummary, error) {
	if err := s.require(actor); err != nil {
		return domain.ConversationSummary{}, err
	}
	summary, err := s.conversations.Detail(ctx, actor.OrganizationID, id)
	if err != nil {
		return domain.ConversationSummary{}, err
	}
	summary.PrivacyState = s.effective(summary.PrivacyState)
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
	mode := s.resolve(conversation.PrivacyState)
	messages, err := s.messages.List(ctx, actor.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	for i := range messages {
		messages[i] = messages[i].Redacted(mode)
	}
	return messages, nil
}

func (s *Service) require(actor domain.Actor) error {
	if !actor.Can(domain.PermissionUsageRead) {
		return repositories.ErrForbidden
	}
	return nil
}

// effective reports the mode actually applied rather than the raw stored
// override, which may be NULL.
func (s *Service) effective(override *string) *string {
	mode := string(domain.ResolvePrivacyMode(override, s.defaultPrivacy))
	return &mode
}

func (s *Service) resolve(override *string) domain.PrivacyMode {
	return domain.ResolvePrivacyMode(override, s.defaultPrivacy)
}
