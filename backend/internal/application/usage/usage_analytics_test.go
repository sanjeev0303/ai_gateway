package usage

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/ports/repositories"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

type mockLlmRequestRepo struct {
	requests []domain.LlmRequest
}

func (m *mockLlmRequestRepo) Record(_ context.Context, _ uuid.UUID, in domain.NewLlmRequest) (domain.LlmRequest, error) {
	req := domain.LlmRequest{
		ID:                 uuid.New(),
		RequestID:          in.RequestID,
		ProjectID:          in.ProjectID,
		UserID:             in.UserID,
		ServiceAccountID:   in.ServiceAccountID,
		TeamID:             in.TeamID,
		Model:              in.Model,
		ProviderKind:       in.ProviderKind,
		Status:             in.Status,
		PromptTokens:       in.PromptTokens,
		CompletionTokens:   in.CompletionTokens,
		CachedInputTokens:  in.CachedInputTokens,
		TotalTokens:        in.TotalTokens,
		LatencyMs:          in.LatencyMs,
		CostMicros:         in.CostMicros,
		CreatedAt:          in.CreatedAt,
		TimeToFirstTokenMs: in.TimeToFirstTokenMs,
		TotalLatencyMs:     in.TotalLatencyMs,
	}
	m.requests = append(m.requests, req)
	return req, nil
}

func (m *mockLlmRequestRepo) GetByRequestID(_ context.Context, _ uuid.UUID, id string) (domain.LlmRequest, error) {
	for _, r := range m.requests {
		if r.RequestID == id {
			return r, nil
		}
	}
	return domain.LlmRequest{}, repositories.ErrNotFound
}

func (m *mockLlmRequestRepo) List(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]domain.LlmRequest, error) {
	return m.requests, nil
}

func (m *mockLlmRequestRepo) GetRecent(_ context.Context, _ uuid.UUID) ([]domain.LlmRequest, error) {
	return m.requests, nil
}

func (m *mockLlmRequestRepo) GetRequests(_ context.Context, _ uuid.UUID, _ domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, error) {
	start := int(offset)
	if start >= len(m.requests) {
		return nil, nil
	}
	end := start + int(limit)
	if end > len(m.requests) {
		end = len(m.requests)
	}
	return m.requests[start:end], nil
}

func (m *mockLlmRequestRepo) PurgeExpired(context.Context, uuid.UUID, time.Time) (int64, error) {
	return 0, nil
}

type mockRollupRepo struct {
	fakeRollups
	summary          domain.UsageSummary
	byUser           []domain.UsageByUser
	byServiceAccount []domain.UsageByServiceAccount
}

func (m *mockRollupRepo) GetSummary(context.Context, uuid.UUID, domain.UsageFilter) (domain.UsageSummary, error) {
	return m.summary, nil
}

func (m *mockRollupRepo) GetByUser(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByUser, error) {
	return m.byUser, nil
}

func (m *mockRollupRepo) GetByServiceAccount(context.Context, uuid.UUID, domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	return m.byServiceAccount, nil
}

func TestUsageAnalytics_AttributionSeparation(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	saID := uuid.New()

	actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		OrganizationID: orgID,
		UserID:         userID,
		Role:           domain.RoleAdmin,
	}

	userMetric := domain.UsageByUser{
		UserID:      userID,
		Email:       "alice@example.com",
		DisplayName: "Alice",
		UsageTotals: domain.UsageTotals{
			Requests:    10,
			TotalTokens: 5000,
			CostMicros:  50000,
		},
	}

	saMetric := domain.UsageByServiceAccount{
		ServiceAccountID: saID,
		Name:             "CI/CD Pipeline Service Account",
		UsageTotals: domain.UsageTotals{
			Requests:    25,
			TotalTokens: 12000,
			CostMicros:  120000,
		},
	}

	rollups := &mockRollupRepo{
		byUser:           []domain.UsageByUser{userMetric},
		byServiceAccount: []domain.UsageByServiceAccount{saMetric},
	}

	svc := NewService(ServiceDeps{
		Rollups: rollups,
		Log:     discardLogger(),
	})

	filter := domain.UsageFilter{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	// 1. Verify GetByUser returns only human users and not service accounts
	users, err := svc.GetByUser(context.Background(), actor, filter)
	if err != nil {
		t.Fatalf("GetByUser failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, users[0].UserID)
	}

	// 2. Verify GetByServiceAccount returns only service accounts and not human users
	sas, err := svc.GetByServiceAccount(context.Background(), actor, filter)
	if err != nil {
		t.Fatalf("GetByServiceAccount failed: %v", err)
	}
	if len(sas) != 1 {
		t.Fatalf("expected 1 service account, got %d", len(sas))
	}
	if sas[0].ServiceAccountID != saID {
		t.Fatalf("expected saID %s, got %s", saID, sas[0].ServiceAccountID)
	}
}

func TestUsageAnalytics_SummaryReconciliation(t *testing.T) {
	orgID := uuid.New()
	actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		OrganizationID: orgID,
		Role:           domain.RoleAdmin,
	}

	expectedSummary := domain.UsageSummary{
		Requests:        100,
		InputTokens:     80000,
		OutputTokens:    20000,
		CachedTokens:    15000,
		TotalTokens:     100000,
		TotalCostMicros: 250000,
		Errors:          2,
	}

	rollups := &mockRollupRepo{
		summary: expectedSummary,
	}

	svc := NewService(ServiceDeps{
		Rollups: rollups,
		Log:     discardLogger(),
	})

	filter := domain.UsageFilter{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	summary, err := svc.GetSummary(context.Background(), actor, filter)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}

	if summary.Requests != 100 {
		t.Errorf("Requests = %d, want 100", summary.Requests)
	}
	if summary.TotalTokens != 100000 {
		t.Errorf("TotalTokens = %d, want 100000", summary.TotalTokens)
	}
	if summary.InputTokens != 80000 {
		t.Errorf("InputTokens = %d, want 80000", summary.InputTokens)
	}
	if summary.OutputTokens != 20000 {
		t.Errorf("OutputTokens = %d, want 20000", summary.OutputTokens)
	}
	if summary.CachedTokens != 15000 {
		t.Errorf("CachedTokens = %d, want 15000", summary.CachedTokens)
	}
	if summary.TotalCostMicros != 250000 {
		t.Errorf("TotalCostMicros = %d, want 250000", summary.TotalCostMicros)
	}
	if summary.Errors != 2 {
		t.Errorf("Errors = %d, want 2", summary.Errors)
	}
}

