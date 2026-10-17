package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"ai-governance-saas/backend/internal/domain"
	"ai-governance-saas/backend/internal/infrastructure/postgres"
	db "ai-governance-saas/backend/internal/infrastructure/postgres/sqlc"
	ports "ai-governance-saas/backend/internal/ports/repositories"
)

type conversationRepo struct {
	store *postgres.Store
}

// NewConversationRepository returns a ConversationRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewConversationRepository(store *postgres.Store) ports.ConversationRepository {
	return &conversationRepo{store: store}
}

func (r *conversationRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewConversation) (domain.Conversation, error) {
	var conversation domain.Conversation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateConversation(ctx, db.CreateConversationParams{
			OrganizationID: orgID,
			UserID:         input.UserID,
			Title:          input.Title,
			Status:         domain.ConversationStatusActive,
		})
		if err != nil {
			return err
		}
		conversation = toConversation(row)
		return nil
	})
	return conversation, err
}

func (r *conversationRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Conversation, error) {
	var conversation domain.Conversation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetConversation(ctx, db.GetConversationParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		conversation = toConversation(row)
		return nil
	})
	return conversation, mapNotFound(err)
}

func (r *conversationRepo) Detail(ctx context.Context, orgID, id uuid.UUID) (domain.ConversationSummary, error) {
	var summary domain.ConversationSummary
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListConversations(ctx, db.ListConversationsParams{
			OrganizationID: orgID,
			ID:             &id,
			LimitCount:     1,
		})
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return pgx.ErrNoRows
		}
		summary = toConversationSummary(rows[0])
		return nil
	})
	return summary, mapNotFound(err)
}

func (r *conversationRepo) List(ctx context.Context, orgID uuid.UUID, filter domain.ConversationFilter, limit, offset int32) ([]domain.ConversationSummary, error) {
	var summaries []domain.ConversationSummary
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListConversations(ctx, db.ListConversationsParams{
			OrganizationID: orgID,
			ID:             filter.ID,
			UserID:         filter.UserID,
			ProjectID:      filter.ProjectID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			FromTs:         timeOrNil(filter.From),
			ToTs:           timeOrNil(filter.To),
			TeamID:         filter.TeamID,
			LimitCount:     limit,
			OffsetCount:    offset,
		})
		if err != nil {
			return err
		}
		summaries = make([]domain.ConversationSummary, 0, len(rows))
		for _, row := range rows {
			summaries = append(summaries, toConversationSummary(row))
		}
		return nil
	})
	return summaries, err
}

func (r *conversationRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.PurgeExpiredConversations(ctx, db.PurgeExpiredConversationsParams{
			OrganizationID: orgID,
			OlderThan:      olderThan,
		})
		if err != nil {
			return err
		}
		count = n
		return nil
	})
	return count, err
}

type conversationMessageRepo struct {
	store *postgres.Store
}

// NewConversationMessageRepository returns a ConversationMessageRepository
// backed by the store. All operations are tenant-scoped via the orgID context.
func NewConversationMessageRepository(store *postgres.Store) ports.ConversationMessageRepository {
	return &conversationMessageRepo{store: store}
}

func (r *conversationMessageRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewConversationMessage) (domain.ConversationMessage, error) {
	var message domain.ConversationMessage
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		contentToStore := input.Content
		conv, err := q.GetConversation(ctx, db.GetConversationParams{OrganizationID: orgID, ID: input.ConversationID})
		if err == nil && conv.PrivacyState != nil && *conv.PrivacyState != "" {
			mode := domain.PrivacyMode(*conv.PrivacyState)
			if !mode.ContentVisible(input.Role) {
				contentToStore = ""
			}
		}

		row, err := q.CreateConversationMessage(ctx, db.CreateConversationMessageParams{
			ConversationID:    input.ConversationID,
			OrganizationID:    orgID,
			UserID:            input.UserID,
			Role:              input.Role,
			Content:           contentToStore,
			Tokens:            input.Tokens,
			ToolName:          input.ToolName,
			ProviderMessageID: input.ProviderMessageID,
		})
		if err != nil {
			return err
		}
		message = toConversationMessage(row)
		return nil
	})
	return message, err
}

func (r *conversationMessageRepo) List(ctx context.Context, orgID, conversationID uuid.UUID) ([]domain.ConversationMessage, error) {
	var messages []domain.ConversationMessage
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListConversationMessages(ctx, db.ListConversationMessagesParams{
			OrganizationID: orgID,
			ConversationID: conversationID,
		})
		if err != nil {
			return err
		}
		messages = make([]domain.ConversationMessage, 0, len(rows))
		for _, row := range rows {
			messages = append(messages, toConversationMessage(row))
		}
		return nil
	})
	return messages, err
}

type llmRequestRepo struct {
	store *postgres.Store
}

// NewLlmRequestRepository returns a LlmRequestRepository backed by the store.
// All operations are tenant-scoped via the orgID context.
func NewLlmRequestRepository(store *postgres.Store) ports.LlmRequestRepository {
	return &llmRequestRepo{store: store}
}

func (r *llmRequestRepo) Record(ctx context.Context, orgID uuid.UUID, input domain.NewLlmRequest) (domain.LlmRequest, error) {
	var llmRequest domain.LlmRequest
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	status := input.Status
	switch status {
	case "success", "succeeded":
		status = "succeeded"
	case "error", "failed":
		status = "failed"
	case "cancelled", "canceled":
		status = "cancelled"
	default:
		status = "succeeded"
	}
	costStatus := input.CostStatus
	if costStatus == "" {
		costStatus = domain.CostStatusCalculated
	}
	policyDecision := input.PolicyDecision
	if policyDecision == nil {
		d := domain.PolicyDecisionAllowed
		if status == "failed" && input.DenialReason != nil && *input.DenialReason != "" {
			d = domain.PolicyDecisionDenied
		}
		policyDecision = &d
	}
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RecordLLMRequest(ctx, db.RecordLLMRequestParams{
			OrganizationID:      orgID,
			ProjectID:           input.ProjectID,
			UserID:              input.UserID,
			TeamID:              input.TeamID,
			GatewayCredentialID: input.GatewayCredentialID,
			RequestID:           input.RequestID,
			ProviderRequestID:   input.ProviderRequestID,
			ProviderKind:        input.ProviderKind,
			Model:               input.Model,
			Status:              status,
			ErrorCode:           input.ErrorCode,
			PromptTokens:        input.PromptTokens,
			CompletionTokens:    input.CompletionTokens,
			CachedInputTokens:   input.CachedInputTokens,
			TotalTokens:         input.TotalTokens,
			LatencyMs:           input.LatencyMs,
			CostMicros:          input.CostMicros,
			CreatedAt:           createdAt,
			RequestStartedAt:    input.RequestStartedAt,
			FirstTokenAt:        input.FirstTokenAt,
			CompletedAt:         input.CompletedAt,
			TimeToFirstTokenMs:  input.TimeToFirstTokenMs,
			TotalLatencyMs:      input.TotalLatencyMs,
			ServiceAccountID:    input.ServiceAccountID,
			ReasoningTokens:     input.ReasoningTokens,
			ToolTokens:          input.ToolTokens,
			PricingVersion:      input.PricingVersion,
			CostStatus:          costStatus,
			PolicyDecision:      policyDecision,
			DenialReason:        input.DenialReason,
		})
		if err != nil {
			return err
		}
		llmRequest = toLlmRequest(row)
		return nil
	})
	return llmRequest, mapConflict(err)
}

