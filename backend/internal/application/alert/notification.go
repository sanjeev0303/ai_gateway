package alert

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// NotificationEvent represents the trigger of an alert.
type NotificationEvent struct {
	AlertID      uuid.UUID
	AlertType    string
	Severity     string
	Message      string
	Context      map[string]any
}

// Notifier abstracts delivery mechanisms (in-app, email, webhook).
type Notifier interface {
	Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error
}

// InAppNotifier stores alerts in a database table or similar for UI consumption.
// (Currently a stub that just logs, pending actual DB table creation for notifications)
type InAppNotifier struct {
	log *slog.Logger
}

func NewInAppNotifier(log *slog.Logger) *InAppNotifier {
	return &InAppNotifier{log: log}
}

func (n *InAppNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	n.log.InfoContext(ctx, "IN-APP NOTIFICATION", "org_id", orgID, "alert_id", event.AlertID, "message", event.Message)
	return nil
}

// EmailNotifier sends emails. It does not require a production email provider yet.
type EmailNotifier struct {
	log *slog.Logger
}

func NewEmailNotifier(log *slog.Logger) *EmailNotifier {
	return &EmailNotifier{log: log}
}

func (n *EmailNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	n.log.InfoContext(ctx, "EMAIL NOTIFICATION", "org_id", orgID, "alert_id", event.AlertID, "message", event.Message)
	return nil
}

// WebhookNotifier sends HTTP POSTs to configured endpoints.
type WebhookNotifier struct {
	log *slog.Logger
}

func NewWebhookNotifier(log *slog.Logger) *WebhookNotifier {
	return &WebhookNotifier{log: log}
}

func (n *WebhookNotifier) Send(ctx context.Context, orgID uuid.UUID, event NotificationEvent) error {
	n.log.InfoContext(ctx, "WEBHOOK NOTIFICATION", "org_id", orgID, "alert_id", event.AlertID, "message", event.Message)
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
