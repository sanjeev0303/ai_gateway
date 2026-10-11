package worker_test

import (
	"context"
	"testing"
	"time"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/interfaces/worker"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

type mockRetentionOrgRepo struct {
	repositories.OrganizationRepository
	orgs map[uuid.UUID]domain.Organization
}

func (m *mockRetentionOrgRepo) ListAllIDs(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	for id := range m.orgs {
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *mockRetentionOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (domain.Organization, error) {
	if o, ok := m.orgs[id]; ok {
		return o, nil
	}
	return domain.Organization{}, repositories.ErrNotFound
}

type mockRetentionConvRepo struct {
	repositories.ConversationRepository
	purgedCalls int
	lastOlderThan time.Time
	deletedCount  int64
}

func (m *mockRetentionConvRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	m.purgedCalls++
	m.lastOlderThan = olderThan
	return m.deletedCount, nil
}

type mockRetentionUsageRepo struct {
	repositories.UsageEventRepository
	purgedCalls int
	lastOlderThan time.Time
	deletedCount  int64
}

func (m *mockRetentionUsageRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	m.purgedCalls++
	m.lastOlderThan = olderThan
	return m.deletedCount, nil
}

func TestRetentionWorker_RunOnce(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	orgRepo := &mockRetentionOrgRepo{
		orgs: map[uuid.UUID]domain.Organization{
			orgID: {
				ID:                         orgID,
				RetentionConversationsDays: 30,
				RetentionUsageDays:         90,
				RetentionAuditsDays:        365,  // Audit logs retained 1 year
				RetentionBillingDays:       2555, // Billing retained 7 years
			},
		},
	}

	convRepo := &mockRetentionConvRepo{deletedCount: 15}
	usageRepo := &mockRetentionUsageRepo{deletedCount: 42}

	job := worker.NewRetentionJob(worker.RetentionDeps{
		Orgs:          orgRepo,
		Conversations: convRepo,
		UsageEvents:   usageRepo,
	})

	before := time.Now().UTC()
	err := job.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	if convRepo.purgedCalls != 1 {
		t.Fatalf("expected 1 conversation purge call, got %d", convRepo.purgedCalls)
	}
	expectedConvCutoff := before.Add(-30 * 24 * time.Hour)
	if convRepo.lastOlderThan.Sub(expectedConvCutoff).Abs() > 2*time.Second {
		t.Fatalf("unexpected conversation cutoff: got %v, want near %v", convRepo.lastOlderThan, expectedConvCutoff)
	}

	if usageRepo.purgedCalls != 1 {
		t.Fatalf("expected 1 usage purge call, got %d", usageRepo.purgedCalls)
	}
	expectedUsageCutoff := before.Add(-90 * 24 * time.Hour)
	if usageRepo.lastOlderThan.Sub(expectedUsageCutoff).Abs() > 2*time.Second {
		t.Fatalf("unexpected usage cutoff: got %v, want near %v", usageRepo.lastOlderThan, expectedUsageCutoff)
	}
}
