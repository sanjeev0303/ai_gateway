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

export type AlertType = "spend_limit" | "rate_limit" | "error_rate" | "anomaly";

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

export interface AlertsModule {
  list(options?: RequestOptions): Promise<ListResponse<Alert>>;
  get(alertId: string, options?: RequestOptions): Promise<Alert>;
  update(
    alertId: string,
    input: UpdateAlertInput,
    options?: RequestOptions,
  ): Promise<Alert>;
  delete(alertId: string, options?: RequestOptions): Promise<void>;
  test(alertId: string, options?: RequestOptions): Promise<AlertTestResult>;
}

export function createAlertsModule(transport: Transport): AlertsModule {
  return {
    list: (options) => transport.request<ListResponse<Alert>>("/v1/alerts", options),
    get: (alertId, options) =>
      transport.requestAt<Alert>(`/v1/alerts/${encodeURIComponent(alertId)}`, options),
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
  };
}