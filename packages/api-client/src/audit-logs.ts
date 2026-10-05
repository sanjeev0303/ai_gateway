import type { ListResponse, RequestOptions, Transport } from "./client";

/**
 * Audit-log payloads are the snake_case DTOs declared in
 * `backend/api/openapi/openapi.yaml`, mirrored here so a consumer of this
 * package does not need `@ai-governance/contracts` at the type level.
 *
 * Audit records are append-only: this module exposes no way to edit or delete
 * one, and the server exposes none either.
 */

export type AuditActorType = "user" | "service_account";

export interface AuditLog {
  id: string;
  /** Reports which of the two actor id columns is populated. */
  actor_type: AuditActorType;
  actor_user_id?: string | null;
  actor_service_account_id?: string | null;
  /** Dotted action name, e.g. `organization.update`. */
  action: string;
  resource_type: string;
  resource_id?: string | null;
  /** State before the change; `{}` when not recorded. */
  before: Record<string, unknown>;
  /** State after the change; `{}` when not recorded. */
  after: Record<string, unknown>;
  ip_address?: string | null;
  /** Correlates the record with gateway and server logs. */
  request_id?: string | null;
  /** RFC3339. */
  created_at: string;
}

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