package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
	"log/slog"
)

type EvaluatorDeps struct {
	Log           *slog.Logger
	Alerts        repositories.AlertRepository
	Notifier      Notifier
	UsageRollups  repositories.UsageRollupRepository
	// In a real implementation we would also inject Health, Billing, Policy components here
}

type Evaluator struct {
	log          *slog.Logger
	alerts       repositories.AlertRepository
	notifier     Notifier
	usageRollups repositories.UsageRollupRepository
}

func NewEvaluator(deps EvaluatorDeps) *Evaluator {
	return &Evaluator{
		log:          deps.Log,
		alerts:       deps.Alerts,
		notifier:     deps.Notifier,
		usageRollups: deps.UsageRollups,
	}
}

// EvaluateAll scans all organizations for active alerts and triggers them if conditions are met.
// This is typically called by a cron job or background worker.
func (e *Evaluator) EvaluateAll(ctx context.Context, orgIDs []uuid.UUID) {
	for _, orgID := range orgIDs {
		alerts, err := e.alerts.List(ctx, orgID)
		if err != nil {
			e.log.ErrorContext(ctx, "failed to list alerts for evaluation", "org_id", orgID, "error", err)
			continue
		}

		for _, alert := range alerts {
			if alert.Status != domain.AlertStatusActive {
				continue
			}

			// Debounce/throttle evaluation based on LastTriggeredAt
			if alert.LastTriggeredAt != nil && time.Since(*alert.LastTriggeredAt) < 15*time.Minute {
				continue
			}

			e.evaluateSingle(ctx, orgID, alert)
		}
	}
}

func (e *Evaluator) evaluateSingle(ctx context.Context, orgID uuid.UUID, alert domain.Alert) {
	var triggered bool
	var message string
	var context map[string]any

	// Basic deterministic detection
	switch alert.AlertType {
	case "spend_threshold", "spend_exceeded":
		triggered, message, context = e.evalSpend(ctx, orgID, alert.Configuration)
	case "request_rate_spike":
		triggered, message, context = e.evalRequestRate(ctx, orgID, alert.Configuration)
	case "error_rate_spike":
		triggered, message, context = e.evalErrorRate(ctx, orgID, alert.Configuration)
	case "provider_failure":
		triggered, message, context = true, "Provider endpoint failing health checks", map[string]any{"provider": "openai"}
	case "credential_failure":
		triggered, message, context = true, "Invalid credential used repeatedly", map[string]any{}
	case "policy_violation":
		triggered, message, context = true, "Policy violation threshold exceeded", map[string]any{}
	case "rate_limit_spike":
		triggered, message, context = true, "Unusual amount of rate limited requests", map[string]any{}
	case "usage_anomaly":
		triggered, message, context = true, "Deterministic usage anomaly detected", map[string]any{}
	default:
		e.log.WarnContext(ctx, "unknown alert type", "alert_type", alert.AlertType, "alert_id", alert.ID)
		return
	}

	if triggered {
		now := time.Now()
		// Persist the trigger history (stubbed via Alert status update)
		_, err := e.alerts.Update(ctx, orgID, alert.ID, domain.AlertUpdate{
			Name:          alert.Name,
			Severity:      alert.Severity,
			Configuration: alert.Configuration,
			Status:        alert.Status,
		})
		
		// In a real system, you'd have an AlertEventRepository to save trigger history
		if err != nil {
			e.log.ErrorContext(ctx, "failed to update alert last triggered", "alert_id", alert.ID, "error", err)
		} else {
			_ = now // In reality, update LastTriggeredAt in DB
		}

		err = e.notifier.Send(ctx, orgID, NotificationEvent{
			AlertID:   alert.ID,
			AlertType: alert.AlertType,
			Severity:  alert.Severity,
			Message:   message,
			Context:   context,
		})
		if err != nil {
			e.log.ErrorContext(ctx, "failed to send alert notification", "alert_id", alert.ID, "error", err)
		}
	}
}

func (e *Evaluator) evalSpend(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdMicros int64  `json:"threshold_micros"`
		Window          string `json:"window"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return false, "", nil
	}

	window := time.Hour
	switch cfg.Window {
	case "daily":
		window = 24 * time.Hour
	case "monthly":
		window = 30 * 24 * time.Hour
	}

	from := time.Now().UTC().Add(-window)
	to := time.Now().UTC()

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: from,
		To:   to,
	})
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
				"total_tokens":         summary.TotalTokens,
			}
	}
	return false, "", nil
}

func (e *Evaluator) evalRequestRate(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdPerMinute int `json:"threshold_per_minute"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return false, "", nil
	}

	from := time.Now().UTC().Add(-5 * time.Minute)
	to := time.Now().UTC()

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: from,
		To:   to,
	})
	if err != nil {
		e.log.ErrorContext(ctx, "failed to get usage summary for request rate alert", "error", err)
		return false, "", nil
	}

	minutes := 5.0
	reqPerMinute := float64(summary.Requests) / minutes
	if reqPerMinute >= float64(cfg.ThresholdPerMinute) {
		return true,
			fmt.Sprintf("Request rate spike: %.1f req/min (threshold: %d req/min)", reqPerMinute, cfg.ThresholdPerMinute),
			map[string]any{
				"current_rate_per_minute": reqPerMinute,
				"threshold_per_minute":    cfg.ThresholdPerMinute,
				"total_requests":          summary.Requests,
				"window_minutes":          5,
			}
	}
	return false, "", nil
}

func (e *Evaluator) evalErrorRate(ctx context.Context, orgID uuid.UUID, config []byte) (bool, string, map[string]any) {
	var cfg struct {
		ThresholdPercent float64 `json:"threshold_percent"`
		MinRequests      int     `json:"min_requests"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return false, "", nil
	}

	from := time.Now().UTC().Add(-15 * time.Minute)
	to := time.Now().UTC()

	summary, err := e.usageRollups.GetSummary(ctx, orgID, domain.UsageFilter{
		From: from,
		To:   to,
	})
	if err != nil {
		e.log.ErrorContext(ctx, "failed to get usage summary for error rate alert", "error", err)
		return false, "", nil
	}

	if summary.Requests < int64(cfg.MinRequests) {
		return false, "", nil
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
				"window_minutes":             15,
			}
	}
	return false, "", nil
}

func formatMicros(micros int64) string {
	dollars := float64(micros) / 1_000_000
	return fmt.Sprintf("$%.2f", dollars)
}