func (r *llmRequestRepo) GetByRequestID(ctx context.Context, orgID uuid.UUID, requestID string) (domain.LlmRequest, error) {
	var llmRequest domain.LlmRequest
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetLLMRequestByRequestID(ctx, db.GetLLMRequestByRequestIDParams{
			OrganizationID: orgID,
			RequestID:      requestID,
		})
		if err != nil {
			return err
		}
		llmRequest = toLlmRequest(row)
		return nil
	})
	return llmRequest, mapNotFound(err)
}

func (r *llmRequestRepo) List(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.LlmRequest, error) {
	var llmRequests []domain.LlmRequest
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListLLMRequests(ctx, db.ListLLMRequestsParams{
			OrganizationID: orgID,
			CreatedAt:      from,
			CreatedAt_2:    to,
		})
		if err != nil {
			return err
		}
		llmRequests = make([]domain.LlmRequest, 0, len(rows))
		for _, row := range rows {
			llmRequests = append(llmRequests, toLlmRequest(row))
		}
		return nil
	})
	return llmRequests, err
}

func (r *llmRequestRepo) GetRecent(ctx context.Context, orgID uuid.UUID) ([]domain.LlmRequest, error) {
	var llmRequests []domain.LlmRequest
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetRecentUsage(ctx, orgID)
		if err != nil {
			return err
		}
		llmRequests = make([]domain.LlmRequest, 0, len(rows))
		for _, row := range rows {
			llmRequests = append(llmRequests, domain.LlmRequest{
				ID:                  row.ID,
				OrganizationID:      orgID,
				ProjectID:           row.ProjectID,
				UserID:              row.UserID,
				ServiceAccountID:    row.ServiceAccountID,
				TeamID:              row.TeamID,
				GatewayCredentialID: row.GatewayCredentialID,
				RequestID:           row.RequestID,
				ProviderKind:        row.ProviderKind,
				Model:               row.Model,
				Status:              row.Status,
				ErrorCode:           row.ErrorCode,
				PromptTokens:        row.PromptTokens,
				CompletionTokens:    row.CompletionTokens,
				CachedInputTokens:   row.CachedInputTokens,
				ReasoningTokens:     row.ReasoningTokens,
				ToolTokens:          row.ToolTokens,
				TotalTokens:         row.TotalTokens,
				LatencyMs:           row.LatencyMs,
				CostMicros:          row.CostMicros,
				PricingVersion:      row.PricingVersion,
				CostStatus:          row.CostStatus,
				PolicyDecision:      row.PolicyDecision,
				DenialReason:        row.DenialReason,
				CreatedAt:           row.CreatedAt,
				RequestStartedAt:    row.RequestStartedAt,
				FirstTokenAt:        row.FirstTokenAt,
				CompletedAt:         row.CompletedAt,
				TimeToFirstTokenMs:  row.TimeToFirstTokenMs,
				TotalLatencyMs:      row.TotalLatencyMs,
			})
		}
		return nil
	})
	return llmRequests, err
}

func (r *llmRequestRepo) GetRequests(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter, limit, offset int32) ([]domain.LlmRequest, error) {
	var llmRequests []domain.LlmRequest
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageRequests(ctx, db.GetUsageRequestsParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			UserID:         filter.UserID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
			LimitCount:     limit,
			OffsetCount:    offset,
		})
		if err != nil {
			return err
		}
		llmRequests = make([]domain.LlmRequest, 0, len(rows))
		for _, row := range rows {
			llmRequests = append(llmRequests, domain.LlmRequest{
				ID:                  row.ID,
				OrganizationID:      orgID,
				ProjectID:           row.ProjectID,
				UserID:              row.UserID,
				ServiceAccountID:    row.ServiceAccountID,
				TeamID:              row.TeamID,
				GatewayCredentialID: row.GatewayCredentialID,
				RequestID:           row.RequestID,
				ProviderKind:        row.ProviderKind,
				Model:               row.Model,
				Status:              row.Status,
				ErrorCode:           row.ErrorCode,
				PromptTokens:        row.PromptTokens,
				CompletionTokens:    row.CompletionTokens,
				CachedInputTokens:   row.CachedInputTokens,
				ReasoningTokens:     row.ReasoningTokens,
				ToolTokens:          row.ToolTokens,
				TotalTokens:         row.TotalTokens,
				LatencyMs:           row.LatencyMs,
				CostMicros:          row.CostMicros,
				PricingVersion:      row.PricingVersion,
				CostStatus:          row.CostStatus,
				PolicyDecision:      row.PolicyDecision,
				DenialReason:        row.DenialReason,
				CreatedAt:           row.CreatedAt,
				RequestStartedAt:    row.RequestStartedAt,
				FirstTokenAt:        row.FirstTokenAt,
				CompletedAt:         row.CompletedAt,
				TimeToFirstTokenMs:  row.TimeToFirstTokenMs,
				TotalLatencyMs:      row.TotalLatencyMs,
			})
		}
		return nil
	})
	return llmRequests, err
}

func (r *llmRequestRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.PurgeExpiredLlmRequests(ctx, db.PurgeExpiredLlmRequestsParams{
			OrganizationID: orgID,
			OlderThan:      olderThan,
		})
		count = n
		return err
	})
	return count, err
}

type usageEventRepo struct {
	store *postgres.Store
}

// NewUsageEventRepository returns a UsageEventRepository backed by the store.
// All operations are tenant-scoped via the orgID context.
func NewUsageEventRepository(store *postgres.Store) ports.UsageEventRepository {
	return &usageEventRepo{store: store}
}

func (r *usageEventRepo) Record(ctx context.Context, orgID uuid.UUID, input domain.NewUsageEvent) (domain.UsageEvent, error) {
	var usageEvent domain.UsageEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RecordUsageEvent(ctx, db.RecordUsageEventParams{
			OrganizationID: orgID,
			RequestID:      input.RequestID,
			EventType:      input.EventType,
			Quantity:       input.Quantity,
			Unit:           input.Unit,
			Column6:        jsonObject(input.Metadata),
		})
		if err != nil {
			return err
		}
		usageEvent = toUsageEvent(row)
		return nil
	})
	return usageEvent, mapConflict(err)
}

func (r *usageEventRepo) Get(ctx context.Context, orgID uuid.UUID, requestID, eventType string) (domain.UsageEvent, error) {
	var usageEvent domain.UsageEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetUsageEvent(ctx, db.GetUsageEventParams{
			OrganizationID: orgID,
			RequestID:      requestID,
			EventType:      eventType,
		})
		if err != nil {
			return err
		}
		usageEvent = toUsageEvent(row)
		return nil
	})
	return usageEvent, mapNotFound(err)
}

func (r *usageEventRepo) ListByRequest(ctx context.Context, orgID uuid.UUID, requestID string) ([]domain.UsageEvent, error) {
	var usageEvents []domain.UsageEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListUsageEventsByRequest(ctx, db.ListUsageEventsByRequestParams{
			OrganizationID: orgID,
			RequestID:      requestID,
		})
		if err != nil {
			return err
		}
		usageEvents = make([]domain.UsageEvent, 0, len(rows))
		for _, row := range rows {
			usageEvents = append(usageEvents, toUsageEvent(row))
		}
		return nil
	})
	return usageEvents, err
}