func TestUsageAnalytics_Pagination(t *testing.T) {
	orgID := uuid.New()
	actor := domain.Actor{
		Kind:           domain.ActorKindUser,
		OrganizationID: orgID,
		Role:           domain.RoleAdmin,
	}

	reqRepo := &mockLlmRequestRepo{}
	for i := 0; i < 75; i++ {
		_, _ = reqRepo.Record(context.Background(), orgID, domain.NewLlmRequest{
			RequestID:   uuid.New().String(),
			TotalTokens: int32(i * 10),
		})
	}

	svc := NewService(ServiceDeps{
		LlmRequests: reqRepo,
		Log:         discardLogger(),
	})

	filter := domain.UsageFilter{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	// Page 1: limit 50, offset 0 -> returns 50, hasMore = true
	items1, hasMore1, err := svc.GetRequests(context.Background(), actor, filter, 50, 0)
	if err != nil {
		t.Fatalf("GetRequests page 1 failed: %v", err)
	}
	if len(items1) != 50 {
		t.Errorf("expected 50 items on page 1, got %d", len(items1))
	}
	if !hasMore1 {
		t.Errorf("expected hasMore1 = true")
	}

	// Page 2: limit 50, offset 50 -> returns 25, hasMore = false
	items2, hasMore2, err := svc.GetRequests(context.Background(), actor, filter, 50, 50)
	if err != nil {
		t.Fatalf("GetRequests page 2 failed: %v", err)
	}
	if len(items2) != 25 {
		t.Errorf("expected 25 items on page 2, got %d", len(items2))
	}
	if hasMore2 {
		t.Errorf("expected hasMore2 = false")
	}
}

func TestUsageAnalytics_VersionedPricingEffectiveFrom(t *testing.T) {
	// Verifies pricing resolution logic: historical requests must resolve
	// the pricing version effective at the time of their creation.
	type priceVersion struct {
		version      int32
		inputMicros  int64
		outputMicros int64
		effective    time.Time
	}

	resolvePricing := func(reqTime time.Time, versions []priceVersion) (int64, int64, bool) {
		var matched *priceVersion
		for _, v := range versions {
			if !v.effective.After(reqTime) {
				if matched == nil || v.effective.After(matched.effective) || (v.effective.Equal(matched.effective) && v.version > matched.version) {
					vCopy := v
					matched = &vCopy
				}
			}
		}
		if matched == nil {
			return 0, 0, false
		}
		return matched.inputMicros, matched.outputMicros, true
	}

	now := time.Now().UTC()
	rateSheet := []priceVersion{
		{version: 1, inputMicros: 1, outputMicros: 2, effective: now.Add(-60 * 24 * time.Hour)},
		{version: 2, inputMicros: 3, outputMicros: 6, effective: now.Add(-10 * 24 * time.Hour)},
	}

	// 1. Request from 30 days ago should match Version 1
	reqOld := now.Add(-30 * 24 * time.Hour)
	in1, out1, ok1 := resolvePricing(reqOld, rateSheet)
	if !ok1 || in1 != 1 || out1 != 2 {
		t.Fatalf("expected Version 1 (1, 2), got (%d, %d, %v)", in1, out1, ok1)
	}

	// 2. Request from 2 days ago should match Version 2
	reqNew := now.Add(-2 * 24 * time.Hour)
	in2, out2, ok2 := resolvePricing(reqNew, rateSheet)
	if !ok2 || in2 != 3 || out2 != 6 {
		t.Fatalf("expected Version 2 (3, 6), got (%d, %d, %v)", in2, out2, ok2)
	}

	// 3. Request before any effective date has no matching rate sheet
	reqAncient := now.Add(-90 * 24 * time.Hour)
	_, _, ok3 := resolvePricing(reqAncient, rateSheet)
	if ok3 {
		t.Fatalf("expected no pricing version before earliest effective date")
	}
}
