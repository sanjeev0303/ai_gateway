package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

type EvaluatorDeps struct {
	Log            *slog.Logger
	Alerts         repositories.AlertRepository
	Notifier       Notifier
	UsageRollups   repositories.UsageRollupRepository
	ProviderHealth repositories.ProviderHealthRepository
	SecurityEvents repositories.SecurityEventRepository
	LlmRequests    repositories.LlmRequestRepository
}

type Evaluator struct {
	log            *slog.Logger
	alerts         repositories.AlertRepository
	notifier       Notifier
	usageRollups   repositories.UsageRollupRepository
	providerHealth repositories.ProviderHealthRepository
	securityEvents repositories.SecurityEventRepository
	llmRequests    repositories.LlmRequestRepository
}

func NewEvaluator(deps EvaluatorDeps) *Evaluator {
	if deps.Log == nil {
		deps.Log = slog.Default()
	}
	return &Evaluator{
		log:            deps.Log,
		alerts:         deps.Alerts,
		notifier:       deps.Notifier,
		usageRollups:   deps.UsageRollups,
		providerHealth: deps.ProviderHealth,
		securityEvents: deps.SecurityEvents,
		llmRequests:    deps.LlmRequests,
	}
}

// EvaluateAll scans all organizations for active alerts and triggers them if conditions are met.
// This is called periodically by the background worker.
func (e *Evaluator) EvaluateAll(ctx context.Context, orgIDs []uuid.UUID) {
	for _, orgID := range orgIDs {
		alerts, err := e.alerts.List(ctx, orgID)
		if err != nil {
			e.log.ErrorContext(ctx, "failed to list alerts for evaluation", "org_id", orgID, "error", err)
			continue
		}

		for _, a := range alerts {
			if a.Status == domain.AlertStatusMuted {
				continue
			}
			e.evaluateSingle(ctx, orgID, a)
		}
	}
}

// EvaluateAlert performs a deterministic evaluation of an alert rule's conditions.
func (e *Evaluator) EvaluateAlert(ctx context.Context, orgID uuid.UUID, a domain.Alert) (bool, string, map[string]any) {
	switch a.AlertType {
	case "budget_warning":
		return e.evalBudgetWarning(ctx, orgID, a.Configuration)
	case "budget_exhaustion":
		return e.evalBudgetExhaustion(ctx, orgID, a.Configuration)
	case "spend_threshold", "spend_exceeded", "spend_limit":
		return e.evalSpend(ctx, orgID, a.Configuration)
	case "spend_spike":
		return e.evalSpendSpike(ctx, orgID, a.Configuration)
	case "usage_spike":
		return e.evalUsageSpike(ctx, orgID, a.Configuration)
	case "request_rate_spike", "rate_limit":
		return e.evalRequestRate(ctx, orgID, a.Configuration)
	case "error_rate_spike", "error_rate":
		return e.evalErrorRate(ctx, orgID, a.Configuration)
	case "provider_failure":
		return e.evalProviderFailure(ctx, orgID, a.Configuration)
	case "credential_failure":
		return e.evalCredentialFailure(ctx, orgID, a.Configuration)
	case "policy_violation", "policy_denial_spike":
		return e.evalPolicyViolation(ctx, orgID, a.Configuration)
	case "rate_limit_spike":
		return e.evalRateLimitSpike(ctx, orgID, a.Configuration)
	case "usage_anomaly", "anomaly":
		return e.evalUsageAnomaly(ctx, orgID, a.Configuration)
	default:
		e.log.WarnContext(ctx, "unknown alert type", "alert_type", a.AlertType, "alert_id", a.ID)
		return false, fmt.Sprintf("unsupported alert type: %s", a.AlertType), nil
	}
}