func (r *usageEventRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.PurgeExpiredUsageEvents(ctx, db.PurgeExpiredUsageEventsParams{
			OrganizationID: orgID,
			OlderThan:      olderThan,
		})
		if err != nil {
			return err
		}
		count = n
		return nil
	})
	return count, err
}

type usageRollupRepo struct {
	store *postgres.Store
}

// NewUsageRollupRepository returns a UsageRollupRepository backed by the
// store. Every rebuild runs in one tenant transaction, so a failure cannot
// leave a window cleared but not refilled.
func NewUsageRollupRepository(store *postgres.Store) ports.UsageRollupRepository {
	return &usageRollupRepo{store: store}
}

func (r *usageRollupRepo) Recompute(ctx context.Context, orgID uuid.UUID, from, to time.Time) error {
	from, to = from.UTC(), to.UTC()
	hourFrom, hourTo := floorHour(from), nextHour(to)
	dayFrom, dayTo := floorDay(from), nextDay(to)
	monthFrom, monthTo := floorMonth(from), nextMonth(to)

	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		steps := []func(context.Context) error{
			func(c context.Context) error {
				return q.BackfillLLMRequestCost(c, db.BackfillLLMRequestCostParams{
					OrganizationID: orgID, FromTs: monthFrom, ToTs: monthTo,
				})
			},
			func(c context.Context) error {
				return q.DeleteUsageHourlyInWindow(c, db.DeleteUsageHourlyInWindowParams{
					OrganizationID: orgID, FromTs: hourFrom, ToTs: hourTo,
				})
			},
			func(c context.Context) error {
				return q.DeleteUsageDailyInWindow(c, db.DeleteUsageDailyInWindowParams{
					OrganizationID: orgID, FromTs: dayFrom, ToTs: dayTo,
				})
			},
			func(c context.Context) error {
				return q.DeleteUsageMonthlyInWindow(c, db.DeleteUsageMonthlyInWindowParams{
					OrganizationID: orgID, FromTs: monthFrom, ToTs: monthTo,
				})
			},
			func(c context.Context) error {
				return q.RecomputeUsageHourly(c, db.RecomputeUsageHourlyParams{
					OrganizationID: orgID, FromTs: hourFrom, ToTs: hourTo,
				})
			},
			func(c context.Context) error {
				return q.RecomputeUsageDaily(c, db.RecomputeUsageDailyParams{
					OrganizationID: orgID, FromTs: dayFrom, ToTs: dayTo,
				})
			},
			func(c context.Context) error {
				return q.RecomputeUsageMonthly(c, db.RecomputeUsageMonthlyParams{
					OrganizationID: orgID, FromTs: monthFrom, ToTs: monthTo,
				})
			},
		}
		for _, step := range steps {
			if err := step(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

// avgLatencyPtr reports the average as absent when the window held no request,
// so an empty period is never rendered as a measured zero-millisecond average.
func avgLatencyPtr(requests int64, avg float64) *float64 {
	if requests == 0 {
		return nil
	}
	return &avg
}

// latencyMetrics converts the SQL coalesce sentinel into absent metrics when
// the sample count is zero. Postgres cannot express percentile_cont's
// nullability to sqlc, so the sentinel stops here and never reaches the wire.
func latencyMetrics(samples int64, avg, p50, p95, p99 float64) (avgOut, p50Out, p95Out, p99Out *float64) {
	if samples == 0 {
		return nil, nil, nil, nil
	}
	return &avg, &p50, &p95, &p99
}

func (r *usageRollupRepo) GetSummary(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageSummary, error) {
	var summary domain.UsageSummary
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetUsageSummary(ctx, db.GetUsageSummaryParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			UserID:         filter.UserID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		summary = domain.UsageSummary{
			Requests:        row.TotalRequests,
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			CachedTokens:    row.CachedTokens,
			ReasoningTokens: row.ReasoningTokens,
			ToolTokens:      row.ToolTokens,
			TotalTokens:     row.TotalTokens,
			TotalCostMicros: row.TotalCostMicros,
			Errors:          row.Errors,
			AvgLatencyMs:    avgLatencyPtr(row.TotalRequests, row.AvgLatencyMs),
			ActiveUsers:     row.ActiveUsers,
			LastActivityAt:  row.LastActivityAt,
		}
		return nil
	})
	return summary, err
}

func (r *usageRollupRepo) GetTimeSeries(ctx context.Context, orgID uuid.UUID, from, to time.Time, projectID *uuid.UUID) ([]domain.UsageTimeSeriesPoint, error) {
	var points []domain.UsageTimeSeriesPoint
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageTimeSeries(ctx, db.GetUsageTimeSeriesParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
			ProjectID:      projectID,
		})
		if err != nil {
			return err
		}
		dataMap := make(map[string]domain.UsageTimeSeriesPoint, len(rows))
		for _, row := range rows {
			dataMap[row.BucketStart.UTC().Format("2006-01-02")] = domain.UsageTimeSeriesPoint{
				BucketStart: row.BucketStart.UTC(),
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
			}
		}

		// Zero-fill daily buckets from 'from' to 'to'
		cur := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
		end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
		if cur.After(end) {
			cur = end
		}
		if end.Sub(cur) < 6*24*time.Hour {
			cur = end.AddDate(0, 0, -6)
		}
		for !cur.After(end) {
			key := cur.Format("2006-01-02")
			if pt, ok := dataMap[key]; ok {
				points = append(points, pt)
			} else {
				points = append(points, domain.UsageTimeSeriesPoint{
					BucketStart: cur,
					Requests:    0,
					TotalTokens: 0,
					CostMicros:  0,
				})
			}
			cur = cur.AddDate(0, 0, 1)
		}
		return nil
	})
	return points, err
}

