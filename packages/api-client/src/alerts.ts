import type { ListResponse, RequestOptions, Transport } from "./client";

/**
 * Alert-rule payloads are the snake_case DTOs declared in
 * `backend/api/openapi/openapi.yaml`, mirrored here so a consumer of this
 * package does not need `@ai-governance/contracts` at the type level.
 *
 * Nothing evaluates alert rules yet. `last_triggered_at` therefore stays null,
 * and `test()` validates a rule's configuration without reporting whether it
 * would fire and without sending a notification.
 */

export type AlertType =
  | "spend_threshold"
  | "spend_exceeded"
  | "request_rate_spike"
  | "error_rate_spike"
  | "provider_failure"
  | "credential_failure"
  | "policy_violation"
  | "rate_limit_spike"
  | "usage_anomaly";

export type AlertSeverity = "info" | "warning" | "critical";

export type AlertStatus = "active" | "muted";

export interface Alert {
  id: string;
  alert_type: AlertType;
  name: string;
  severity: AlertSeverity;
  /** Rule thresholds and options; `{}` when unset. */
  configuration: Record<string, unknown>;
  status: AlertStatus;
  /** Always null today, because nothing evaluates alert rules. */
  last_triggered_at?: string | null;
  /** RFC3339. */
  created_at: string;
  /** RFC3339. */
  updated_at: string;
}

export interface CreateAlertInput {
  name: string;
  alert_type: AlertType;
  severity: AlertSeverity;
  configuration: Record<string, unknown>;
}

/** Full replacement of an alert rule's mutable fields; all four are required. */
export interface UpdateAlertInput {
  name: string;
  severity: AlertSeverity;
  configuration: Record<string, unknown>;
  status: AlertStatus;
}

export interface AlertTestResult {
  alert_id: string;
  alert_type: AlertType;
  severity: AlertSeverity;
  status: AlertStatus;
  configuration: Record<string, unknown>;
  /** True when the rule is muted and so would not be reported. */
  status_is_muted: boolean;
  /** Always false; no notifier is invoked. */
  notification_sent: boolean;
  message: string;
}

export interface AlertEvent {
  id: string;
  alert_id: string;
  alert_type: string;
  severity: string;
  status: string;
  message: string;
  context: Record<string, unknown>;
  triggered_at: string;
  resolved_at?: string | null;
  created_at: string;
}

export interface InAppNotification {
  id: string;
  alert_id?: string | null;
  alert_event_id?: string | null;
  title: string;
  message: string;
  severity: string;
  read: boolean;
  created_at: string;
}

export interface AlertsModule {
  list(options?: RequestOptions): Promise<ListResponse<Alert>>;
  get(alertId: string, options?: RequestOptions): Promise<Alert>;
  create(input: CreateAlertInput, options?: RequestOptions): Promise<Alert>;
  update(
    alertId: string,
    input: UpdateAlertInput,
    options?: RequestOptions,
  ): Promise<Alert>;
  delete(alertId: string, options?: RequestOptions): Promise<void>;
  test(alertId: string, options?: RequestOptions): Promise<AlertTestResult>;
  listEvents(alertId?: string, options?: RequestOptions): Promise<ListResponse<AlertEvent>>;
  acknowledgeEvent(eventId: string, options?: RequestOptions): Promise<AlertEvent>;
  resolveEvent(eventId: string, options?: RequestOptions): Promise<AlertEvent>;
  listNotifications(unreadOnly?: boolean, options?: RequestOptions): Promise<ListResponse<InAppNotification>>;
  markNotificationRead(notificationId: string, options?: RequestOptions): Promise<void>;
  markAllNotificationsRead(options?: RequestOptions): Promise<void>;
}

export function createAlertsModule(transport: Transport): AlertsModule {
  return {
    list: (options) => transport.request<ListResponse<Alert>>("/v1/alerts", options),
    get: (alertId, options) =>
      transport.requestAt<Alert>(`/v1/alerts/${encodeURIComponent(alertId)}`, options),
    create: (input, options) =>
      transport.requestAt<Alert>("/v1/alerts", {
        ...options,
        method: "POST",
        body: input,
      }),
    update: (alertId, input, options) =>
      transport.requestAt<Alert>(`/v1/alerts/${encodeURIComponent(alertId)}`, {
        ...options,
        method: "PUT",
        body: input,
      }),
    delete: (alertId, options) =>
      transport.requestAt<void>(`/v1/alerts/${encodeURIComponent(alertId)}`, {
        ...options,
        method: "DELETE",
      }),
    test: (alertId, options) =>
      transport.requestAt<AlertTestResult>(
        `/v1/alerts/${encodeURIComponent(alertId)}/test`,
        { ...options, method: "POST" },
      ),
    listEvents: (alertId, options) => {
      const path = alertId
        ? `/v1/alerts/${encodeURIComponent(alertId)}/events`
        : "/v1/alert-events";
      return transport.request<ListResponse<AlertEvent>>(path, options);
    },
    acknowledgeEvent: (eventId, options) =>
      transport.request<AlertEvent>(
        `/v1/alert-events/${encodeURIComponent(eventId)}/acknowledge`,
        { ...options, method: "POST" },
      ),
    resolveEvent: (eventId, options) =>
      transport.request<AlertEvent>(
        `/v1/alert-events/${encodeURIComponent(eventId)}/resolve`,
        { ...options, method: "POST" },
      ),
    listNotifications: (unreadOnly, options) =>
      transport.request<ListResponse<InAppNotification>>(
        unreadOnly ? "/v1/notifications?unread_only=true" : "/v1/notifications",
        options,
      ),
    markNotificationRead: (notificationId, options) =>
      transport.request<void>(
        `/v1/notifications/${encodeURIComponent(notificationId)}/read`,
        { ...options, method: "PATCH" },
      ),
    markAllNotificationsRead: (options) =>
      transport.request<void>("/v1/notifications/read-all", {
        ...options,
        method: "POST",
      }),
  };
}