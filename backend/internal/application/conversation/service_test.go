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
	deleted   []uuid.UUID
	countsFor *uuid.UUID
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

// fakeTranscripts stands in for gateway memory, the only place text lives.
type fakeTranscripts map[uuid.UUID]domain.Transcript

func (f fakeTranscripts) Transcripts(ctx context.Context, orgID uuid.UUID, ids []uuid.UUID, withMessages bool) (map[uuid.UUID]domain.Transcript, error) {
	out := map[uuid.UUID]domain.Transcript{}
	for _, id := range ids {
		if t, ok := f[id]; ok {
			out[id] = t
		}
	}
	return out, nil
}

func (f fakeTranscripts) Delete(ctx context.Context, orgID, conversationID uuid.UUID) error {
	delete(f, conversationID)
	return nil
}

func (m *mockConvRepo) CountsByKey(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID) ([]domain.ConversationKeyCount, error) {
	m.countsFor = userID
	return nil, nil
}

// Members see only their own counts; owners see everyone's.
func TestConversationService_CountsByKeyScope(t *testing.T) {
	member := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: uuid.New(), UserID: uuid.New(), Role: domain.RoleMember}
	repo := &mockConvRepo{}
	svc := conversation.NewService(conversation.ServiceDeps{Organizations: &mockOrgRepo{}, Conversations: repo, Messages: &mockMessageRepo{}})
	if _, err := svc.CountsByKey(context.Background(), member); err != nil {
		t.Fatal(err)
	}
	if repo.countsFor == nil || *repo.countsFor != member.UserID {
		t.Fatal("a member's counts were not narrowed to themselves")
	}
	owner := member
	owner.Role = domain.RoleOwner
	if _, err := svc.CountsByKey(context.Background(), owner); err != nil || repo.countsFor != nil {
		t.Fatal("an owner's counts were narrowed")
	}
}

func (m *mockConvRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	m.deleted = append(m.deleted, id)
	return nil
}

// Deleting removes the text from gateway memory and the record, and only
// owners and admins may do it.
func TestConversationService_Delete(t *testing.T) {
	ctx := context.Background()
	orgID, convID := uuid.New(), uuid.New()
	for _, role := range []string{domain.RoleOwner, domain.RoleAdmin, domain.RoleAuditor, domain.RoleMember} {
		live := fakeTranscripts{convID: {Title: "hello"}}
		repo := &mockConvRepo{conv: domain.Conversation{ID: convID, OrganizationID: orgID}}
		svc := conversation.NewService(conversation.ServiceDeps{Organizations: &mockOrgRepo{}, Conversations: repo, Messages: &mockMessageRepo{}, Transcripts: live})
		err := svc.Delete(ctx, domain.Actor{Kind: domain.ActorKindUser, OrganizationID: orgID, UserID: uuid.New(), Role: role}, convID)

		mayDelete := role == domain.RoleOwner || role == domain.RoleAdmin
		if mayDelete && (err != nil || len(live) != 0 || len(repo.deleted) != 1) {
			t.Errorf("%s: err %v, text left %d, records deleted %d", role, err, len(live), len(repo.deleted))
		}
		if !mayDelete && (err != repositories.ErrForbidden || len(live) != 1 || len(repo.deleted) != 0) {
			t.Errorf("%s: should be forbidden and change nothing, got err %v", role, err)
		}
	}
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
		{ID: uuid.New(), ConversationID: convID, Role: "system", Tokens: 10},
		{ID: uuid.New(), ConversationID: convID, Role: "user", Tokens: 5},
		{ID: uuid.New(), ConversationID: convID, Role: "assistant", Tokens: 8},
	}
	live := fakeTranscripts{convID: {Messages: map[uuid.UUID]string{
		messages[0].ID: "You are a helpful assistant.",
		messages[1].ID: "What is Sentinel?",
		messages[2].ID: "Sentinel is an enterprise AI gateway.",
	}}}

	tests := []struct {
		name        string
		privacyMode domain.PrivacyMode
		wantSystem  string
		wantUser    string
		wantAssist  string
	}{
		{
			name:        "METADATA_ONLY still shows the transcript",
			privacyMode: domain.PrivacyMetadataOnly,
			wantSystem:  "You are a helpful assistant.",
			wantUser:    "What is Sentinel?",
			wantAssist:  "Sentinel is an enterprise AI gateway.",
		},
		{
			name:        "PROMPT_ONLY still shows the transcript",
			privacyMode: domain.PrivacyPromptOnly,
			wantSystem:  "You are a helpful assistant.",
			wantUser:    "What is Sentinel?",
			wantAssist:  "Sentinel is an enterprise AI gateway.",
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
				Transcripts:   live,
			})

			res, _, err := svc.ListMessages(ctx, actor, convID)
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

	_, _, err := svc.ListMessages(ctx, actor, convID)
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	_, _, err = svc.List(ctx, actor, domain.ConversationFilter{}, 10, 0)
	if err != repositories.ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// With the text gone from gateway memory the turns stay empty and the state
// says why, so the console can explain it instead of showing blank bubbles.
func TestConversationService_TranscriptStates(t *testing.T) {
	ctx := context.Background()
	orgID, convID := uuid.New(), uuid.New()
	actor := domain.Actor{Kind: domain.ActorKindUser, OrganizationID: orgID, UserID: uuid.New(), Role: domain.RoleOwner}
	msg := domain.ConversationMessage{ID: uuid.New(), ConversationID: convID, Role: "user", Tokens: 3}

	cases := []struct {
		name string
		mode domain.PrivacyMode
		live fakeTranscripts
		want conversation.TranscriptState
		text string
	}{
		{"live text shown", domain.PrivacyFullContent, fakeTranscripts{convID: {Messages: map[uuid.UUID]string{msg.ID: "hi"}}}, conversation.TranscriptAvailable, "hi"},
		{"gone from memory", domain.PrivacyFullContent, fakeTranscripts{}, conversation.TranscriptExpired, ""},
		{"privacy mode no longer hides it", domain.PrivacyMetadataOnly, fakeTranscripts{convID: {Messages: map[uuid.UUID]string{msg.ID: "hi"}}}, conversation.TranscriptAvailable, "hi"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := conversation.NewService(conversation.ServiceDeps{
				Organizations: &mockOrgRepo{org: domain.Organization{ID: orgID, PrivacyDefault: tc.mode}},
				Conversations: &mockConvRepo{conv: domain.Conversation{ID: convID, OrganizationID: orgID}},
				Messages:      &mockMessageRepo{messages: []domain.ConversationMessage{msg}},
				Transcripts:   tc.live,
			})
			res, state, err := svc.ListMessages(ctx, actor, convID)
			if err != nil {
				t.Fatal(err)
			}
			if state != tc.want || res[0].Content != tc.text {
				t.Fatalf("got state %q text %q, want %q %q", state, res[0].Content, tc.want, tc.text)
			}
		})
	}
}
