import { NOW } from "./clock";
import { users } from "./users";
import type { AuditAction, AuditActorType, AuditLogEntry } from "@/types";

const minutesAgo = (m: number) =>
  new Date(NOW.getTime() - m * 60_000).toISOString();

function mulberry32(seed: number) {
  let a = seed;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

const rand = mulberry32(88117);
const pick = <T,>(items: readonly T[]): T =>
  items[Math.floor(rand() * items.length)]!;

const ACTIVE_USERS = users.filter(
  (user) => user.status === "active" || user.status === "invited",
);

const USER_AGENTS = [
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36",
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0 Safari/537.36",
  "Sentinel-CLI/2.4.1 (linux; x86_64)",
  "Sentinel-SDK-Python/1.9.0",
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
] as const;

const IP_COUNTRIES = [
  "GB",
  "US",
  "DE",
  "NL",
  "PT",
  "IN",
  "CA",
  "SG",
  "BR",
  "ES",
] as const;

const API_KEY_NAMES = [
  "gateway-prod-key",
  "eval-runner",
  "billing-reconciler",
  "status-page-bot",
] as const;

/** Resources are drawn from real fixture ids so the paths are navigable. */
const RESOURCE_KINDS = [
  { kind: "policy", ids: ["pol_1", "pol_3", "pol_4", "pol_6", "pol_8", "pol_10"] },
  { kind: "project", ids: ["prj_1", "prj_2", "prj_4", "prj_6", "prj_9", "prj_13"] },
  { kind: "credential", ids: ["cred_1", "cred_2", "cred_4", "cred_5", "cred_11"] },
  { kind: "budget", ids: ["bud_1", "bud_2", "bud_3", "bud_6", "bud_8"] },
  { kind: "model", ids: ["mdl_claude_sonnet", "mdl_gpt5", "mdl_gpt4o", "mdl_o3"] },
  { kind: "user", ids: ["usr_5", "usr_11", "usr_15", "usr_21"] },
] as const;

const DETAILS: Record<AuditAction, readonly string[]> = {
  create: [
    "Created resource and bound it to the default organisation scope.",
    "Created from the console form; all fields set to defaults.",
  ],
  update: [
    "Updated threshold from 0.7 to 0.78 after reviewing false positives.",
    "Changed description only; enforcement and scope unchanged.",
    "Raised monthly limit following a spend review.",
  ],
  delete: [
    "Deleted an unused draft left over from the previous quarter.",
  ],
  login: [
    "Interactive console session established.",
    "Authenticated with SSO and completed the second factor.",
  ],
  logout: ["Session ended by the user."],
  export: [
    "Exported 12,400 rows of conversation metadata as CSV.",
    "Exported the audit trail for the Q3 compliance review.",
  ],
  policy_change: [
    "Switched enforcement from audit to enforce following the calibration window.",
    "Temporarily disabled while the recall regression was investigated.",
    "Re-enabled after the false-positive rate fell below 0.4%.",
  ],
  permission_change: [
    "Granted workspace admin to a new team lead.",
    "Revoked project access on offboarding.",
    "Suspended an account after repeated failed authentications.",
  ],
  credential_use: [
    "Credential read by the nightly reconciliation job.",
    "Credential used for a provider call with no active session.",
  ],
};

const ACTIONS: readonly AuditAction[] = [
  "login",
  "login",
  "update",
  "credential_use",
  "credential_use",
  "export",
  "policy_change",
  "permission_change",
  "create",
  "delete",
  "logout",
  "update",
];

const octet = () => Math.floor(rand() * 254) + 1;

export const auditLogs: AuditLogEntry[] = Array.from({ length: 80 }, (_, i) => {
  const action = pick(ACTIONS);
  const resourceKind = pick(RESOURCE_KINDS);
  const resourceId = pick(resourceKind.ids);

  // Most entries are human; service actors cover automation and job activity.
  const actorType: AuditActorType =
    action === "credential_use"
      ? pick(["api_key", "api_key", "system"] as const)
      : pick(["user", "user", "user", "user", "system"] as const);

  const human = actorType === "user" ? pick(ACTIVE_USERS) : null;
  const apiKeyName = actorType === "api_key" ? pick(API_KEY_NAMES) : null;

  const actorId =
    human?.id ?? (apiKeyName ? `key_${apiKeyName}` : "sys_sentinel");

  return {
    id: `aud_${String(i + 1).padStart(4, "0")}`,
    occurredAt: minutesAgo(i * 37 + Math.round(rand() * 20)),
    actorType,
    actorId,
    actorName: human?.name ?? apiKeyName ?? "Sentinel Control Plane",
    ...(human ? { actorEmail: human.email } : {}),
    ipAddress: `${pick(["81.2", "104.16", "185.60", "34.120", "89.187"])}.${octet()}.${octet()}.${octet()}`,
    userAgent: pick(USER_AGENTS),
    action,
    resource: `${resourceKind.kind}/${resourceId}`,
    details: pick(DETAILS[action]),
    ipCountry: pick(IP_COUNTRIES),
    mfaVerified: actorType === "user" && action !== "logout" && rand() > 0.12,
  };
});
