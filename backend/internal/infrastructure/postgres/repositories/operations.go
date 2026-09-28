package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

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

func (r *conversationRepo) List(ctx context.Context, orgID, userID uuid.UUID) ([]domain.Conversation, error) {
	var conversations []domain.Conversation
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.ListConversations(ctx, db.ListConversationsParams{OrganizationID: orgID, UserID: userID})
		if err != nil {
			return err
		}
		conversations = make([]domain.Conversation, 0, len(rows))
		for _, row := range rows {
			conversations = append(conversations, toConversation(row))
		}
		return nil
	})
	return conversations, err
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
		row, err := q.CreateConversationMessage(ctx, db.CreateConversationMessageParams{
			ConversationID:    input.ConversationID,
			OrganizationID:    orgID,
			UserID:            input.UserID,
			Role:              input.Role,
			Content:           input.Content,
			Tokens:            input.Tokens,
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
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.RecordLLMRequest(ctx, db.RecordLLMRequestParams{
			OrganizationID:    orgID,
			ProjectID:         input.ProjectID,
			UserID:            input.UserID,
			ApiCredentialID:   input.APICredentialID,
			RequestID:         input.RequestID,
			ProviderRequestID: input.ProviderRequestID,
			ProviderKind:      input.ProviderKind,
			Model:             input.Model,
			Status:            input.Status,
			ErrorCode:         input.ErrorCode,
			PromptTokens:      input.PromptTokens,
			CompletionTokens:  input.CompletionTokens,
			CachedInputTokens: input.CachedInputTokens,
			TotalTokens:       input.TotalTokens,
			LatencyMs:         input.LatencyMs,
			CostMicros:        input.CostMicros,
			CreatedAt:         createdAt,
			RequestStartedAt:  input.RequestStartedAt,
			FirstTokenAt:      input.FirstTokenAt,
			CompletedAt:       input.CompletedAt,
			TimeToFirstTokenMs: input.TimeToFirstTokenMs,
			TotalLatencyMs:    input.TotalLatencyMs,
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
				ID:             row.ID,
				OrganizationID: orgID,
				ProjectID:      row.ProjectID,
				UserID:         row.UserID,
				RequestID:      row.RequestID,
				ProviderKind:   row.ProviderKind,
				Model:          row.Model,
				Status:         row.Status,
				TotalTokens:    int32(row.TotalTokens),
				LatencyMs:      int32(row.LatencyMs),
				CostMicros:     row.CostMicros,
				CreatedAt:      row.CreatedAt,
				RequestStartedAt:  row.RequestStartedAt,
				FirstTokenAt:      row.FirstTokenAt,
				CompletedAt:       row.CompletedAt,
				TimeToFirstTokenMs: row.TimeToFirstTokenMs,
				TotalLatencyMs:    row.TotalLatencyMs,
			})
		}
		return nil
	})
	return llmRequests, err
}

func (r *llmRequestRepo) GetRequests(ctx context.Context, orgID uuid.UUID, from, to time.Time, limit, offset int32) ([]domain.LlmRequest, error) {
	var llmRequests []domain.LlmRequest
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageRequests(ctx, db.GetUsageRequestsParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
			LimitCount:     limit,
			OffsetCount:    offset,
		})
		if err != nil {
			return err
		}
		llmRequests = make([]domain.LlmRequest, 0, len(rows))
		for _, row := range rows {
			llmRequests = append(llmRequests, domain.LlmRequest{
				ID:             row.ID,
				OrganizationID: orgID,
				ProjectID:      row.ProjectID,
				UserID:         row.UserID,
				RequestID:      row.RequestID,
				ProviderKind:   row.ProviderKind,
				Model:          row.Model,
				Status:         row.Status,
				TotalTokens:    int32(row.TotalTokens),
				LatencyMs:      int32(row.LatencyMs),
				CostMicros:     row.CostMicros,
				CreatedAt:      row.CreatedAt,
				RequestStartedAt:  row.RequestStartedAt,
				FirstTokenAt:      row.FirstTokenAt,
				CompletedAt:       row.CompletedAt,
				TimeToFirstTokenMs: row.TimeToFirstTokenMs,
				TotalLatencyMs:    row.TotalLatencyMs,
			})
		}
		return nil
	})
	return llmRequests, err
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