func (e *Evaluator) evaluateSingle(ctx context.Context, orgID uuid.UUID, a domain.Alert) {
	triggered, message, details := e.EvaluateAlert(ctx, orgID, a)
	now := time.Now().UTC()

	if triggered {
		// Deduplication check:
		latestEvent, err := e.alerts.GetLatestEvent(ctx, orgID, a.ID)
		if err == nil && latestEvent != nil {
			if latestEvent.Status == "triggered" && a.Status == domain.AlertStatusActive {
				if time.Since(latestEvent.TriggeredAt) < 15*time.Minute {
					return
				}
			}
		}

		ctxBytes, _ := json.Marshal(details)

		// 1. Record trigger event in history
		_, err = e.alerts.CreateEvent(ctx, orgID, domain.NewAlertEvent{
			AlertID:     a.ID,
			AlertType:   a.AlertType,
			Severity:    a.Severity,
			Status:      "triggered",
			Message:     message,
			Context:     ctxBytes,
			TriggeredAt: now,
		})
		if err != nil {
			e.log.ErrorContext(ctx, "failed to record alert trigger event", "alert_id", a.ID, "error", err)
		}

		// 2. Update alert status and last_triggered_at
		_, err = e.alerts.UpdateTriggered(ctx, orgID, a.ID, domain.AlertStatusActive)
		if err != nil {
			e.log.ErrorContext(ctx, "failed to update alert status to active", "alert_id", a.ID, "error", err)
		}

		// 3. Dispatch notification with failure isolation
		if e.notifier != nil {
			err = e.notifier.Send(ctx, orgID, NotificationEvent{
				AlertID:   a.ID,
				AlertType: a.AlertType,
				Severity:  a.Severity,
				Message:   message,
				Context:   details,
			})
			if err != nil {
				e.log.ErrorContext(ctx, "failed to dispatch alert notification", "alert_id", a.ID, "error", err)
			}
		}
	} else if a.Status == domain.AlertStatusActive {
		// Condition returned to normal: record resolution event and update status
		ctxBytes, _ := json.Marshal(map[string]any{"resolved_at": now.Format(time.RFC3339)})
		_, err := e.alerts.CreateEvent(ctx, orgID, domain.NewAlertEvent{
			AlertID:     a.ID,
			AlertType:   a.AlertType,
			Severity:    a.Severity,
			Status:      "resolved",
			Message:     fmt.Sprintf("Alert resolved: %s condition returned to normal", a.Name),
			Context:     ctxBytes,
			TriggeredAt: now,
			ResolvedAt:  &now,
		})
		if err != nil {
			e.log.ErrorContext(ctx, "failed to record alert resolution event", "alert_id", a.ID, "error", err)
		}

		_, err = e.alerts.UpdateTriggered(ctx, orgID, a.ID, domain.AlertStatusResolved)
		if err != nil {
			e.log.ErrorContext(ctx, "failed to update alert status to resolved", "alert_id", a.ID, "error", err)
		}

		if e.notifier != nil {
			_ = e.notifier.Send(ctx, orgID, NotificationEvent{
				AlertID:   a.ID,
				AlertType: a.AlertType,
				Severity:  "info",
				Message:   fmt.Sprintf("Resolved: %s condition returned to normal", a.Name),
				Context:   map[string]any{"resolved_at": now.Format(time.RFC3339)},
			})
		}
	}
}

func (e *Evaluator) evalSpend(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdMicros int64  `json:"threshold_micros"`
		Window          string `json:"window"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ThresholdMicros <= 0 {
		cfg.ThresholdMicros = 50_000_000 // $50 default
	}

	window := 24 * time.Hour
	switch cfg.Window {
	case "hourly", "1h":
		window = time.Hour
	case "monthly", "30d":
		window = 30 * 24 * time.Hour
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	from := to.Add(-window)

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{From: from, To: to})
	if err != nil {
		e.log.ErrorContext(ctx, "failed to get usage summary for spend alert", "error", err)
		return false, "", nil
	}

	if summary.TotalCostMicros >= cfg.ThresholdMicros {
		return true,
			fmt.Sprintf("Spend threshold exceeded: %s (threshold: %s)", formatMicros(summary.TotalCostMicros), formatMicros(cfg.ThresholdMicros)),
			map[string]any{
				"current_spend_micros": summary.TotalCostMicros,
				"threshold_micros":     cfg.ThresholdMicros,
				"window":               cfg.Window,
				"total_requests":       summary.Requests,
			}
	}
	return false, "spend within limit", map[string]any{
		"current_spend_micros": summary.TotalCostMicros,
		"threshold_micros":     cfg.ThresholdMicros,
	}
}

func (e *Evaluator) evalRequestRate(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdPerMinute int `json:"threshold_per_minute"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ThresholdPerMinute <= 0 {
		cfg.ThresholdPerMinute = 60
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	from := to.Add(-5 * time.Minute)

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{From: from, To: to})
	if err != nil {
		return false, "", nil
	}

	reqPerMinute := float64(summary.Requests) / 5.0
	if reqPerMinute >= float64(cfg.ThresholdPerMinute) {
		return true,
			fmt.Sprintf("Request rate spike: %.1f req/min (threshold: %d req/min)", reqPerMinute, cfg.ThresholdPerMinute),
			map[string]any{
				"current_rate_per_minute": reqPerMinute,
				"threshold_per_minute":    cfg.ThresholdPerMinute,
				"total_requests":          summary.Requests,
			}
	}
	return false, "request rate normal", map[string]any{
		"current_rate_per_minute": reqPerMinute,
		"threshold_per_minute":    cfg.ThresholdPerMinute,
	}
}

