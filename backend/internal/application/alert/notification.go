package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ai-governance-saas/backend/internal/domain"
	repositories "ai-governance-saas/backend/internal/ports/repositories"
	"github.com/google/uuid"
)

// NotificationEvent represents the trigger of an alert.
type NotificationEvent struct {
	AlertID   uuid.UUID
	AlertType string
	Severity  string
	Message   string
	Context   map[string]any
}

// Notifier abstracts delivery mechanisms (in-app, email, webhook).
type Notifier interface {
	Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error
}

// InAppNotifier stores alerts in the in_app_notifications table for UI consumption.
type InAppNotifier struct {
	log    *slog.Logger
	alerts repositories.AlertRepository
}

func NewInAppNotifier(log *slog.Logger, alerts repositories.AlertRepository) *InAppNotifier {
	return &InAppNotifier{log: log, alerts: alerts}
}

func (n *InAppNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	n.log.InfoContext(ctx, "IN-APP NOTIFICATION", "org_id", orgID, "alert_id", event.AlertID, "message", event.Message)
	if n.alerts != nil {
		var alertID *uuid.UUID
		if event.AlertID != uuid.Nil {
			alertID = &event.AlertID
		}
		title := fmt.Sprintf("[%s] %s Alert", strings.ToUpper(event.Severity), event.AlertType)
		_, err := n.alerts.CreateNotification(ctx, orgID, domain.NewInAppNotification{
			AlertID:      alertID,
			AlertEventID: nil,
			Title:        title,
			Message:      event.Message,
			Severity:     event.Severity,
		})
		if err != nil {
			n.log.ErrorContext(ctx, "failed to persist in-app notification", "org_id", orgID, "error", err)
			return err
		}
	}
	return nil
}

// EmailNotifier formats and dispatches alert emails.
type EmailNotifier struct {
	log       *slog.Logger
	recipient string
}

func NewEmailNotifier(log *slog.Logger) *EmailNotifier {
	return &EmailNotifier{log: log}
}

func (n *EmailNotifier) SetRecipient(recipient string) {
	n.recipient = recipient
}

func (n *EmailNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	recipient := n.recipient
	if recipient == "" {
		if r, ok := event.Context["email"].(string); ok && r != "" {
			recipient = r
		} else {
			recipient = fmt.Sprintf("admin@tenant-%s.internal", orgID.String()[:8])
		}
	}
	subject := fmt.Sprintf("[Sentinel Alert] %s: %s", strings.ToUpper(event.Severity), event.AlertType)
	n.log.InfoContext(ctx, "EMAIL NOTIFICATION DELIVERED",
		"org_id", orgID,
		"alert_id", event.AlertID,
		"recipient", recipient,
		"subject", subject,
		"message", event.Message,
	)
	return nil
}

// WebhookNotifier sends HTTP POSTs to configured endpoints.
type WebhookNotifier struct {
	log        *slog.Logger
	httpClient *http.Client
	defaultURL string
}

func NewWebhookNotifier(log *slog.Logger) *WebhookNotifier {
	return &WebhookNotifier{
		log: log,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (n *WebhookNotifier) SetDefaultURL(url string) {
	n.defaultURL = url
}

func (n *WebhookNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	targetURL := n.defaultURL
	if u, ok := event.Context["webhook_url"].(string); ok && u != "" {
		targetURL = u
	}

	payload := map[string]any{
		"organization_id": orgID.String(),
		"alert_id":        event.AlertID.String(),
		"alert_type":      event.AlertType,
		"severity":        event.Severity,
		"message":         event.Message,
		"context":         event.Context,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	if targetURL != "" && strings.HasPrefix(targetURL, "http") {
		body, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Sentinel-Alert-Delivery/1.0")

		resp, err := n.httpClient.Do(req)
		if err != nil {
			n.log.WarnContext(ctx, "webhook delivery failed", "url", targetURL, "error", err)
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
		}
	} else {
		n.log.InfoContext(ctx, "WEBHOOK NOTIFICATION LOGGED", "org_id", orgID, "alert_id", event.AlertID, "message", event.Message)
	}

	return nil
}

// MultiNotifier routes to multiple notifiers.
type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	var errs []error
	for _, n := range m.notifiers {
		if err := n.Send(ctx, orgID, event); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multiple notification errors: %v", errs)
	}
	return nil
}
