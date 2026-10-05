package repositories_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	repos "ai-governance-saas/backend/internal/infrastructure/postgres/repositories"
)

// newConversation creates one conversation through the repository, which is the
// only write path the API has.
func newConversation(t *testing.T, orgID, userID uuid.UUID, title string) domain.Conversation {
	t.Helper()
	conversation, err := repos.NewConversationRepository(testStore).Create(testCtx, orgID, domain.NewConversation{
		UserID: userID,
		Title:  title,
	})
	requireNoErr(t, err)
	return conversation
}

// addMessage appends one turn and returns it so a test can assert on the
// stored timestamp.
func addMessage(t *testing.T, orgID, userID, conversationID uuid.UUID, role, content string, tokens int32) domain.ConversationMessage {
	t.Helper()
	message, err := repos.NewConversationMessageRepository(testStore).Create(testCtx, orgID, domain.NewConversationMessage{
		ConversationID: conversationID,
		UserID:         userID,
		Role:           role,
		Content:        content,
		Tokens:         tokens,
	})
	requireNoErr(t, err)
	return message
}

// stampConversation fills the optional columns CreateConversation does not
// populate, so filter tests do not depend on a write path that does not exist.
func stampConversation(t *testing.T, orgID, id, projectID uuid.UUID, provider, model, privacy string) {
	t.Helper()
	if _, err := testStore.Pool().Exec(testCtx,
		`UPDATE conversations
		    SET project_id = $1, provider = $2, model = $3, privacy_state = $4
		  WHERE organization_id = $5 AND id = $6`,
		projectID, provider, model, privacy, orgID, id); err != nil {
		t.Fatalf("stamp conversation: %v", err)
	}
}

// setConversationCreatedAt pins created_at so ORDER BY is deterministic
// instead of depending on uuid tie-breaking within one clock tick.
func setConversationCreatedAt(t *testing.T, orgID, id uuid.UUID, createdAt time.Time) {
	t.Helper()
	if _, err := testStore.Pool().Exec(testCtx,
		`UPDATE conversations SET created_at = $1 WHERE organization_id = $2 AND id = $3`,
		createdAt, orgID, id); err != nil {
		t.Fatalf("pin created_at: %v", err)
	}
}

func conversationTitles(summaries []domain.ConversationSummary) []string {
	titles := make([]string, 0, len(summaries))
	for _, s := range summaries {
		titles = append(titles, s.Title)
	}
	return titles
}

func TestConversationCreateDefaultsStatus(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)

	conversation, err := repo.Create(testCtx, orgID, domain.NewConversation{UserID: userID, Title: "triage"})
	requireNoErr(t, err)

	if conversation.Status != domain.ConversationStatusActive {
		t.Fatalf("status = %q, want the database default %q", conversation.Status, domain.ConversationStatusActive)
	}
	if conversation.UserID != userID || conversation.Title != "triage" {
		t.Fatalf("caller fields not round-tripped: %+v", conversation)
	}
	if conversation.ProjectID != nil || conversation.Model != nil ||
		conversation.Provider != nil || conversation.PrivacyState != nil {
		t.Fatalf("unrecorded optional columns must stay nil, got %+v", conversation)
	}
}

func TestConversationListIsTenantScoped(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	otherOrg, otherUser, _ := newTenant(t)

	mine := newConversation(t, orgID, userID, "mine")
	theirs := newConversation(t, otherOrg, otherUser, "theirs")

	got, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 50, 0)
	requireNoErr(t, err)
	if len(got) != 1 || got[0].ID != mine.ID {
		t.Fatalf("tenant A saw %v, want only its own conversation %s", conversationTitles(got), mine.ID)
	}

	other, err := repo.List(testCtx, otherOrg, domain.ConversationFilter{}, 50, 0)
	requireNoErr(t, err)
	if len(other) != 1 || other[0].ID != theirs.ID {
		t.Fatalf("tenant B saw %v, want only its own conversation %s", conversationTitles(other), theirs.ID)
	}
}