func (e *Evaluator) evalErrorRate(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdPercent float64 `json:"threshold_percent"`
		MinRequests      int     `json:"min_requests"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ThresholdPercent <= 0 {
		cfg.ThresholdPercent = 10.0
	}
	if cfg.MinRequests <= 0 {
		cfg.MinRequests = 5
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	from := to.Add(-15 * time.Minute)

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{From: from, To: to})
	if err != nil || summary.Requests < int64(cfg.MinRequests) {
		return false, "insufficient requests to evaluate error rate", nil
	}

	errorRate := float64(summary.Errors) / float64(summary.Requests) * 100
	if errorRate >= cfg.ThresholdPercent {
		return true,
			fmt.Sprintf("Error rate spike: %.1f%% (threshold: %.1f%%)", errorRate, cfg.ThresholdPercent),
			map[string]any{
				"current_error_rate_percent": errorRate,
				"threshold_percent":          cfg.ThresholdPercent,
				"total_errors":               summary.Errors,
				"total_requests":             summary.Requests,
			}
	}
	return false, "error rate normal", map[string]any{
		"current_error_rate_percent": errorRate,
		"threshold_percent":          cfg.ThresholdPercent,
	}
}

func (e *Evaluator) evalProviderFailure(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Provider            string `json:"provider"`
		ConsecutiveFailures int    `json:"consecutive_failures"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ConsecutiveFailures <= 0 {
		cfg.ConsecutiveFailures = 3
	}

	// 1. Check live provider health probes if available
	if e.providerHealth != nil {
		if cfg.Provider != "" {
			state, err := e.providerHealth.Get(ctx, cfg.Provider)
			if err == nil && state.Status == "unhealthy" {
				return true,
					fmt.Sprintf("Provider %s is reporting unhealthy status", cfg.Provider),
					map[string]any{"provider": cfg.Provider, "status": state.Status, "error": state.LastError}
			}
		} else {
			states, err := e.providerHealth.List(ctx)
			if err == nil {
				for _, state := range states {
					if state.Status == "unhealthy" {
						return true,
							fmt.Sprintf("Provider %s is reporting unhealthy status", state.ProviderKind),
							map[string]any{"provider": state.ProviderKind, "status": state.Status, "error": state.LastError}
					}
				}
			}
		}

	}

	// 2. Check recent failed LLM requests for this organization
	if e.llmRequests != nil {
		to := time.Now().UTC()
		from := to.Add(-15 * time.Minute)
		statusFailed := domain.LlmRequestStatusFailed
		filter := domain.UsageFilter{
			From:   from,
			To:     to,
			Status: &statusFailed,
		}
		if cfg.Provider != "" {
			filter.Provider = &cfg.Provider
		}
		reqs, err := e.llmRequests.GetRequests(ctx, orgID, filter, int32(cfg.ConsecutiveFailures), 0)
		if err == nil && len(reqs) >= cfg.ConsecutiveFailures {
			return true,
				fmt.Sprintf("Multiple consecutive upstream provider failures detected (%d failures in 15m)", len(reqs)),
				map[string]any{"failure_count": len(reqs), "threshold": cfg.ConsecutiveFailures, "provider": cfg.Provider}
		}
	}

	return false, "all provider endpoints operational", nil
}