func (r *usageRollupRepo) GetByProject(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByProject, error) {
	var results []domain.UsageByProject
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByProject(ctx, db.GetUsageByProjectParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			UserID:         filter.UserID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByProject, 0, len(rows))
		for _, row := range rows {
			var projectID uuid.UUID
			if row.ProjectID != nil {
				projectID = *row.ProjectID
			}
			results = append(results, domain.UsageByProject{
				ProjectID: projectID,
				Name:      row.Name,
				UsageTotals: domain.UsageTotals{
					Requests:       row.Requests,
					InputTokens:    row.InputTokens,
					OutputTokens:   row.OutputTokens,
					TotalTokens:    row.TotalTokens,
					CostMicros:     row.CostMicros,
					Errors:         row.Errors,
					AvgLatencyMs:   avgLatencyPtr(row.Requests, row.AvgLatencyMs),
					LastActivityAt: row.LastActivityAt,
				},
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetByModel(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByModel, error) {
	var results []domain.UsageByModel
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByModel(ctx, db.GetUsageByModelParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByModel, 0, len(rows))
		for _, row := range rows {
			results = append(results, domain.UsageByModel{
				Model:       row.Model,
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetByProvider(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByProvider, error) {
	var results []domain.UsageByProvider
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByProvider(ctx, db.GetUsageByProviderParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByProvider, 0, len(rows))
		for _, row := range rows {
			results = append(results, domain.UsageByProvider{
				Provider:    row.Provider,
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetByUser(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByUser, error) {
	var results []domain.UsageByUser
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByUser(ctx, db.GetUsageByUserParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByUser, 0, len(rows))
		for _, row := range rows {
			var uid uuid.UUID
			if row.UserID != nil {
				uid = *row.UserID
			}
			results = append(results, domain.UsageByUser{
				UserID:      uid,
				Email:       row.Email,
				DisplayName: row.DisplayName,
				UsageTotals: domain.UsageTotals{
					Requests:       row.Requests,
					InputTokens:    row.InputTokens,
					OutputTokens:   row.OutputTokens,
					TotalTokens:    row.TotalTokens,
					CostMicros:     row.CostMicros,
					Errors:         row.Errors,
					AvgLatencyMs:   avgLatencyPtr(row.Requests, row.AvgLatencyMs),
					LastActivityAt: row.LastActivityAt,
				},
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetByTeam(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByTeam, error) {
	var results []domain.UsageByTeam
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByTeam(ctx, db.GetUsageByTeamParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			UserID:         filter.UserID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByTeam, 0, len(rows))
		for _, row := range rows {
			results = append(results, domain.UsageByTeam{
				TeamID: *row.TeamID,
				Name:   row.Name,
				UsageTotals: domain.UsageTotals{
					Requests:       int64(row.Requests),
					InputTokens:    row.InputTokens,
					OutputTokens:   row.OutputTokens,
					TotalTokens:    row.TotalTokens,
					CostMicros:     row.CostMicros,
					Errors:         row.Errors,
					AvgLatencyMs:   avgLatencyPtr(int64(row.Requests), row.AvgLatencyMs),
					LastActivityAt: row.LastActivityAt,
				},
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetLatency(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) (domain.UsageLatency, error) {
	var latency domain.UsageLatency
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetUsageLatencyOverall(ctx, db.GetUsageLatencyOverallParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			UserID:         filter.UserID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		avgTTFT, p50TTFT, p95TTFT, p99TTFT := latencyMetrics(row.TtftSamples, row.AvgTtftMs, row.P50TtftMs, row.P95TtftMs, row.P99TtftMs)
		avgTotal, p50Total, p95Total, p99Total := latencyMetrics(row.TotalSamples, row.AvgTotalMs, row.P50TotalMs, row.P95TotalMs, row.P99TotalMs)
		latency = domain.UsageLatency{
			TtftSamples:  row.TtftSamples,
			TotalSamples: row.TotalSamples,
			AvgTTFTMs:    avgTTFT,
			P50TTFTMs:    p50TTFT,
			P95TTFTMs:    p95TTFT,
			P99TTFTMs:    p99TTFT,
			AvgTotalMs:   avgTotal,
			P50TotalMs:   p50Total,
			P95TotalMs:   p95Total,
			P99TotalMs:   p99Total,
		}
		return nil
	})
	return latency, err
}

func (r *usageRollupRepo) GetLatencyByProviderModel(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageLatency, error) {
	var results []domain.UsageLatency
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageLatencyByProviderModel(ctx, db.GetUsageLatencyByProviderModelParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			UserID:         filter.UserID,
			TeamID:         filter.TeamID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageLatency, 0, len(rows))
		for _, row := range rows {
			avgTTFT, p50TTFT, p95TTFT, p99TTFT := latencyMetrics(row.TtftSamples, row.AvgTtftMs, row.P50TtftMs, row.P95TtftMs, row.P99TtftMs)
			avgTotal, p50Total, p95Total, p99Total := latencyMetrics(row.TotalSamples, row.AvgTotalMs, row.P50TotalMs, row.P95TotalMs, row.P99TotalMs)
			results = append(results, domain.UsageLatency{
				Provider:     &row.Provider,
				Model:        &row.Model,
				TtftSamples:  row.TtftSamples,
				TotalSamples: row.TotalSamples,
				AvgTTFTMs:    avgTTFT,
				P50TTFTMs:    p50TTFT,
				P95TTFTMs:    p95TTFT,
				P99TTFTMs:    p99TTFT,
				AvgTotalMs:   avgTotal,
				P50TotalMs:   p50Total,
				P95TotalMs:   p95Total,
				P99TotalMs:   p99Total,
			})
		}
		return nil
	})
	return results, err
}

func floorHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
}

func nextHour(t time.Time) time.Time {
	return floorHour(t).Add(time.Hour)
}

func floorDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func nextDay(t time.Time) time.Time {
	return floorDay(t).AddDate(0, 0, 1)
}

func floorMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func nextMonth(t time.Time) time.Time {
	return floorMonth(t).AddDate(0, 1, 0)
}

type idempotencyRepo struct {
	store *postgres.Store
}

// NewIdempotencyRepository returns an IdempotencyRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewIdempotencyRepository(store *postgres.Store) ports.IdempotencyRepository {
	return &idempotencyRepo{store: store}
}

func (r *idempotencyRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewIdempotencyKey) (domain.IdempotencyKey, error) {
	var key domain.IdempotencyKey
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateIdempotencyKey(ctx, db.CreateIdempotencyKeyParams{
			OrganizationID: orgID,
			ScopeKey:       input.ScopeKey,
			IdempotencyKey: input.Key,
			ExpiresAt:      input.ExpiresAt,
		})
		if err != nil {
			return err
		}
		key = toIdempotencyKey(row)
		return nil
	})
	return key, mapConflict(err)
}

func (r *idempotencyRepo) Get(ctx context.Context, orgID uuid.UUID, scopeKey, key string) (domain.IdempotencyKey, error) {
	var idempotencyKey domain.IdempotencyKey
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetIdempotencyKey(ctx, db.GetIdempotencyKeyParams{
			OrganizationID: orgID,
			ScopeKey:       scopeKey,
			IdempotencyKey: key,
		})
		if err != nil {
			return err
		}
		idempotencyKey = toIdempotencyKey(row)
		return nil
	})
	return idempotencyKey, mapNotFound(err)
}

type planRepo struct {
	store *postgres.Store
}

// NewPlanRepository returns a PlanRepository backed by the store. Plans are a
// global table, so operations run outside any tenant context.
func NewPlanRepository(store *postgres.Store) ports.PlanRepository {
	return &planRepo{store: store}
}

func (r *planRepo) Create(ctx context.Context, input domain.NewPlan) (domain.Plan, error) {
	var plan domain.Plan
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		row, err := q.CreatePlan(ctx, db.CreatePlanParams{
			Code:        input.Code,
			Name:        input.Name,
			Description: input.Description,
			Status:      domain.PlanStatusActive,
		})
		if err != nil {
			return err
		}
		plan = toPlan(row)
		return nil
	})
	return plan, err
}

func (r *planRepo) List(ctx context.Context) ([]domain.Plan, error) {
	var plans []domain.Plan
	err := r.store.WithTx(ctx, func(q *db.Queries) error {
		rows, err := q.ListPlans(ctx)
		if err != nil {
			return err
		}
		plans = make([]domain.Plan, 0, len(rows))
		for _, row := range rows {
			plans = append(plans, toPlan(row))
		}
		return nil
	})
	return plans, err
}

type subscriptionRepo struct {
	store *postgres.Store
}

// NewSubscriptionRepository returns a SubscriptionRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewSubscriptionRepository(store *postgres.Store) ports.SubscriptionRepository {
	return &subscriptionRepo{store: store}
}

func (r *subscriptionRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewSubscription) (domain.Subscription, error) {
	var subscription domain.Subscription
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateSubscription(ctx, db.CreateSubscriptionParams{
			OrganizationID:     orgID,
			PlanID:             input.PlanID,
			Status:             domain.SubscriptionStatusActive,
			CurrentPeriodStart: input.CurrentPeriodStart,
			CurrentPeriodEnd:   input.CurrentPeriodEnd,
			CancelAtPeriodEnd:  input.CancelAtPeriodEnd,
		})
		if err != nil {
			return err
		}
		subscription = toSubscription(row)
		return nil
	})
	return subscription, err
}

func (r *subscriptionRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Subscription, error) {
	var subscription domain.Subscription
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetSubscription(ctx, db.GetSubscriptionParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		subscription = toSubscription(row)
		return nil
	})
	return subscription, mapNotFound(err)
}

func (r *subscriptionRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Subscription, error) {
	var subscriptions []domain.Subscription
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListSubscriptions(ctx, orgID)
		if err != nil {
			return err
		}
		subscriptions = make([]domain.Subscription, 0, len(rows))
		for _, row := range rows {
			subscriptions = append(subscriptions, toSubscription(row))
		}
		return nil
	})
	return subscriptions, err
}

func (r *subscriptionRepo) ActiveWithPlan(ctx context.Context, orgID uuid.UUID) (domain.OrganizationPlan, error) {
	var plan domain.OrganizationPlan
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetActiveSubscriptionWithPlan(ctx, orgID)
		if err != nil {
			return err
		}
		plan = domain.OrganizationPlan{
			PlanCode:           row.PlanCode,
			PlanName:           row.PlanName,
			Status:             row.SubscriptionStatus,
			CurrentPeriodStart: row.CurrentPeriodStart,
			CurrentPeriodEnd:   row.CurrentPeriodEnd,
			CancelAtPeriodEnd:  row.CancelAtPeriodEnd,
		}
		return nil
	})
	return plan, mapNotFound(err)
}

func (r *subscriptionRepo) SetPlan(ctx context.Context, orgID, planID uuid.UUID) error {
	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		tag, err := r.store.Pool().Exec(ctx, `
			UPDATE subscriptions
			SET plan_id = $1, status = 'active', cancel_at_period_end = false, updated_at = now()
			WHERE organization_id = $2 AND status IN ('active', 'trialing')
		`, planID, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			now := time.Now().UTC()
			end := now.AddDate(0, 1, 0)
			_, err = q.CreateSubscription(ctx, db.CreateSubscriptionParams{
				OrganizationID:     orgID,
				PlanID:             planID,
				Status:             "active",
				CurrentPeriodStart: &now,
				CurrentPeriodEnd:   &end,
				CancelAtPeriodEnd:  false,
			})
			return err
		}
		return nil
	})
}