func TestConversationListAggregates(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	conversation := newConversation(t, orgID, userID, "with turns")

	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleSystem, "be brief", 5)
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleUser, "hello", 7)
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleAssistant, "hi there", 11)
	newest := addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleTool, "{}", 3)

	got, err := repo.List(testCtx, orgID, domain.ConversationFilter{ID: &conversation.ID}, 50, 0)
	requireNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	summary := got[0]

	// An assistant turn is the only evidence of a provider request, so the
	// single assistant message must count once and the other three not at all.
	if summary.MessageCount != 4 {
		t.Fatalf("message_count = %d, want 4", summary.MessageCount)
	}
	if summary.RequestCount != 1 {
		t.Fatalf("request_count = %d, want 1 (assistant turns only)", summary.RequestCount)
	}
	if summary.TotalTokens != 26 {
		t.Fatalf("total_tokens = %d, want 26", summary.TotalTokens)
	}
	if summary.LastActivityAt == nil {
		t.Fatal("last_activity_at is nil for a conversation that has messages")
	}
	if got := summary.LastActivityAt.UTC(); !got.Equal(newest.CreatedAt.UTC()) {
		t.Fatalf("last_activity_at = %s, want the newest message %s", got, newest.CreatedAt.UTC())
	}
	if summary.StartedAt.IsZero() {
		t.Fatal("started_at is zero")
	}
}

func TestConversationWithNoMessagesHasNoLastActivity(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	conversation := newConversation(t, orgID, userID, "no turns")

	got, err := repo.List(testCtx, orgID, domain.ConversationFilter{ID: &conversation.ID}, 50, 0)
	requireNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("want 1 row, got %d", len(got))
	}
	summary := got[0]

	// The SQL coalesces an empty MAX() to a zero-time sentinel; the repository
	// must drop it back to nil rather than report year 1 as real activity.
	if summary.LastActivityAt != nil {
		t.Fatalf("last_activity_at = %s, want nil for a conversation with no messages", *summary.LastActivityAt)
	}
	if summary.MessageCount != 0 || summary.RequestCount != 0 || summary.TotalTokens != 0 {
		t.Fatalf("empty conversation aggregates = %d/%d/%d, want zeros",
			summary.MessageCount, summary.RequestCount, summary.TotalTokens)
	}
}

func TestConversationListFilters(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, projectID := newTenant(t)
	otherOrg, otherUser, _ := newTenant(t)

	otherUserRepo := repos.NewUserRepository(testStore)
	second, err := otherUserRepo.Create(testCtx, domain.NewUser{
		Email:       uuid.NewString() + "@example.test",
		DisplayName: "Second User",
	})
	requireNoErr(t, err)

	anthropic := newConversation(t, orgID, userID, "anthropic")
	stampConversation(t, orgID, anthropic.ID, projectID, "anthropic", "claude", string(domain.PrivacyFullContent))

	openai := newConversation(t, orgID, second.ID, "openai")
	stampConversation(t, orgID, openai.ID, projectID, "openai", "gpt", string(domain.PrivacyPromptOnly))

	// Belongs to another tenant and must never surface, even though its
	// provider and model match a filter.
	foreign := newConversation(t, otherOrg, otherUser, "foreign")
	stampConversation(t, otherOrg, foreign.ID, projectID, "anthropic", "claude", string(domain.PrivacyOff))

	cases := []struct {
		name   string
		filter domain.ConversationFilter
		want   []string
	}{
		{"no filter returns the tenant's rows only", domain.ConversationFilter{}, []string{"openai", "anthropic"}},
		{"by user", domain.ConversationFilter{UserID: &userID}, []string{"anthropic"}},
		{"by provider", domain.ConversationFilter{Provider: ptr("anthropic")}, []string{"anthropic"}},
		{"by model", domain.ConversationFilter{Model: ptr("gpt")}, []string{"openai"}},
		{"by project", domain.ConversationFilter{ProjectID: &projectID}, []string{"openai", "anthropic"}},
		{"by id", domain.ConversationFilter{ID: &anthropic.ID}, []string{"anthropic"}},
		{"by provider and model together", domain.ConversationFilter{Provider: ptr("anthropic"), Model: ptr("claude")}, []string{"anthropic"}},
		{"unknown provider matches nothing", domain.ConversationFilter{Provider: ptr("nope")}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.List(testCtx, orgID, tc.filter, 50, 0)
			requireNoErr(t, err)
			titles := conversationTitles(got)
			if len(titles) != len(tc.want) {
				t.Fatalf("titles = %v, want %v", titles, tc.want)
			}
			for i := range titles {
				if titles[i] != tc.want[i] {
					t.Fatalf("titles = %v, want %v", titles, tc.want)
				}
			}
		})
	}
}

