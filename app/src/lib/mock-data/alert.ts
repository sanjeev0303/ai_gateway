import { NOW } from "./clock";
import { users } from "./users";
import type { Alert } from "@/types";

const minutesAgo = (m: number) =>
  new Date(NOW.getTime() - m * 60_000).toISOString();

/** Only real user ids may appear in notification and acknowledgement fields. */
const userIds = (...names: string[]) =>
  names.map((name) => {
    const match = users.find((user) => user.name === name);
    if (!match) throw new Error(`alert references unknown user: ${name}`);
    return match.id;
  });

const PLATFORM_TEAM = ["Fatima Al-Rashid", "Priya Sharma"];
const DATA_TEAM = ["Sofia Marchetti", "Ravi Deshpande"];
const LEGAL_TEAM = ["Beatriz Almeida", "Nathaniel Boyd"];
const SUPPORT_TEAM = ["Amara Nwosu", "Jonas Berg"];

export const alerts: Alert[] = [
  {
    id: "alt_1",
    occurredAt: minutesAgo(8),
    severity: "critical",
    state: "open",
    category: "provider_outage",
    title: "OpenAI error rate above 12% for 8 minutes",
    description:
      "Upstream 5xx rate on openai exceeded 12% against a 2% baseline. Retry amplification is making recovery slower than the incident itself.",
    link: "/app/observe/providers",
    notifiedUserIds: userIds(...PLATFORM_TEAM),
  },
  {
    id: "alt_2",
    occurredAt: minutesAgo(34),
    severity: "high",
    state: "open",
    category: "policy_violation",
    title: "Prompt injection blocked 412 times in Internal Tools",
    description:
      "Internal Tools is triggering the injection detector far above its 30-day average, concentrated in a single retrieval index.",
    link: "/app/govern/policies",
    notifiedUserIds: userIds(...DATA_TEAM, "Fatima Al-Rashid"),
  },
  {
    id: "alt_3",
    occurredAt: minutesAgo(96),
    severity: "high",
    state: "acknowledged",
    category: "budget_exceeded",
    title: "Marketing at 99% of monthly budget",
    description:
      "Marketing has consumed $1,190 of a $1,200 monthly budget with four days remaining in the period.",
    link: "/app/govern/budgets",
    notifiedUserIds: userIds("Marcus Bell", "Priya Sharma"),
    acknowledgedAt: minutesAgo(80),
    acknowledgedById: userIds("Marcus Bell")[0]!,
  },
  {
    id: "alt_4",
    occurredAt: minutesAgo(150),
    severity: "critical",
    state: "open",
    category: "credential_expiry",
    title: "Cohere key expires in 4 days",
    description:
      "The Cohere production key expires on 3 Oct. Two projects depend on it and no rotation has been scheduled.",
    link: "/app/govern/credentials",
    notifiedUserIds: userIds(...PLATFORM_TEAM),
  },
  {
    id: "alt_5",
    occurredAt: minutesAgo(220),
    severity: "medium",
    state: "open",
    category: "cost_anomaly",
    title: "Daily spend 41% above trailing baseline",
    description:
      "Spend for 29 Sep is tracking 41% above the trailing 7-day mean, driven by o3 preview traffic from the eval harness.",
    link: "/app/observe/costs",
    notifiedUserIds: userIds("Priya Sharma", "Michael Chen"),
  },
  {
    id: "alt_6",
    occurredAt: minutesAgo(310),
    severity: "high",
    state: "open",
    category: "policy_violation",
    title: "EU residency policy blocked 74 EU requests",
    description:
      "Requests carrying EU-resident data were blocked by the residency policy. This is expected enforcement, but the volume suggests a routing misconfiguration upstream.",
    link: "/app/govern/policies",
    notifiedUserIds: userIds(...LEGAL_TEAM),
  },
  {
    id: "alt_7",
    occurredAt: minutesAgo(420),
    severity: "medium",
    state: "resolved",
    category: "error_rate",
    title: "Mistral small error rate spiked to 6%",
    description:
      "A malformed tool-call schema in segment-builder caused elevated parse failures on mistral-small-4.",
    link: "/app/observe/models",
    notifiedUserIds: userIds(...DATA_TEAM),
    acknowledgedAt: minutesAgo(400),
    acknowledgedById: userIds("Ravi Deshpande")[0]!,
    resolvedAt: minutesAgo(360),
  },
  {
    id: "alt_8",
    occurredAt: minutesAgo(515),
    severity: "low",
    state: "resolved",
    category: "unusual_access",
    title: "Export of 12,400 conversation rows",
    description:
      "A bulk export of conversation records was performed outside the normal monthly reporting job.",
    link: "/app/security/audit-logs",
    notifiedUserIds: userIds(...PLATFORM_TEAM),
    acknowledgedAt: minutesAgo(505),
    acknowledgedById: userIds("Priya Sharma")[0]!,
    resolvedAt: minutesAgo(470),
  },
  {
    id: "alt_9",
    occurredAt: minutesAgo(690),
    severity: "medium",
    state: "resolved",
    category: "provider_outage",
    title: "Cohere maintenance window degraded embeddings",
    description:
      "Embedding calls failed during a scheduled Cohere maintenance window. Traffic failed over to a backup index.",
    link: "/app/observe/providers",
    notifiedUserIds: userIds(...DATA_TEAM),
    acknowledgedAt: minutesAgo(680),
    acknowledgedById: userIds("Sofia Marchetti")[0]!,
    resolvedAt: minutesAgo(640),
  },
  {
    id: "alt_10",
    occurredAt: minutesAgo(880),
    severity: "high",
    state: "resolved",
    category: "budget_exceeded",
    title: "Customer Portal hit its hard stop",
    description:
      "Customer Portal exhausted its $1,500 monthly budget and was hard-stopped. Customer-facing traffic was affected for 11 minutes.",
    link: "/app/govern/budgets",
    notifiedUserIds: userIds(...SUPPORT_TEAM),
    acknowledgedAt: minutesAgo(872),
    acknowledgedById: userIds("Amara Nwosu")[0]!,
    resolvedAt: minutesAgo(840),
  },
  {
    id: "alt_11",
    occurredAt: minutesAgo(1_020),
    severity: "medium",
    state: "resolved",
    category: "cost_anomaly",
    title: "Claude Opus spend tripled week on week",
    description:
      "Opus spend rose from $410 to $1,180 driven by contract-review. The traffic pattern changed rather than the pricing.",
    link: "/app/observe/costs",
    notifiedUserIds: userIds(...LEGAL_TEAM),
    acknowledgedAt: minutesAgo(1_010),
    acknowledgedById: userIds("Beatriz Almeida")[0]!,
    resolvedAt: minutesAgo(960),
  },
  {
    id: "alt_12",
    occurredAt: minutesAgo(1_340),
    severity: "low",
    state: "resolved",
    category: "policy_violation",
    title: "Deprecated model accessed outside allowlist",
    description:
      "gpt-4o was called by a project whose allowlist excludes it. The request succeeded because no allowlist policy is bound to that project.",
    link: "/app/govern/policies",
    notifiedUserIds: userIds(...PLATFORM_TEAM),
    acknowledgedAt: minutesAgo(1_330),
    acknowledgedById: userIds("Michael Chen")[0]!,
    resolvedAt: minutesAgo(1_290),
  },
  {
    id: "alt_13",
    occurredAt: minutesAgo(1_680),
    severity: "critical",
    state: "resolved",
    category: "unusual_access",
    title: "API key used from a new country",
    description:
      "A production API key was first seen from a new IP country, 4 hours after a credential rotation.",
    link: "/app/security/audit-logs",
    notifiedUserIds: userIds(...PLATFORM_TEAM, ...LEGAL_TEAM),
    acknowledgedAt: minutesAgo(1_676),
    acknowledgedById: userIds("Priya Sharma")[0]!,
    resolvedAt: minutesAgo(1_600),
  },
  {
    id: "alt_14",
    occurredAt: minutesAgo(2_010),
    severity: "low",
    state: "resolved",
    category: "credential_expiry",
    title: "Contract review OAuth app expired",
    description:
      "The Legal contract-review OAuth grant expired. It has since been re-authorised; no requests failed in the interim.",
    link: "/app/govern/credentials",
    notifiedUserIds: userIds(...LEGAL_TEAM),
    acknowledgedAt: minutesAgo(2_000),
    acknowledgedById: userIds("Nathaniel Boyd")[0]!,
    resolvedAt: minutesAgo(1_950),
  },
];