func (e *Evaluator) evalCredentialFailure(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Threshold int `json:"threshold"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Threshold <= 0 {
		cfg.Threshold = 3
	}

	if e.llmRequests != nil {
		to := time.Now().UTC()
		from := to.Add(-15 * time.Minute)
		statusFailed := domain.LlmRequestStatusFailed
		filter := domain.UsageFilter{
			From:   from,
			To:     to,
			Status: &statusFailed,
		}
		reqs, err := e.llmRequests.GetRequests(ctx, orgID, filter, 50, 0)
		if err == nil {
			var credFailures int
			for _, r := range reqs {
				if r.ErrorCode != nil && (*r.ErrorCode == "invalid_api_key" || *r.ErrorCode == "unauthorized" || *r.ErrorCode == "401" || *r.ErrorCode == "403") {
					credFailures++
				}
			}
			if credFailures >= cfg.Threshold {
				return true,
					fmt.Sprintf("Repeated credential authentication failures detected (%d in 15m)", credFailures),
					map[string]any{"credential_failures": credFailures, "threshold": cfg.Threshold}
			}
		}
	}

	return false, "credential authentication normal", nil
}

func (e *Evaluator) evalPolicyViolation(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Threshold int `json:"threshold"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Threshold <= 0 {
		cfg.Threshold = 3
	}

	if e.securityEvents != nil {
		to := time.Now().UTC()
		from := to.Add(-15 * time.Minute)
		cat := domain.SecurityEventPolicyViolation
		filter := domain.SecurityEventFilter{
			Category: &cat,
			From:     from,
			To:       to,
		}
		events, err := e.securityEvents.List(ctx, orgID, filter, int32(cfg.Threshold), 0)
		if err == nil && len(events) >= cfg.Threshold {
			return true,
				fmt.Sprintf("Governance policy violation threshold exceeded (%d violations in 15m)", len(events)),
				map[string]any{"violation_count": len(events), "threshold": cfg.Threshold}
		}
	}

	return false, "policy enforcement normal", nil
}

func (e *Evaluator) evalRateLimitSpike(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Threshold int `json:"threshold"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}

	if e.llmRequests != nil {
		to := time.Now().UTC()
		from := to.Add(-15 * time.Minute)
		statusFailed := domain.LlmRequestStatusFailed
		filter := domain.UsageFilter{
			From:   from,
			To:     to,
			Status: &statusFailed,
		}
		reqs, err := e.llmRequests.GetRequests(ctx, orgID, filter, 50, 0)
		if err == nil {
			var rateLimitCount int
			for _, r := range reqs {
				if r.ErrorCode != nil && (*r.ErrorCode == "rate_limit_exceeded" || *r.ErrorCode == "429") {
					rateLimitCount++
				}
			}
			if rateLimitCount >= cfg.Threshold {
				return true,
					fmt.Sprintf("Rate limit spike detected: %d requests rejected by rate limits in 15m", rateLimitCount),
					map[string]any{"rate_limited_requests": rateLimitCount, "threshold": cfg.Threshold}
			}
		}
	}

	return false, "rate limit throughput normal", nil
}

func (e *Evaluator) evalUsageAnomaly(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Multiplier      float64 `json:"multiplier"`
		ThresholdMicros int64   `json:"threshold_micros"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.ThresholdMicros <= 0 {
		cfg.ThresholdMicros = 1_000_000 // $1.00
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	// Current 1h
	currentSummary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: to.Add(-time.Hour),
		To:   to,
	})
	if err != nil || currentSummary.TotalCostMicros < cfg.ThresholdMicros {
		return false, "spend within nominal variance", nil
	}

	// Prior 24h baseline
	baselineSummary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: to.Add(-25 * time.Hour),
		To:   to.Add(-time.Hour),
	})
	if err != nil {
		return false, "", nil
	}

	hourlyAvg := float64(baselineSummary.TotalCostMicros) / 24.0
	if hourlyAvg > 0 && float64(currentSummary.TotalCostMicros) >= (hourlyAvg*cfg.Multiplier) {
		return true,
			fmt.Sprintf("Usage anomaly: current hourly spend of %s is %.1fx greater than 24h average (%s/hr)",
				formatMicros(currentSummary.TotalCostMicros),
				float64(currentSummary.TotalCostMicros)/hourlyAvg,
				formatMicros(int64(hourlyAvg))),
			map[string]any{
				"current_hourly_spend_micros": currentSummary.TotalCostMicros,
				"baseline_hourly_avg_micros":  int64(hourlyAvg),
				"ratio":                       float64(currentSummary.TotalCostMicros) / hourlyAvg,
			}
	}

	return false, "usage metrics within expected baseline", nil
}

func formatMicros(micros int64) string {
	dollars := float64(micros) / 1_000_000
	return fmt.Sprintf("$%.2f", dollars)
}