func TestConversationListFiltersByTimeWindow(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)

	old := newConversation(t, orgID, userID, "old")
	recent := newConversation(t, orgID, userID, "recent")
	base := time.Now().UTC().Truncate(time.Second)
	setConversationCreatedAt(t, orgID, old.ID, base.Add(-48*time.Hour))
	setConversationCreatedAt(t, orgID, recent.ID, base)

	cases := []struct {
		name       string
		from, to   time.Time
		wantTitles []string
	}{
		{"no window", time.Time{}, time.Time{}, []string{"recent", "old"}},
		{"from only", base.Add(-24 * time.Hour), time.Time{}, []string{"recent"}},
		{"to only", time.Time{}, base.Add(-24 * time.Hour), []string{"old"}},
		{"half-open window excludes its upper bound", base.Add(-24 * time.Hour), base, nil},
		{"closed window spanning both", base.Add(-72 * time.Hour), base.Add(time.Hour), []string{"recent", "old"}},
		{"window entirely before both", base.Add(-96 * time.Hour), base.Add(-72 * time.Hour), nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.List(testCtx, orgID, domain.ConversationFilter{From: tc.from, To: tc.to}, 50, 0)
			requireNoErr(t, err)
			titles := conversationTitles(got)
			if len(titles) != len(tc.wantTitles) {
				t.Fatalf("titles = %v, want %v", titles, tc.wantTitles)
			}
			for i := range titles {
				if titles[i] != tc.wantTitles[i] {
					t.Fatalf("titles = %v, want %v", titles, tc.wantTitles)
				}
			}
		})
	}
}

func TestConversationListFiltersByTeamMembership(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)

	inTeam := newConversation(t, orgID, userID, "in team")
	outsider, err := repos.NewUserRepository(testStore).Create(testCtx, domain.NewUser{
		Email:       uuid.NewString() + "@example.test",
		DisplayName: "Outsider",
	})
	requireNoErr(t, err)
	notInTeam := newConversation(t, orgID, outsider.ID, "not in team")

	team, err := repos.NewTeamRepository(testStore).Create(testCtx, orgID, domain.NewTeam{Name: "Guild"})
	requireNoErr(t, err)
	_, err = repos.NewTeamMemberRepository(testStore).Add(testCtx, orgID, team.ID, userID, domain.RoleMember)
	requireNoErr(t, err)

	got, err := repo.List(testCtx, orgID, domain.ConversationFilter{TeamID: &team.ID}, 50, 0)
	requireNoErr(t, err)
	if len(got) != 1 || got[0].ID != inTeam.ID {
		t.Fatalf("team filter returned %v, want only the member's conversation", conversationTitles(got))
	}

	// Guards the assertion above: with a single conversation in the tenant it
	// would still pass if the team_members join were dropped entirely.
	all, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 50, 0)
	requireNoErr(t, err)
	if len(all) != 2 {
		t.Fatalf("unfiltered list returned %v, want both conversations", conversationTitles(all))
	}
	for _, want := range []uuid.UUID{inTeam.ID, notInTeam.ID} {
		if !containsID(all, want) {
			t.Fatalf("unfiltered list %v is missing %s", conversationTitles(all), want)
		}
	}

	// A team with no membership must match nothing rather than everyone.
	empty, err := repos.NewTeamRepository(testStore).Create(testCtx, orgID, domain.NewTeam{Name: "Empty"})
	requireNoErr(t, err)
	none, err := repo.List(testCtx, orgID, domain.ConversationFilter{TeamID: &empty.ID}, 50, 0)
	requireNoErr(t, err)
	if len(none) != 0 {
		t.Fatalf("empty team returned %v, want no conversations", conversationTitles(none))
	}
}

func containsID(summaries []domain.ConversationSummary, id uuid.UUID) bool {
	for _, s := range summaries {
		if s.ID == id {
			return true
		}
	}
	return false
}

