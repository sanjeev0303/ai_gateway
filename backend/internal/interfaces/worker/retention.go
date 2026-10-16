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
	LlmRequests   repositories.LlmRequestRepository
	AuditLogs     repositories.AuditLogRepository
	BillingEvents repositories.BillingEventRepository
	Log           *slog.Logger
}

type RetentionJob struct {
	orgs          repositories.OrganizationRepository
	conversations repositories.ConversationRepository
	usageEvents   repositories.UsageEventRepository
	llmRequests   repositories.LlmRequestRepository
	auditLogs     repositories.AuditLogRepository
	billingEvents repositories.BillingEventRepository
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
		llmRequests:   deps.LlmRequests,
		auditLogs:     deps.AuditLogs,
		billingEvents: deps.BillingEvents,
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

		// 2. Retention for raw usage events and detailed llm requests
		// Prunes raw usage ledger events older than retention_usage_days
		usageDays := org.RetentionUsageDays
		if usageDays > 0 {
			olderThan := now.Add(-time.Duration(usageDays) * 24 * time.Hour)
			if j.usageEvents != nil {
				deleted, err := j.usageEvents.PurgeExpired(ctx, orgID, olderThan)
				if err != nil {
					j.log.ErrorContext(ctx, "retention: usage event purge failed", "org_id", orgID, "error", err)
				} else if deleted > 0 {
					j.log.InfoContext(ctx, "retention: pruned expired usage events", "org_id", orgID, "count", deleted)
				}
			}
			if j.llmRequests != nil {
				deleted, err := j.llmRequests.PurgeExpired(ctx, orgID, olderThan)
				if err != nil {
					j.log.ErrorContext(ctx, "retention: llm requests purge failed", "org_id", orgID, "error", err)
				} else if deleted > 0 {
					j.log.InfoContext(ctx, "retention: pruned expired llm requests", "org_id", orgID, "count", deleted)
				}
			}
		}

		// 3. Retention for audit logs (SEPARATE RETENTION CATEGORY)
		// Prunes administrative audit records older than retention_audits_days
		auditDays := org.RetentionAuditsDays
		if auditDays > 0 && j.auditLogs != nil {
			olderThan := now.Add(-time.Duration(auditDays) * 24 * time.Hour)
			deleted, err := j.auditLogs.PurgeExpired(ctx, orgID, olderThan)
			if err != nil {
				j.log.ErrorContext(ctx, "retention: audit log purge failed", "org_id", orgID, "error", err)
			} else if deleted > 0 {
				j.log.InfoContext(ctx, "retention: pruned expired audit logs", "org_id", orgID, "count", deleted)
			}
		}

		// 4. Retention for billing events (SEPARATE RETENTION CATEGORY)
		// Prunes financial ledger records older than retention_billing_days
		// Never deleted just because conversation or usage retention has expired!
		billingDays := org.RetentionBillingDays
		if billingDays > 0 && j.billingEvents != nil {
			olderThan := now.Add(-time.Duration(billingDays) * 24 * time.Hour)
			deleted, err := j.billingEvents.PurgeExpired(ctx, orgID, olderThan)
			if err != nil {
				j.log.ErrorContext(ctx, "retention: billing event purge failed", "org_id", orgID, "error", err)
			} else if deleted > 0 {
				j.log.InfoContext(ctx, "retention: pruned expired billing events", "org_id", orgID, "count", deleted)
			}
		}
	}

	return nil
}