func (e *Evaluator) evalBudgetWarning(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		BudgetMicros            int64   `json:"budget_micros"`
		ThresholdMicros         int64   `json:"threshold_micros"`
		WarningThresholdPercent float64 `json:"warning_threshold_percent"`
		Window                  string  `json:"window"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ThresholdMicros <= 0 && cfg.BudgetMicros > 0 {
		pct := cfg.WarningThresholdPercent
		if pct <= 0 {
			pct = 80.0
		}
		cfg.ThresholdMicros = int64(float64(cfg.BudgetMicros) * (pct / 100.0))
	}
	if cfg.ThresholdMicros <= 0 {
		cfg.ThresholdMicros = 40_000_000 // $40 default
	}

	window := 30 * 24 * time.Hour
	switch cfg.Window {
	case "daily", "24h":
		window = 24 * time.Hour
	case "weekly", "7d":
		window = 7 * 24 * time.Hour
	case "hourly", "1h":
		window = time.Hour
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	from := to.Add(-window)

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{From: from, To: to})
	if err != nil {
		return false, "", nil
	}

	if summary.TotalCostMicros >= cfg.ThresholdMicros {
		return true,
			fmt.Sprintf("Budget warning: spend of %s has reached or exceeded warning threshold %s",
				formatMicros(summary.TotalCostMicros), formatMicros(cfg.ThresholdMicros)),
			map[string]any{
				"current_spend_micros": summary.TotalCostMicros,
				"threshold_micros":     cfg.ThresholdMicros,
				"budget_micros":        cfg.BudgetMicros,
				"window":               cfg.Window,
			}
	}

	return false, "spend within warning threshold", map[string]any{
		"current_spend_micros": summary.TotalCostMicros,
		"threshold_micros":     cfg.ThresholdMicros,
	}
}

func (e *Evaluator) evalBudgetExhaustion(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		BudgetMicros    int64  `json:"budget_micros"`
		ThresholdMicros int64  `json:"threshold_micros"`
		Window          string `json:"window"`
	}
	_ = json.Unmarshal(config, &cfg)
	limit := cfg.BudgetMicros
	if limit <= 0 {
		limit = cfg.ThresholdMicros
	}
	if limit <= 0 {
		limit = 50_000_000 // $50 default
	}

	window := 30 * 24 * time.Hour
	switch cfg.Window {
	case "daily", "24h":
		window = 24 * time.Hour
	case "hourly", "1h":
		window = time.Hour
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	from := to.Add(-window)

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{From: from, To: to})
	if err != nil {
		return false, "", nil
	}

	if summary.TotalCostMicros >= limit {
		return true,
			fmt.Sprintf("Budget exhausted: total spend of %s reached or exceeded budget ceiling %s",
				formatMicros(summary.TotalCostMicros), formatMicros(limit)),
			map[string]any{
				"current_spend_micros":  summary.TotalCostMicros,
				"budget_ceiling_micros": limit,
				"window":                cfg.Window,
			}
	}

	return false, "budget not exhausted", map[string]any{
		"current_spend_micros":  summary.TotalCostMicros,
		"budget_ceiling_micros": limit,
	}
}

func (e *Evaluator) evalSpendSpike(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		Multiplier      float64 `json:"multiplier"`
		ThresholdMicros int64   `json:"threshold_micros"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.ThresholdMicros <= 0 {
		cfg.ThresholdMicros = 10_000_000 // $10.00
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	currentSummary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: to.Add(-time.Hour),
		To:   to,
	})
	if err != nil {
		return false, "", nil
	}

	if currentSummary.TotalCostMicros >= cfg.ThresholdMicros {
		return true,
			fmt.Sprintf("Spend spike detected: hourly spend of %s exceeded threshold of %s",
				formatMicros(currentSummary.TotalCostMicros), formatMicros(cfg.ThresholdMicros)),
			map[string]any{
				"hourly_spend_micros": currentSummary.TotalCostMicros,
				"threshold_micros":    cfg.ThresholdMicros,
			}
	}

	return false, "spend velocity normal", nil
}

func (e *Evaluator) evalUsageSpike(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdRequests int `json:"threshold_requests"`
	}
	_ = json.Unmarshal(config, &cfg)
	if cfg.ThresholdRequests <= 0 {
		cfg.ThresholdRequests = 50
	}

	if e.usageRollups == nil {
		return false, "usage rollups unavailable", nil
	}

	to := time.Now().UTC()
	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: to.Add(-15 * time.Minute),
		To:   to,
	})
	if err != nil {
		return false, "", nil
	}

	if int(summary.Requests) >= cfg.ThresholdRequests {
		return true,
			fmt.Sprintf("Unexpected usage spike detected: %d requests in the last 15 minutes (threshold: %d)",
				summary.Requests, cfg.ThresholdRequests),
			map[string]any{
				"requests_in_window": summary.Requests,
				"threshold_requests": cfg.ThresholdRequests,
			}
	}

	return false, "usage rate normal", nil
}