func (r *subscriptionRepo) Cancel(ctx context.Context, orgID uuid.UUID) error {
	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		_, err := r.store.Pool().Exec(ctx, `
			UPDATE subscriptions
			SET cancel_at_period_end = true, updated_at = now()
			WHERE organization_id = $1 AND status IN ('active', 'trialing')
		`, orgID)
		return err
	})
}

type billingEventRepo struct {
	store *postgres.Store
}

// NewBillingEventRepository returns a BillingEventRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewBillingEventRepository(store *postgres.Store) ports.BillingEventRepository {
	return &billingEventRepo{store: store}
}

func (r *billingEventRepo) Record(ctx context.Context, orgID uuid.UUID, input domain.NewBillingEvent) (domain.BillingEvent, error) {
	var billingEvent domain.BillingEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RecordBillingEvent(ctx, db.RecordBillingEventParams{
			OrganizationID: orgID,
			BillingEventID: input.BillingEventID,
			EventType:      input.EventType,
			AmountMicros:   input.AmountMicros,
			Currency:       input.Currency,
			Status:         domain.BillingEventStatusPending,
			Column7:        jsonObject(input.Payload),
		})
		if err != nil {
			return err
		}
		billingEvent = toBillingEvent(row)
		return nil
	})
	return billingEvent, mapConflict(err)
}

func (r *billingEventRepo) Get(ctx context.Context, orgID uuid.UUID, billingEventID string) (domain.BillingEvent, error) {
	var billingEvent domain.BillingEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetBillingEvent(ctx, db.GetBillingEventParams{
			OrganizationID: orgID,
			BillingEventID: billingEventID,
		})
		if err != nil {
			return err
		}
		billingEvent = toBillingEvent(row)
		return nil
	})
	return billingEvent, mapNotFound(err)
}

func (r *billingEventRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.PurgeExpiredBillingEvents(ctx, db.PurgeExpiredBillingEventsParams{
			OrganizationID: orgID,
			OlderThan:      olderThan,
		})
		count = n
		return err
	})
	return count, err
}

type invoiceRepo struct {
	store *postgres.Store
}

// NewInvoiceRepository returns an InvoiceRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewInvoiceRepository(store *postgres.Store) ports.InvoiceRepository {
	return &invoiceRepo{store: store}
}

func (r *invoiceRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Invoice, error) {
	var invoices []domain.Invoice
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListInvoices(ctx, orgID)
		if err != nil {
			return err
		}
		invoices = make([]domain.Invoice, 0, len(rows))
		for _, row := range rows {
			invoice := toInvoice(row)
			lines, err := q.ListInvoiceLines(ctx, db.ListInvoiceLinesParams{
				OrganizationID: orgID,
				InvoiceID:      row.ID,
			})
			if err != nil {
				return err
			}
			invoice.Lines = make([]domain.InvoiceLine, 0, len(lines))
			for _, l := range lines {
				invoice.Lines = append(invoice.Lines, toInvoiceLine(l))
			}
			invoices = append(invoices, invoice)
		}
		return nil
	})
	return invoices, err
}

func (r *invoiceRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Invoice, error) {
	var invoice domain.Invoice
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetInvoice(ctx, db.GetInvoiceParams{OrganizationID: orgID, ID: id})
		if err != nil {
			return err
		}
		invoice = toInvoice(row)
		lines, err := q.ListInvoiceLines(ctx, db.ListInvoiceLinesParams{OrganizationID: orgID, InvoiceID: id})
		if err != nil {
			return err
		}
		invoice.Lines = make([]domain.InvoiceLine, 0, len(lines))
		for _, l := range lines {
			invoice.Lines = append(invoice.Lines, toInvoiceLine(l))
		}
		return nil
	})
	return invoice, mapNotFound(err)
}

type pricingModelRepo struct {
	store *postgres.Store
}

// NewPricingModelRepository returns a PricingModelRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewPricingModelRepository(store *postgres.Store) ports.PricingModelRepository {
	return &pricingModelRepo{store: store}
}

