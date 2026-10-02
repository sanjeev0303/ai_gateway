import type { ID, ISODateString, StatusTone } from "./common";

export type ActivityStatus = "success" | "error" | "blocked" | "pending";

export type ActivityEvent = "completion" | "chat" | "embedding" | "tool_call";

export interface ActivityEntry {
  id: ID;
  occurredAt: ISODateString;
  userId: ID;
  userName: string;
  projectId: ID;
  projectName: string;
  modelId: ID;
  modelName: string;
  providerId: ID;
  providerName: string;
  event: ActivityEvent;
  status: ActivityStatus;
  inputTokens: number;
  outputTokens: number;
  /** Whole USD. */
  cost: number;
  latencyMs: number;
  /** Short human-readable summary, e.g. "summarized 4 documents". */
  summary: string;
  /** Redacted upstream request id, safe to display. */
  requestId: string;
}

export const ACTIVITY_STATUS_TONE: Record<ActivityStatus, StatusTone> = {
  success: "success",
  error: "danger",
  blocked: "warning",
  pending: "info",
};
