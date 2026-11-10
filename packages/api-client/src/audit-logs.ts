import type { components } from "@ai-governance/contracts";
import type { ListResponse, RequestOptions, Transport } from "./client";

/** Audit records are append-only: neither this module nor the server can edit or delete one. */
export type AuditLog = components["schemas"]["AuditLog"];

/** Filter value; a returned record's actor_type can also be "" when no actor is recorded. */
export type AuditActorType = "user" | "service_account";

export interface AuditLogQuery {
  /** Matches either actor id column; combine with `actor_type` to disambiguate. */
  actor_id?: string;
  actor_type?: AuditActorType;
  action?: string;
  resource_type?: string;
  resource_id?: string;
  /** RFC3339; a `Date` is serialised as ISO 8601. */
  from?: string | Date;
  /** Exclusive upper bound. */
  to?: string | Date;
}

export interface AuditLogListQuery extends AuditLogQuery {
  /** 1 to 200; the server defaults to 50. */
  limit?: number;
  offset?: number;
}

export interface AuditLogsModule {
  list(query?: AuditLogListQuery, options?: RequestOptions): Promise<ListResponse<AuditLog>>;
  get(auditLogId: string, options?: RequestOptions): Promise<AuditLog>;
}

export function createAuditLogsModule(transport: Transport): AuditLogsModule {
  return {
    list: (query, options) =>
      transport.request<ListResponse<AuditLog>>("/v1/audit-logs", {
        ...options,
        query: {
          actor_id: query?.actor_id,
          actor_type: query?.actor_type,
          action: query?.action,
          resource_type: query?.resource_type,
          resource_id: query?.resource_id,
          from: query?.from,
          to: query?.to,
          limit: query?.limit,
          offset: query?.offset,
        },
      }),
    get: (auditLogId, options) =>
      transport.requestAt<AuditLog>(
        `/v1/audit-logs/${encodeURIComponent(auditLogId)}`,
        options,
      ),
  };
}