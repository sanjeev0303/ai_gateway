package worker

import (
	"context"
	"log/slog"
	"time"

	repositories "ai-governance-saas/backend/internal/ports/repositories"
)

type RetentionDeps struct {
	Orgs          repositories.OrganizationRepository
	Conversations repositories.ConversationRepository
	UsageEvents   repositories.UsageEventRepository
	Log           *slog.Logger
}

type RetentionJob struct {
	orgs          repositories.OrganizationRepository
	conversations repositories.ConversationRepository
	usageEvents   repositories.UsageEventRepository
	log           *slog.Logger
}

func NewRetentionJob(deps RetentionDeps) *RetentionJob {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &RetentionJob{
		orgs:          deps.Orgs,
		conversations: deps.Conversations,
		usageEvents:   deps.UsageEvents,
		log:           deps.Log,
	}
}

// RunRetention executes retention passes periodically.
func RunRetention(ctx context.Context, job *RetentionJob, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial pass
	if err := job.RunOnce(ctx); err != nil {
		job.log.Error("initial retention pass failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			job.log.Info("retention worker stopped")
			return
		case <-ticker.C:
			if err := job.RunOnce(ctx); err != nil {
				job.log.Error("retention pass failed", "error", err)
			}
		}
	}
}

func (j *RetentionJob) RunOnce(ctx context.Context) error {
	orgIDs, err := j.orgs.ListAllIDs(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, orgID := range orgIDs {
		org, err := j.orgs.GetByID(ctx, orgID)
		if err != nil {
			j.log.ErrorContext(ctx, "retention: failed to get org", "org_id", orgID, "error", err)
			continue
		}

		// 1. Retention for conversations
		// Prunes conversations and cascading messages older than retention_conversations_days
		convDays := org.RetentionConversationsDays
		if convDays > 0 {
			olderThan := now.Add(-time.Duration(convDays) * 24 * time.Hour)
			deleted, err := j.conversations.PurgeExpired(ctx, orgID, olderThan)
			if err != nil {
				j.log.ErrorContext(ctx, "retention: conversation purge failed", "org_id", orgID, "error", err)
			} else if deleted > 0 {
				j.log.InfoContext(ctx, "retention: pruned expired conversations", "org_id", orgID, "count", deleted)
			}
		}

		// 2. Retention for raw usage events
		// Prunes raw usage ledger events older than retention_usage_days
		usageDays := org.RetentionUsageDays
		if usageDays > 0 {
			olderThan := now.Add(-time.Duration(usageDays) * 24 * time.Hour)
			deleted, err := j.usageEvents.PurgeExpired(ctx, orgID, olderThan)
			if err != nil {
				j.log.ErrorContext(ctx, "retention: usage event purge failed", "org_id", orgID, "error", err)
			} else if deleted > 0 {
				j.log.InfoContext(ctx, "retention: pruned expired usage events", "org_id", orgID, "count", deleted)
			}
		}

		// CRITICAL SAFETY REQUIREMENT:
		// Audit records (retention_audits_days) and financial records (retention_billing_days)
		// are strictly retained according to their own requirements (minimum 365 days / 2555 days).
		// Expired conversations NEVER delete immutable audit logs or financial billing records.
	}

	return nil
}