func (r *usageRollupRepo) GetSummary(ctx context.Context, orgID uuid.UUID, from, to time.Time, projectID *uuid.UUID, model *string) (domain.UsageSummary, error) {
	var summary domain.UsageSummary
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.GetUsageSummary(ctx, db.GetUsageSummaryParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
			ProjectID:      projectID,
			Model:          model,
		})
		if err != nil {
			return err
		}
		summary = domain.UsageSummary{
			Requests:        row.TotalRequests,
			InputTokens:     row.InputTokens,
			OutputTokens:    row.OutputTokens,
			CachedTokens:    row.CachedTokens,
			TotalTokens:     row.TotalTokens,
			TotalCostMicros: row.TotalCostMicros,
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
		for _, row := range rows {
			points = append(points, domain.UsageTimeSeriesPoint{
				BucketStart: row.BucketStart,
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
			})
		}
		return nil
	})
	return points, err
}

func (r *usageRollupRepo) GetByProject(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByProject, error) {
	var results []domain.UsageByProject
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByProject(ctx, db.GetUsageByProjectParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			results = append(results, domain.UsageByProject{
				ProjectID:   row.ProjectID,
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
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

func (r *usageRollupRepo) GetByUser(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByUser, error) {
	var results []domain.UsageByUser
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByUser(ctx, db.GetUsageByUserParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			var uid uuid.UUID
			if row.UserID != nil {
				uid = *row.UserID
			}
			results = append(results, domain.UsageByUser{
				UserID:      uid,
				Requests:    row.Requests,
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
			})
		}
		return nil
	})
	return results, err
}

func (r *usageRollupRepo) GetByTeam(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]domain.UsageByTeam, error) {
	var results []domain.UsageByTeam
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		rows, err := q.GetUsageByTeam(ctx, db.GetUsageByTeamParams{
			OrganizationID: orgID,
			FromTs:         from.UTC(),
			ToTs:           to.UTC(),
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			results = append(results, domain.UsageByTeam{
				TeamID:      row.TeamID,
				Requests:    int64(row.Requests),
				TotalTokens: row.TotalTokens,
				CostMicros:  row.CostMicros,
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

func (r *alertRepo) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) (domain.Alert, error) {
	var alert domain.Alert
	err := r.store.WithTenantTx(ctx, orgID, func(q *db.Queries) error {
		row, err := q.SetAlertStatus(ctx, db.SetAlertStatusParams{
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
		Title:          row.Title,
		Status:         row.Status,
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
		ProviderMessageID: row.ProviderMessageID,
		CreatedAt:         row.CreatedAt,
	}
}

func toLlmRequest(row db.LlmRequest) domain.LlmRequest {
	return domain.LlmRequest{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		ProjectID:         row.ProjectID,
		UserID:            row.UserID,
		APICredentialID:   row.ApiCredentialID,
		RequestID:         row.RequestID,
		ProviderRequestID: row.ProviderRequestID,
		ProviderKind:      row.ProviderKind,
		Model:             row.Model,
		Status:            row.Status,
		ErrorCode:         row.ErrorCode,
		PromptTokens:      row.PromptTokens,
		CompletionTokens:  row.CompletionTokens,
		CachedInputTokens: row.CachedInputTokens,
		TotalTokens:       row.TotalTokens,
		LatencyMs:         row.LatencyMs,
		CostMicros:        row.CostMicros,
		CreatedAt:         row.CreatedAt,
		RequestStartedAt:  row.RequestStartedAt,
		FirstTokenAt:      row.FirstTokenAt,
		CompletedAt:       row.CompletedAt,
		TimeToFirstTokenMs: row.TimeToFirstTokenMs,
		TotalLatencyMs:    row.TotalLatencyMs,
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
