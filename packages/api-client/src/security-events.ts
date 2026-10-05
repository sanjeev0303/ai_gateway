import type { ListResponse, RequestOptions, Transport } from "./client";

/**
 * Security-event payloads are the snake_case DTOs declared in
 * `backend/api/openapi/openapi.yaml`, mirrored here so a consumer of this
 * package does not need `@ai-governance/contracts` at the type level.
 *
 * Security events are platform-originated and append-only: this module exposes
 * no way to edit or delete one, and the server exposes none either. Unlike the
 * audit log, this feed is not a record of what users did — it reports changes
 * in the platform itself.
 */

export type SecurityEventCategory = "provider_incident" | "pipeline_failure";

export type SecurityEventSeverity = "info" | "warning" | "critical";

export interface SecurityEvent {
  id: string;
  category: SecurityEventCategory;
  severity: SecurityEventSeverity;
  /** Short human-readable summary of what changed. */
  title: string;
  /**
   * Structured context, e.g. the provider kind and the from/to status for a
   * `provider_incident`, or the failing stage for a `pipeline_failure`.
   * `{}` when nothing was recorded.
   */
  detail: Record<string, unknown>;
  /** Originating subsystem, e.g. `provider_health` or `usage_rollup`. */
  source: string;
  /** RFC3339. */
  created_at: string;
}

export interface SecurityEventListQuery {
  category?: SecurityEventCategory;
  severity?: SecurityEventSeverity;
  /** RFC3339; a `Date` is serialised as ISO 8601. */
  from?: string | Date;
  /** Exclusive upper bound. */
  to?: string | Date;
  /** 1 to 200; the server defaults to 50. */
  limit?: number;
  offset?: number;
}

export interface SecurityEventsModule {
  list(query?: SecurityEventListQuery, options?: RequestOptions): Promise<ListResponse<SecurityEvent>>;
}

export function createSecurityEventsModule(transport: Transport): SecurityEventsModule {
  return {
    list: (query, options) =>
      transport.request<ListResponse<SecurityEvent>>("/v1/security-events", {
        ...options,
        query: {
          category: query?.category,
          severity: query?.severity,
          from: query?.from,
          to: query?.to,
          limit: query?.limit,
          offset: query?.offset,
        },
      }),
  };
}