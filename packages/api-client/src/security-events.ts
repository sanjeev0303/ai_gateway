import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

/** Platform-originated and append-only; unlike the audit log, not a record of what users did. */
export type SecurityEvent = components["schemas"]["SecurityEvent"];
export type SecurityEventCategory = SecurityEvent["category"];
export type SecurityEventSeverity = SecurityEvent["severity"];

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