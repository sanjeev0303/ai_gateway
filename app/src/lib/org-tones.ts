import type { OrganizationRole, StatusTone } from "@/types";

/** Roles as the control API returns them, most privileged first. */
export const ROLE_ORDER: readonly OrganizationRole[] = [
  "owner",
  "admin",
  "billing_admin",
  "auditor",
  "member",
];

export const ROLE_TONE: Record<string, StatusTone> = {
  owner: "info",
  admin: "info",
  billing_admin: "neutral",
  auditor: "neutral",
  member: "neutral",
};

/** Member status as the control API returns it: active or disabled. */
export const STATUS_TONE: Record<string, StatusTone> = {
  active: "success",
  disabled: "warning",
};

export const toneFor = (map: Record<string, StatusTone>, key: string): StatusTone =>
  map[key] ?? "neutral";