func (r *pricingModelRepo) Upsert(ctx context.Context, orgID uuid.UUID, input domain.NewPricingModel) (domain.PricingModel, error) {
	var pricingModel domain.PricingModel
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.UpsertPricingModel(ctx, db.UpsertPricingModelParams{
			OrganizationID: orgID,
			ModelSlug:      input.ModelSlug,
			ProviderKind:   input.ProviderKind,
			DisplayName:    input.DisplayName,
			Currency:       input.Currency,
			Status:         domain.PricingModelStatusActive,
		})
		if err != nil {
			return err
		}
		pricingModel = toPricingModel(row)
		return nil
	})
	return pricingModel, err
}

func (r *pricingModelRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.PricingModel, error) {
	var pricingModels []domain.PricingModel
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListPricingModels(ctx, orgID)
		if err != nil {
			return err
		}
		pricingModels = make([]domain.PricingModel, 0, len(rows))
		for _, row := range rows {
			pricingModels = append(pricingModels, toPricingModel(row))
		}
		return nil
	})
	return pricingModels, err
}

type pricingVersionRepo struct {
	store *postgres.Store
}

// NewPricingVersionRepository returns a PricingVersionRepository backed by the
// store. All operations are tenant-scoped via the orgID context.
func NewPricingVersionRepository(store *postgres.Store) ports.PricingVersionRepository {
	return &pricingVersionRepo{store: store}
}

func (r *pricingVersionRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewPricingVersion) (domain.PricingVersion, error) {
	var pricingVersion domain.PricingVersion
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreatePricingVersion(ctx, db.CreatePricingVersionParams{
			PricingModelID:        input.PricingModelID,
			OrganizationID:        orgID,
			Version:               input.Version,
			InputUnitPriceMicros:  input.InputUnitPriceMicros,
			OutputUnitPriceMicros: input.OutputUnitPriceMicros,
			EffectiveFrom:         input.EffectiveFrom,
		})
		if err != nil {
			return err
		}
		pricingVersion = toPricingVersion(row)
		return nil
	})
	return pricingVersion, err
}

func (r *pricingVersionRepo) List(ctx context.Context, orgID, pricingModelID uuid.UUID) ([]domain.PricingVersion, error) {
	var pricingVersions []domain.PricingVersion
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListPricingVersions(ctx, db.ListPricingVersionsParams{
			OrganizationID: orgID,
			PricingModelID: pricingModelID,
		})
		if err != nil {
			return err
		}
		pricingVersions = make([]domain.PricingVersion, 0, len(rows))
		for _, row := range rows {
			pricingVersions = append(pricingVersions, toPricingVersion(row))
		}
		return nil
	})
	return pricingVersions, err
}

type alertRepo struct {
	store *postgres.Store
}

// NewAlertRepository returns an AlertRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewAlertRepository(store *postgres.Store) ports.AlertRepository {
	return &alertRepo{store: store}
}

func (r *alertRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.NewAlert) (domain.Alert, error) {
	var alert domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateAlert(ctx, db.CreateAlertParams{
			OrganizationID: orgID,
			AlertType:      input.AlertType,
			Name:           input.Name,
			Severity:       input.Severity,
			Column5:        jsonObject(input.Configuration),
			Status:         domain.AlertStatusActive,
		})
		if err != nil {
			return err
		}
		alert = toAlert(row)
		return nil
	})
	return alert, err
}

func (r *alertRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.Alert, error) {
	var alerts []domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListAlerts(ctx, orgID)
		if err != nil {
			return err
		}
		alerts = make([]domain.Alert, 0, len(rows))
		for _, row := range rows {
			alerts = append(alerts, toAlert(row))
		}
		return nil
	})
	return alerts, err
}

func (r *alertRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.Alert, error) {
	var alert domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetAlert(ctx, db.GetAlertParams{
			OrganizationID: orgID,
			ID:             id,
		})
		if err != nil {
			return err
		}
		alert = toAlert(row)
		return nil
	})
	return alert, mapNotFound(err)
}

func (r *alertRepo) Update(ctx context.Context, orgID, id uuid.UUID, input domain.AlertUpdate) (domain.Alert, error) {
	var alert domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.UpdateAlert(ctx, db.UpdateAlertParams{
			OrganizationID: orgID,
			ID:             id,
			Name:           input.Name,
			Severity:       input.Severity,
			Column5:        jsonObject(input.Configuration),
			Status:         input.Status,
		})
		if err != nil {
			return err
		}
		alert = toAlert(row)
		return nil
	})
	return alert, mapNotFound(err)
}

func (r *alertRepo) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.DeleteAlert(ctx, db.DeleteAlertParams{
			OrganizationID: orgID,
			ID:             id,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return ports.ErrNotFound
		}
		return nil
	})
}

func (r *alertRepo) UpdateTriggered(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Alert, error) {
	var alert domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.UpdateAlertTriggered(ctx, db.UpdateAlertTriggeredParams{
			OrganizationID: orgID,
			ID:             id,
			Status:         status,
		})
		if err != nil {
			return err
		}
		alert = toAlert(row)
		return nil
	})
	return alert, mapNotFound(err)
}

func (r *alertRepo) CreateEvent(ctx context.Context, orgID uuid.UUID, input domain.NewAlertEvent) (domain.AlertEvent, error) {
	var event domain.AlertEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateAlertEvent(ctx, db.CreateAlertEventParams{
			OrganizationID: orgID,
			AlertID:        input.AlertID,
			AlertType:      input.AlertType,
			Severity:       input.Severity,
			Status:         input.Status,
			Message:        input.Message,
			Column7:        jsonObject(input.Context),
			TriggeredAt:    input.TriggeredAt,
			ResolvedAt:     input.ResolvedAt,
		})
		if err != nil {
			return err
		}
		event = toAlertEvent(row)
		return nil
	})
	return event, err
}

func (r *alertRepo) ListEvents(ctx context.Context, orgID uuid.UUID, alertID *uuid.UUID, limit, offset int32) ([]domain.AlertEvent, error) {
	var events []domain.AlertEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListAlertEvents(ctx, db.ListAlertEventsParams{
			OrganizationID: orgID,
			AlertID:        alertID,
			Limit:          limit,
			Offset:         offset,
		})
		if err != nil {
			return err
		}
		events = make([]domain.AlertEvent, 0, len(rows))
		for _, row := range rows {
			events = append(events, toAlertEvent(row))
		}
		return nil
	})
	return events, err
}

func (r *alertRepo) GetLatestEvent(ctx context.Context, orgID, alertID uuid.UUID) (*domain.AlertEvent, error) {
	var event *domain.AlertEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetLatestAlertEvent(ctx, db.GetLatestAlertEventParams{
			OrganizationID: orgID,
			AlertID:        alertID,
		})
		if err != nil {
			return err
		}
		e := toAlertEvent(row)
		event = &e
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return event, err
}

func (r *alertRepo) UpdateEventStatus(ctx context.Context, orgID, eventID uuid.UUID, status string) (domain.AlertEvent, error) {
	var event domain.AlertEvent
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.UpdateAlertEventStatus(ctx, db.UpdateAlertEventStatusParams{
			OrganizationID: orgID,
			ID:             eventID,
			Status:         status,
		})
		if err != nil {
			return err
		}
		event = toAlertEvent(row)
		return nil
	})
	return event, mapNotFound(err)
}

