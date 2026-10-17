package conversation_test

import (
	"context"
	"testing"

	"ai-governance-saas/backend/internal/application/conversation"
	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

type mockOrgRepo struct {
	repositories.OrganizationRepository
	org domain.Organization
}

func (m *mockOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	return m.org, nil
}

type mockConvRepo struct {
	repositories.ConversationRepository
	conv      domain.Conversation
	summaries []domain.ConversationSummary
}

func (m *mockConvRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Conversation, error) {
	return m.conv, nil
}

func (m *mockConvRepo) Detail(ctx context.Context, orgID, id uuid.UUID) (domain.ConversationSummary, error) {
	return m.summaries[0], nil
}

func (m *mockConvRepo) List(ctx context.Context, orgID uuid.UUID, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, error) {
	return m.summaries, nil
}

type mockMessageRepo struct {
	repositories.ConversationMessageRepository
	messages []domain.ConversationMessage
}

func (m *mockMessageRepo) List(ctx context.Context, orgID, conversationID uuid.UUID) ([]domain.ConversationMessage, error) {
	out := make([]domain.ConversationMessage, len(m.messages))
	copy(out, m.messages)
	return out, nil
}

func TestConversationService_PrivacyModes(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	convID := uuid.New()
	userID := uuid.New()

	actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		OrganizationID: orgID,
		UserID:         userID,
		Role:           domain.RoleOwner, // grants PermissionUsageRead
	}

	messages := []domain.ConversationMessage{
		{
			ID:             uuid.New(),
			ConversationID: convID,
			Role:           "system",
			Content:        "You are a helpful assistant.",
			Tokens:         10,
		},
		{
			ID:             uuid.New(),
			ConversationID: convID,
			Role:           "user",
			Content:        "What is Sentinel?",
			Tokens:         5,
		},
		{
			ID:             uuid.New(),
			ConversationID: convID,
			Role:           "assistant",
			Content:        "Sentinel is an enterprise AI gateway.",
			Tokens:         8,
		},
	}

	tests := []struct {
		name        string
		privacyMode domain.PrivacyMode
		wantSystem  string
		wantUser    string
		wantAssist  string
	}{
		{
			name:        "METADATA_ONLY redacts all turn content",
			privacyMode: domain.PrivacyMetadataOnly,
			wantSystem:  "",
			wantUser:    "",
			wantAssist:  "",
		},
		{
			name:        "PROMPT_ONLY keeps prompts and redacts assistant turns",
			privacyMode: domain.PrivacyPromptOnly,
			wantSystem:  "You are a helpful assistant.",
			wantUser:    "What is Sentinel?",
			wantAssist:  "",
		},
		{
			name:        "FULL_CONTENT preserves all turns",
			privacyMode: domain.PrivacyFullContent,
			wantSystem:  "You are a helpful assistant.",
			wantUser:    "What is Sentinel?",
			wantAssist:  "Sentinel is an enterprise AI gateway.",
		},
		{
			name:        "OFF preserves all turns",
			privacyMode: domain.PrivacyOff,
			wantSystem:  "You are a helpful assistant.",
			wantUser:    "What is Sentinel?",
			wantAssist:  "Sentinel is an enterprise AI gateway.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			orgRepo := &mockOrgRepo{
				org: domain.Organization{
					ID:             orgID,
					PrivacyDefault: tc.privacyMode,
				},
			}
			convRepo := &mockConvRepo{
				conv: domain.Conversation{
					ID:             convID,
					OrganizationID: orgID,
				},
			}
			msgRepo := &mockMessageRepo{messages: messages}

			svc := conversation.NewService(conversation.ServiceDeps{
				Organizations: orgRepo,
				Conversations: convRepo,
				Messages:      msgRepo,
			})

			res, err := svc.ListMessages(ctx, actor, convID)
			if err != nil {
				t.Fatalf("ListMessages failed: %v", err)
			}

			if len(res) != 3 {
				t.Fatalf("expected 3 messages, got %d", len(res))
			}

			// Verify tokens and metadata remain available regardless of privacy mode
			if res[0].Tokens != 10 || res[1].Tokens != 5 || res[2].Tokens != 8 {
				t.Fatalf("metadata tokens must remain available under all modes")
			}

			if res[0].Content != tc.wantSystem {
				t.Errorf("system turn = %q, want %q", res[0].Content, tc.wantSystem)
			}
			if res[1].Content != tc.wantUser {
				t.Errorf("user turn = %q, want %q", res[1].Content, tc.wantUser)
			}
			if res[2].Content != tc.wantAssist {
				t.Errorf("assistant turn = %q, want %q", res[2].Content, tc.wantAssist)
			}
		})
	}
}

func TestConversationService_PermissionDenied(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	convID := uuid.New()

	// Member role does not have PermissionUsageRead by default
	actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		OrganizationID: orgID,
		UserID:         uuid.New(),
		Role:           "guest", // No permissions
	}

	svc := conversation.NewService(conversation.ServiceDeps{
		Organizations: &mockOrgRepo{},
		Conversations: &mockConvRepo{},
		Messages:      &mockMessageRepo{},
	})

	_, err := svc.ListMessages(ctx, actor, convID)
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	_, _, err = svc.List(ctx, actor, domain.ConversationFilter{}, 10, 0)
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}