func TestConversationListPaginatesNewestFirst(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	base := time.Now().UTC().Truncate(time.Second)

	titles := []string{"first", "second", "third"}
	for i, title := range titles {
		conversation := newConversation(t, orgID, userID, title)
		setConversationCreatedAt(t, orgID, conversation.ID, base.Add(time.Duration(i)*time.Hour))
	}

	firstPage, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 2, 0)
	requireNoErr(t, err)
	if got := conversationTitles(firstPage); len(got) != 2 || got[0] != "third" || got[1] != "second" {
		t.Fatalf("first page = %v, want [third second]", got)
	}

	secondPage, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 2, 2)
	requireNoErr(t, err)
	if got := conversationTitles(secondPage); len(got) != 1 || got[0] != "first" {
		t.Fatalf("second page = %v, want [first]", got)
	}

	beyond, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 2, 99)
	requireNoErr(t, err)
	if len(beyond) != 0 {
		t.Fatalf("offset past the end returned %v, want none", conversationTitles(beyond))
	}
}

func TestConversationListSurfacesRawPrivacyOverride(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, projectID := newTenant(t)

	overridden := newConversation(t, orgID, userID, "overridden")
	stampConversation(t, orgID, overridden.ID, projectID, "", "", string(domain.PrivacyPromptOnly))
	newConversation(t, orgID, userID, "inherits default")

	got, err := repo.List(testCtx, orgID, domain.ConversationFilter{}, 50, 0)
	requireNoErr(t, err)

	byTitle := map[string]*string{}
	for i := range got {
		byTitle[got[i].Title] = got[i].PrivacyState
	}
	if p := byTitle["overridden"]; p == nil || *p != string(domain.PrivacyPromptOnly) {
		t.Fatalf("privacy override = %v, want PROMPT_ONLY surfaced verbatim", p)
	}
	// The repository has no knowledge of the server default, so an absent
	// override must stay nil for the service to resolve.
	if p := byTitle["inherits default"]; p != nil {
		t.Fatalf("privacy override = %q, want nil so the service can apply the server default", *p)
	}
}

func TestConversationDetailReturnsAggregates(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, projectID := newTenant(t)
	conversation := newConversation(t, orgID, userID, "detail")
	stampConversation(t, orgID, conversation.ID, projectID, "anthropic", "claude", string(domain.PrivacyFullContent))
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleUser, "hi", 4)
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleAssistant, "hello", 6)

	got, err := repo.Detail(testCtx, orgID, conversation.ID)
	requireNoErr(t, err)
	if got.ID != conversation.ID || got.Title != "detail" {
		t.Fatalf("Detail identity = %s/%s, want %s/detail", got.ID, got.Title, conversation.ID)
	}
	if got.MessageCount != 2 || got.RequestCount != 1 || got.TotalTokens != 10 {
		t.Fatalf("Detail aggregates = %d/%d/%d, want 2/1/10", got.MessageCount, got.RequestCount, got.TotalTokens)
	}
	if got.ProjectID == nil || *got.ProjectID != projectID {
		t.Fatalf("Detail project_id = %v, want %s", got.ProjectID, projectID)
	}
	if got.Provider == nil || *got.Provider != "anthropic" {
		t.Fatalf("Detail provider = %v, want anthropic", got.Provider)
	}
	if got.PrivacyState == nil || *got.PrivacyState != string(domain.PrivacyFullContent) {
		t.Fatalf("Detail privacy_state = %v, want FULL_CONTENT", got.PrivacyState)
	}

	// Get returns the bare row, Detail adds the aggregates; they must agree on
	// identity so the handler can serve either.
	bare, err := repo.Get(testCtx, orgID, conversation.ID)
	requireNoErr(t, err)
	if bare.ID != got.ID || bare.Title != got.Title {
		t.Fatalf("Get and Detail disagree: %+v vs %+v", bare, got)
	}
}

func TestConversationDetailIsTenantScoped(t *testing.T) {
	repo := repos.NewConversationRepository(testStore)
	orgID, userID, _ := newTenant(t)
	otherOrg, _, _ := newTenant(t)
	conversation := newConversation(t, orgID, userID, "private")

	requireNotFound(t, func() error { _, err := repo.Detail(testCtx, orgID, uuid.New()); return err }())
	requireNotFound(t, func() error { _, err := repo.Detail(testCtx, otherOrg, conversation.ID); return err }())
	requireNotFound(t, func() error { _, err := repo.Get(testCtx, otherOrg, conversation.ID); return err }())
}

