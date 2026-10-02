import type { ID, ISODateString, StatusTone } from "./common";

export type AuditAction =
  | "create"
  | "update"
  | "delete"
  | "login"
  | "logout"
  | "export"
  | "policy_change"
  | "permission_change"
  | "credential_use";

export type AuditActorType = "user" | "api_key" | "system";

export interface AuditLogEntry {
  id: ID;
  occurredAt: ISODateString;
  actorType: AuditActorType;
  actorId: ID;
  actorName: string;
  actorEmail?: string;
  /** Source IP, already truncated by the edge proxy. */
  ipAddress: string;
  userAgent: string;
  action: AuditAction;
  /** Dotted resource path, e.g. "policy/pol_12". */
  resource: string;
  /** Human-readable before/after summary of the change. */
  details: string;
  ipCountry: string;
  /** True when the actor authenticated with a second factor. */
  mfaVerified: boolean;
}

export const AUDIT_ACTION_TONE: Record<AuditAction, StatusTone> = {
  create: "success",
  update: "info",
  delete: "danger",
  login: "neutral",
  logout: "neutral",
  export: "warning",
  policy_change: "warning",
  permission_change: "warning",
  credential_use: "info",
};