func (r *alertRepo) CreateNotification(ctx context.Context, orgID uuid.UUID, input domain.NewInAppNotification) (domain.InAppNotification, error) {
	var notification domain.InAppNotification
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.CreateInAppNotification(ctx, db.CreateInAppNotificationParams{
			OrganizationID: orgID,
			AlertID:        input.AlertID,
			AlertEventID:   input.AlertEventID,
			Title:          input.Title,
			Message:        input.Message,
			Severity:       input.Severity,
		})
		if err != nil {
			return err
		}
		notification = toInAppNotification(row)
		return nil
	})
	return notification, err
}

func (r *alertRepo) ListNotifications(ctx context.Context, orgID uuid.UUID, unreadOnly bool, limit, offset int32) ([]domain.InAppNotification, error) {
	var notifications []domain.InAppNotification
	var unreadPtr *bool
	if unreadOnly {
		u := true
		unreadPtr = &u
	}
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListInAppNotifications(ctx, db.ListInAppNotificationsParams{
			OrganizationID: orgID,
			Limit:          limit,
			Offset:         offset,
			UnreadOnly:     unreadPtr,
		})
		if err != nil {
			return err
		}
		notifications = make([]domain.InAppNotification, 0, len(rows))
		for _, row := range rows {
			notifications = append(notifications, toInAppNotification(row))
		}
		return nil
	})
	return notifications, err
}

func (r *alertRepo) MarkNotificationRead(ctx context.Context, orgID, id uuid.UUID) error {
	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		_, err := q.MarkNotificationRead(ctx, db.MarkNotificationReadParams{
			OrganizationID: orgID,
			ID:             id,
		})
		return err
	})
}

func (r *alertRepo) MarkAllNotificationsRead(ctx context.Context, orgID uuid.UUID) error {
	return r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		_, err := q.MarkAllNotificationsRead(ctx, orgID)
		return err
	})
}

type auditLogRepo struct {
	store *postgres.Store
}


// NewAuditLogRepository returns an AuditLogRepository backed by the store. All
// operations are tenant-scoped via the orgID context.
func NewAuditLogRepository(store *postgres.Store) ports.AuditLogRepository {
	return &auditLogRepo{store: store}
}

func (r *auditLogRepo) Append(ctx context.Context, orgID uuid.UUID, input domain.NewAuditLog) (domain.AuditLog, error) {
	var auditLog domain.AuditLog
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.AppendAuditLog(ctx, db.AppendAuditLogParams{
			OrganizationID:        orgID,
			ActorUserID:           input.ActorUserID,
			ActorServiceAccountID: input.ActorServiceAccountID,
			Action:                input.Action,
			ResourceType:          input.ResourceType,
			ResourceID:            input.ResourceID,
			Column7:               input.Before,
			Column8:               input.After,
			IpAddress:             input.IPAddress,
			RequestID:             input.RequestID,
		})
		if err != nil {
			return err
		}
		auditLog = toAuditLog(row)
		return nil
	})
	return auditLog, err
}

func (r *auditLogRepo) List(ctx context.Context, orgID uuid.UUID, filter domain.AuditLogFilter, limit, offset int32) ([]domain.AuditLog, error) {
	var auditLogs []domain.AuditLog
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListAuditLogs(ctx, db.ListAuditLogsParams{
			OrganizationID: orgID,
			ActorID:        filter.ActorID,
			ActorType:      filter.ActorType,
			Action:         filter.Action,
			ResourceType:   filter.ResourceType,
			ResourceID:     filter.ResourceID,
			FromTs:         timeOrNil(filter.From),
			ToTs:           timeOrNil(filter.To),
			LimitCount:     limit,
			OffsetCount:    offset,
		})
		if err != nil {
			return err
		}
		auditLogs = make([]domain.AuditLog, 0, len(rows))
		for _, row := range rows {
			auditLogs = append(auditLogs, toAuditLog(row))
		}
		return nil
	})
	return auditLogs, err
}

func (r *auditLogRepo) Get(ctx context.Context, orgID, id uuid.UUID) (domain.AuditLog, error) {
	var auditLog domain.AuditLog
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetAuditLog(ctx, db.GetAuditLogParams{
			OrganizationID: orgID,
			ID:             id,
		})
		if err != nil {
			return err
		}
		auditLog = toAuditLog(row)
		return nil
	})
	return auditLog, mapNotFound(err)
}

func (r *auditLogRepo) PurgeExpired(ctx context.Context, orgID uuid.UUID, olderThan time.Time) (int64, error) {
	var count int64
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		n, err := q.PurgeExpiredAuditLogs(ctx, db.PurgeExpiredAuditLogsParams{
			OrganizationID: orgID,
			OlderThan:      olderThan,
		})
		count = n
		return err
	})
	return count, err
}

func toInAppNotification(row db.InAppNotification) domain.InAppNotification {
	return domain.InAppNotification{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		AlertID:        row.AlertID,
		AlertEventID:   row.AlertEventID,
		Title:          row.Title,
		Message:        row.Message,
		Severity:       row.Severity,
		Read:           row.Read,
		CreatedAt:      row.CreatedAt,
	}
}

// jsonObject returns the JSON bytes to store in a jsonb column declared NOT
// NULL DEFAULT '{}'::jsonb. The domain "nil means default" contract is
// preserved by substituting an empty JSON object.
func jsonObject(b []byte) []byte {
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}