func TestConversationMessageListIsScopedAndOrdered(t *testing.T) {
	messages := repos.NewConversationMessageRepository(testStore)
	orgID, userID, _ := newTenant(t)
	otherOrg, otherUser, _ := newTenant(t)

	conversation := newConversation(t, orgID, userID, "thread")
	other := newConversation(t, orgID, otherUser, "other thread")

	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleSystem, "be brief", 1)
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleUser, "first question", 2)
	addMessage(t, orgID, userID, conversation.ID, domain.MessageRoleAssistant, "first answer", 3)
	addMessage(t, orgID, otherUser, other.ID, domain.MessageRoleUser, "belongs elsewhere", 4)

	got, err := messages.List(testCtx, orgID, conversation.ID)
	requireNoErr(t, err)
	if len(got) != 3 {
		t.Fatalf("got %d messages, want 3 (the fourth belongs to another conversation)", len(got))
	}
	wantRoles := []string{domain.MessageRoleSystem, domain.MessageRoleUser, domain.MessageRoleAssistant}
	for i, message := range got {
		if message.Role != wantRoles[i] {
			t.Fatalf("message[%d].Role = %s, want %s (ordering must follow created_at, id)", i, message.Role, wantRoles[i])
		}
		if message.ConversationID != conversation.ID {
			t.Fatalf("message[%d] belongs to conversation %s, want %s", i, message.ConversationID, conversation.ID)
		}
	}

	// The repository returns raw content; redaction is the service's job, so a
	// message read here must still carry its stored text.
	if got[1].Content != "first question" {
		t.Fatalf("message content = %q, want the stored value unredacted at this layer", got[1].Content)
	}

	empty, err := messages.List(testCtx, otherOrg, conversation.ID)
	requireNoErr(t, err)
	if len(empty) != 0 {
		t.Fatalf("another tenant read %d messages, want none", len(empty))
	}
}

func TestConversationMessageListEmptyConversation(t *testing.T) {
	messages := repos.NewConversationMessageRepository(testStore)
	orgID, userID, _ := newTenant(t)
	conversation := newConversation(t, orgID, userID, "silent")

	got, err := messages.List(testCtx, orgID, conversation.ID)
	requireNoErr(t, err)
	if len(got) != 0 {
		t.Fatalf("got %d messages, want none", len(got))
	}
}

func TestConversationMessageCarriesToolMetadata(t *testing.T) {
	messages := repos.NewConversationMessageRepository(testStore)
	orgID, userID, _ := newTenant(t)
	conversation := newConversation(t, orgID, userID, "tool use")

	tool := "web_search"
	providerID := "msg_abc123"
	addMessageWithMetadata(t, orgID, userID, conversation.ID, domain.MessageRoleTool, `{"q":"go"}`, 9, &tool, &providerID)

	got, err := messages.List(testCtx, orgID, conversation.ID)
	requireNoErr(t, err)
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1", len(got))
	}
	if got[0].ToolName == nil || *got[0].ToolName != tool {
		t.Fatalf("tool_name = %v, want %s", got[0].ToolName, tool)
	}
	if got[0].ProviderMessageID == nil || *got[0].ProviderMessageID != providerID {
		t.Fatalf("provider_message_id = %v, want %s", got[0].ProviderMessageID, providerID)
	}
}

// addMessageWithMetadata appends a turn carrying the optional tool columns.
func addMessageWithMetadata(t *testing.T, orgID, userID, conversationID uuid.UUID, role, content string, tokens int32, toolName, providerMessageID *string) {
	t.Helper()
	_, err := repos.NewConversationMessageRepository(testStore).Create(testCtx, orgID, domain.NewConversationMessage{
		ConversationID:    conversationID,
		UserID:            userID,
		Role:              role,
		Content:           content,
		Tokens:            tokens,
		ToolName:          toolName,
		ProviderMessageID: providerMessageID,
	})
	requireNoErr(t, err)
}
