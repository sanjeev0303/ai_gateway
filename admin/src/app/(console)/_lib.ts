import type { AdminWindow } from "@ai-governance/api-client";

import { DEFAULT_RANGE, isDateRange } from "@ai-governance/console-ui/lib/date-range";
import { formatUsd, microsToUsd } from "@ai-governance/console-ui/lib/format";

export type SearchParams = Promise<{ [key: string]: string | string[] | undefined }>;

/** One value of a query parameter; repeated parameters keep the first. */
export function param(params: Awaited<SearchParams>, key: string): string {
  const value = params[key];
  return (Array.isArray(value) ? value[0] : value) ?? "";
}

/** The `range` parameter RangeTabs writes, as the admin API's window. */
export function windowFrom(params: Awaited<SearchParams>): AdminWindow {
  const range = param(params, "range");
  return isDateRange(range) ? range : DEFAULT_RANGE;
}

export const WINDOW_LABEL: Record<AdminWindow, string> = {
  "24h": "last 24 hours",
  "7d": "last 7 days",
  "30d": "last 30 days",
  "90d": "last 90 days",
};

/** Money arrives in micro-dollars everywhere in the admin API. */
export function usd(micros: number): string {
  return formatUsd(microsToUsd(micros));
}

export const PAGE_SIZE = 25;

/** The offset for a 1-based `page` parameter. */
export function pageFrom(params: Awaited<SearchParams>): { page: number; offset: number } {
  const page = Math.max(1, Number.parseInt(param(params, "page"), 10) || 1);
  return { page, offset: (page - 1) * PAGE_SIZE };
}

/** Builds a link to the same list with one parameter changed. */
export function hrefWith(path: string, params: Awaited<SearchParams>, changes: Record<string, string | number | null>): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    const first = Array.isArray(value) ? value[0] : value;
    if (first) query.set(key, first);
  }
  for (const [key, value] of Object.entries(changes)) {
    // Page 1 and empty filters stay implicit, keeping shared links clean.
    if (value === null || value === "" || (key === "page" && value === 1)) query.delete(key);
    else query.set(key, String(value));
  }
  const qs = query.toString();
  return qs ? `${path}?${qs}` : path;
}

const ACTION_LABEL: Record<string, string> = {
  "platform.organization_suspended": "Suspended organization",
  "platform.organization_reactivated": "Reactivated organization",
  "platform.organization_plan_changed": "Changed plan",
  "platform.organization_keys_revoked": "Revoked all gateway keys",
  "platform.user_disabled": "Disabled user",
  "platform.user_enabled": "Re-enabled user",
  "platform.model_activated": "Activated model",
  "platform.model_deactivated": "Deactivated model",
  "platform.alert_acknowledged": "Acknowledged alert",
};

/** Plain words for a platform audit action; unknown ones show as recorded. */
export function actionLabel(action: string): string {
  return ACTION_LABEL[action] ?? action;
}

export type Tone = "success" | "warning" | "danger" | "info" | "neutral";

export function healthTone(status: string): Tone {
  if (status === "healthy") return "success";
  if (status === "degraded") return "warning";
  if (status === "unhealthy") return "danger";
  return "neutral";
}

/** Bytes as the console shows memory: KB, MB or GB, one decimal past MB. */
export function formatBytes(bytes: number | undefined | null): string {
  if (bytes === undefined || bytes === null) return "—";
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

/** Seconds of uptime as "3d 4h", "5h 12m" or "8m". */
export function formatUptime(seconds: number | undefined): string {
  if (!seconds || seconds < 0) return "—";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  return `${Math.max(1, minutes)}m`;
}

const SERVICE_LABEL: Record<string, string> = {
  "gateway-v2": "Gateway",
  web: "Customer dashboard",
  "control-api": "Control API",
  worker: "Worker",
  postgres: "PostgreSQL",
  redis: "Redis",
};

export function serviceLabel(service: string): string {
  return SERVICE_LABEL[service] ?? service;
}
