import { NOW } from "./clock";
import type { PlatformEvent, PlatformHealth, TimeSeriesPoint } from "@/types";

const MINUTE = 60_000;
const HOUR = 3_600_000;
const DAY = 86_400_000;

const daysAgo = (n: number) => new Date(NOW.getTime() - n * DAY).toISOString();
const hoursAgo = (n: number) => new Date(NOW.getTime() - n * HOUR).toISOString();
const minutesAgo = (n: number) => new Date(NOW.getTime() - n * MINUTE).toISOString();

/**
 * Rolling 30d gateway request volume. Generated with a fixed formula rather
 * than a random source so every render, and every test, sees the same series.
 */
const requestVolume: TimeSeriesPoint[] = Array.from({ length: 30 }, (_, i) => {
  const daysBack = 29 - i;
  const date = new Date(NOW.getTime() - daysBack * DAY);
  const weekday = date.getUTCDay();
  const weekend = weekday === 0 || weekday === 6;
  // Gentle upward trend plus a deterministic wobble for a believable curve.
  const trend = 52_000 + i * 940;
  const wobble = ((i * 7_919) % 7_300) - 3_600;
  return {
    date: date.toISOString().slice(0, 10),
    value: Math.round((weekend ? trend * 0.58 : trend) + wobble),
  };
});

export const platformHealth: PlatformHealth = {
  status: "operational",
  apiLatencyP50Ms: 142,
  apiLatencyP95Ms: 388,
  apiErrorRate: 0.0021,
  overallScore: 99,
  pipelines: [
    {
      component: "Request ingest",
      status: "operational",
      freshnessSlaPercent: 99.8,
      ingestLagP50Seconds: 1.4,
      eventsPerSecond: 1_284,
    },
    {
      component: "Cost attribution",
      status: "operational",
      freshnessSlaPercent: 99.6,
      ingestLagP50Seconds: 4.8,
      eventsPerSecond: 1_284,
    },
    {
      component: "Policy engine",
      status: "operational",
      freshnessSlaPercent: 100,
      ingestLagP50Seconds: 0.6,
      eventsPerSecond: 1_284,
    },
    {
      component: "Audit writer",
      status: "degraded",
      freshnessSlaPercent: 97.1,
      ingestLagP50Seconds: 46,
      eventsPerSecond: 986,
    },
  ],
  requestVolume,
  updatedAt: minutesAgo(2),
};

export const platformEvents: PlatformEvent[] = [
  {
    id: "pev_1",
    occurredAt: minutesAgo(2),
    severity: "info",
    source: "Status page",
    message: "All gateway pipelines reporting normal freshness.",
    link: "/app/platform/provider-health",
  },
  {
    id: "pev_2",
    occurredAt: minutesAgo(41),
    severity: "warning",
    source: "Audit writer",
    message: "Audit write queue backed up to 46s median lag; no events lost.",
    link: "/app/security/audit-logs",
  },
  {
    id: "pev_3",
    occurredAt: hoursAgo(3),
    severity: "info",
    source: "Provider catalog",
    message: "Catalog sync added mdl_nova_pro and refreshed Bedrock pricing.",
    link: "/app/observe/models",
  },
  {
    id: "pev_4",
    occurredAt: hoursAgo(7),
    severity: "info",
    source: "Policy engine",
    message: "Guardrail grd_4 reloaded with threshold 0.62.",
    link: "/app/govern/guardrails",
  },
  {
    id: "pev_5",
    occurredAt: hoursAgo(11),
    severity: "error",
    source: "Failover",
    message: "Upstream Anthropic returned elevated 529s; traffic failed over to OpenAI.",
    link: "/app/observe/providers",
  },
  {
    id: "pev_6",
    occurredAt: daysAgo(1),
    severity: "warning",
    source: "Budget engine",
    message: "Project prj_2 crossed 90% of its monthly budget.",
    link: "/app/govern/budgets",
  },
  {
    id: "pev_7",
    occurredAt: daysAgo(2),
    severity: "info",
    source: "Schema migration",
    message: "Added modality column to conversation turn analytics.",
    link: "/app/observe/conversations",
  },
  {
    id: "pev_8",
    occurredAt: daysAgo(3),
    severity: "info",
    source: "SSO",
    message: "SCIM sync completed for 24 members; 0 deprovisioned.",
    link: "/app/security/events",
  },
  {
    id: "pev_9",
    occurredAt: daysAgo(4),
    severity: "error",
    source: "Credentials",
    message: "Credential for AWS Bedrock rejected on refresh; rotation required.",
    link: "/app/govern/credentials",
  },
  {
    id: "pev_10",
    occurredAt: daysAgo(5),
    severity: "info",
    source: "Status page",
    message: "Weekly uptime report: 99.98% across all providers.",
    link: "/app/platform/provider-health",
  },
];