func toConversation(row db.Conversation) domain.Conversation {
	return domain.Conversation{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		UserID:         row.UserID,
		ProjectID:      row.ProjectID,
		Title:          row.Title,
		Status:         row.Status,
		Model:          row.Model,
		Provider:       row.Provider,
		PrivacyState:   row.PrivacyState,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toConversationMessage(row db.ConversationMessage) domain.ConversationMessage {
	return domain.ConversationMessage{
		ID:                row.ID,
		ConversationID:    row.ConversationID,
		OrganizationID:    row.OrganizationID,
		UserID:            row.UserID,
		Role:              row.Role,
		Content:           row.Content,
		Tokens:            row.Tokens,
		ToolName:          row.ToolName,
		ProviderMessageID: row.ProviderMessageID,
		CreatedAt:         row.CreatedAt,
	}
}

func toConversationSummary(row db.ListConversationsRow) domain.ConversationSummary {
	summary := domain.ConversationSummary{
		ID:           row.ID,
		Title:        row.Title,
		UserID:       row.UserID,
		ProjectID:    row.ProjectID,
		Model:        row.Model,
		Provider:     row.Provider,
		StartedAt:    row.CreatedAt,
		RequestCount: int32(row.RequestCount),
		MessageCount: int32(row.MessageCount),
		TotalTokens:  int32(row.TotalTokens),
		PrivacyState: row.PrivacyState,
	}
	// SQL replaced an empty MAX() with a zero-time sentinel, so the absence of
	// messages is only recoverable from the count.
	if row.MessageCount > 0 {
		last := row.LastActivityAt
		summary.LastActivityAt = &last
	}
	return summary
}

func toLlmRequest(row db.LlmRequest) domain.LlmRequest {
	return domain.LlmRequest{
		ID:                  row.ID,
		OrganizationID:      row.OrganizationID,
		ProjectID:           row.ProjectID,
		UserID:              row.UserID,
		ServiceAccountID:    row.ServiceAccountID,
		TeamID:              row.TeamID,
		GatewayCredentialID: row.GatewayCredentialID,
		RequestID:           row.RequestID,
		ProviderRequestID:   row.ProviderRequestID,
		ProviderKind:        row.ProviderKind,
		Model:               row.Model,
		Status:              row.Status,
		ErrorCode:           row.ErrorCode,
		PromptTokens:        row.PromptTokens,
		CompletionTokens:    row.CompletionTokens,
		CachedInputTokens:   row.CachedInputTokens,
		ReasoningTokens:     row.ReasoningTokens,
		ToolTokens:          row.ToolTokens,
		TotalTokens:         row.TotalTokens,
		LatencyMs:           row.LatencyMs,
		CostMicros:          row.CostMicros,
		PricingVersion:      row.PricingVersion,
		CostStatus:          row.CostStatus,
		PolicyDecision:      row.PolicyDecision,
		DenialReason:        row.DenialReason,
		CreatedAt:           row.CreatedAt,
		RequestStartedAt:    row.RequestStartedAt,
		FirstTokenAt:        row.FirstTokenAt,
		CompletedAt:         row.CompletedAt,
		TimeToFirstTokenMs:  row.TimeToFirstTokenMs,
		TotalLatencyMs:      row.TotalLatencyMs,
	}
}

func toUsageEvent(row db.UsageEvent) domain.UsageEvent {
	return domain.UsageEvent{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		RequestID:      row.RequestID,
		EventType:      row.EventType,
		Quantity:       row.Quantity,
		Unit:           row.Unit,
		Metadata:       row.Metadata,
		CreatedAt:      row.CreatedAt,
	}
}

func toIdempotencyKey(row db.IdempotencyKey) domain.IdempotencyKey {
	return domain.IdempotencyKey{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ScopeKey:       row.ScopeKey,
		Key:            row.IdempotencyKey,
		ExpiresAt:      row.ExpiresAt,
		CreatedAt:      row.CreatedAt,
	}
}

func toPlan(row db.Plan) domain.Plan {
	return domain.Plan{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Description: row.Description,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func toSubscription(row db.Subscription) domain.Subscription {
	return domain.Subscription{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		PlanID:             row.PlanID,
		Status:             row.Status,
		CurrentPeriodStart: row.CurrentPeriodStart,
		CurrentPeriodEnd:   row.CurrentPeriodEnd,
		CancelAtPeriodEnd:  row.CancelAtPeriodEnd,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func toBillingEvent(row db.BillingEvent) domain.BillingEvent {
	return domain.BillingEvent{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		BillingEventID: row.BillingEventID,
		EventType:      row.EventType,
		AmountMicros:   row.AmountMicros,
		Currency:       row.Currency,
		Status:         row.Status,
		Payload:        row.Payload,
		CreatedAt:      row.CreatedAt,
	}
}

func toInvoice(row db.Invoice) domain.Invoice {
	return domain.Invoice{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		Number:         row.Number,
		Status:         row.Status,
		Currency:       row.Currency,
		TotalMicros:    row.TotalMicros,
		PeriodStart:    row.PeriodStart,
		PeriodEnd:      row.PeriodEnd,
		IssuedAt:       row.IssuedAt,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toInvoiceLine(row db.InvoiceLine) domain.InvoiceLine {
	return domain.InvoiceLine{
		ID:           row.ID,
		InvoiceID:    row.InvoiceID,
		Description:  row.Description,
		Quantity:     row.Quantity,
		UnitMicros:   row.UnitMicros,
		AmountMicros: row.AmountMicros,
		CreatedAt:    row.CreatedAt,
	}
}

func toPricingModel(row db.PricingModel) domain.PricingModel {
	return domain.PricingModel{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ModelSlug:      row.ModelSlug,
		ProviderKind:   row.ProviderKind,
		DisplayName:    row.DisplayName,
		Currency:       row.Currency,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func toPricingVersion(row db.PricingVersion) domain.PricingVersion {
	return domain.PricingVersion{
		ID:                    row.ID,
		PricingModelID:        row.PricingModelID,
		OrganizationID:        row.OrganizationID,
		Version:               row.Version,
		InputUnitPriceMicros:  row.InputUnitPriceMicros,
		OutputUnitPriceMicros: row.OutputUnitPriceMicros,
		EffectiveFrom:         row.EffectiveFrom,
		CreatedAt:             row.CreatedAt,
	}
}

func toAlert(row db.Alert) domain.Alert {
	return domain.Alert{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		AlertType:       row.AlertType,
		Name:            row.Name,
		Severity:        row.Severity,
		Configuration:   row.Configuration,
		Status:          row.Status,
		LastTriggeredAt: row.LastTriggeredAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func toAlertEvent(row db.AlertEvent) domain.AlertEvent {
	return domain.AlertEvent{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		AlertID:        row.AlertID,
		AlertType:      row.AlertType,
		Severity:       row.Severity,
		Status:         row.Status,
		Message:        row.Message,
		Context:        row.Context,
		TriggeredAt:    row.TriggeredAt,
		ResolvedAt:     row.ResolvedAt,
		CreatedAt:      row.CreatedAt,
	}
}


func toAuditLog(row db.AuditLog) domain.AuditLog {
	return domain.AuditLog{
		ID:                    row.ID,
		OrganizationID:        row.OrganizationID,
		ActorUserID:           row.ActorUserID,
		ActorServiceAccountID: row.ActorServiceAccountID,
		Action:                row.Action,
		ResourceType:          row.ResourceType,
		ResourceID:            row.ResourceID,
		Before:                row.Before,
		After:                 row.After,
		IPAddress:             row.IpAddress,
		RequestID:             row.RequestID,
		CreatedAt:             row.CreatedAt,
	}
}

func (r *usageRollupRepo) GetByServiceAccount(ctx context.Context, orgID uuid.UUID, filter domain.UsageFilter) ([]domain.UsageByServiceAccount, error) {
	var results []domain.UsageByServiceAccount
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByServiceAccount(ctx, db.GetUsageByServiceAccountParams{
			OrganizationID: orgID,
			FromTs:         filter.From.UTC(),
			ToTs:           filter.To.UTC(),
			ProjectID:      filter.ProjectID,
			Provider:       filter.Provider,
			Model:          filter.Model,
			Status:         filter.Status,
		})
		if err != nil {
			return err
		}
		results = make([]domain.UsageByServiceAccount, 0, len(rows))
		for _, row := range rows {
			var saID uuid.UUID
			if row.ServiceAccountID != nil {
				saID = *row.ServiceAccountID
			}
			results = append(results, domain.UsageByServiceAccount{
				ServiceAccountID: saID,
				Name:             row.Name,
				UsageTotals: domain.UsageTotals{
					Requests:       row.Requests,
					InputTokens:    row.InputTokens,
					OutputTokens:   row.OutputTokens,
					TotalTokens:    row.TotalTokens,
					CostMicros:     row.CostMicros,
					Errors:         row.Errors,
					AvgLatencyMs:   avgLatencyPtr(row.Requests, row.AvgLatencyMs),
					LastActivityAt: row.LastActivityAt,
				},
			})
		}
		return nil
	})
	return results, err
}
