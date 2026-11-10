import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

export type AlertType = components["schemas"]["AlertRuleType"];
export type AlertSeverity = components["schemas"]["Alert"]["severity"];
export type AlertStatus = components["schemas"]["Alert"]["status"];
export type Alert = components["schemas"]["Alert"];
export type CreateAlertInput = components["schemas"]["CreateAlertRequest"];
export type UpdateAlertInput = components["schemas"]["UpdateAlertRequest"];
export type AlertTestResult = components["schemas"]["AlertTestResult"];
export type AlertEvent = components["schemas"]["AlertEvent"];
export type InAppNotification = components["schemas"]["InAppNotification"];

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