import type { ID, ISODateString, StatusTone } from "./common";

export type AlertSeverity = "critical" | "high" | "medium" | "low";

export type AlertState = "open" | "acknowledged" | "resolved";

export type AlertCategory =
  | "cost_anomaly"
  | "policy_violation"
  | "provider_outage"
  | "error_rate"
  | "credential_expiry"
  | "budget_exceeded"
  | "unusual_access";

export interface Alert {
  id: ID;
  occurredAt: ISODateString;
  severity: AlertSeverity;
  state: AlertState;
  category: AlertCategory;
  title: string;
  description: string;
  /** Dashed path into the relevant dashboard page. */
  link: string;
  /** Ids of users notified. */
  notifiedUserIds: ID[];
  acknowledgedAt?: ISODateString;
  acknowledgedById?: ID;
  resolvedAt?: ISODateString;
}

export const ALERT_SEVERITY_TONE: Record<AlertSeverity, StatusTone> = {
  critical: "danger",
  high: "danger",
  medium: "warning",
  low: "info",
};
