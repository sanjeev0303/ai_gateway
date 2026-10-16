import type { Metadata } from "next";

import {
  DataTable,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatRelative } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { ProviderHealthClient, ProbeProviderDialog } from "./client";

export const metadata: Metadata = {
  title: "Provider health · Sentinel",
  description: "Result of the last health probe for each upstream provider, with on-demand checking.",
};

const HEALTH_TONE: Record<string, StatusTone> = {
  healthy: "success",
  degraded: "warning",
  unhealthy: "danger",
};

const display = (kind: string) =>
  kind
    .split("_")
    .map((part) => (part ? part[0].toUpperCase() + part.slice(1) : part))
    .join(" ");

interface HealthRow {
  provider_kind: string;
  status: string;
  status_reason?: string;
  last_error?: string;
  checked_at: string;
}

async function loadProviderHealth() {
  const { providers } = await getApiClient();
  const list = await providers.health();

  const rows: HealthRow[] = [...list.items]
    .sort((a, b) => a.provider_kind.localeCompare(b.provider_kind))
    .map((health) => ({
      provider_kind: health.provider_kind,
      status: health.status,
      status_reason: health.status_reason,
      last_error: health.last_error,
      checked_at: health.checked_at,
    }));

  const count = (status: string) => rows.filter((row) => row.status === status).length;

  return {
    rows,
    probed: rows.length,
    healthy: count("healthy"),
    degraded: count("degraded"),
    unhealthy: count("unhealthy"),
  };
}

export default async function ProviderHealthPage() {
  const { rows, probed, healthy, degraded, unhealthy } = await loadProviderHealth();

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Provider health"
        description="Outcome of the most recent probe of each upstream provider, run by the background worker."
        actions={<ProbeProviderDialog />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Providers probed"
            value={formatNumber(probed)}
            hint="Distinct upstream providers"
          />
          <StatCard
            label="Healthy"
            value={formatNumber(healthy)}
            tone={probed === healthy ? "success" : "neutral"}
            hint="Last probe succeeded"
          />
          <StatCard
            label="Degraded"
            value={formatNumber(degraded)}
            tone={degraded > 0 ? "warning" : "success"}
            hint="Reachable, but impaired"
          />
          <StatCard
            label="Unhealthy"
            value={formatNumber(unhealthy)}
            tone={unhealthy > 0 ? "danger" : "success"}
            hint="Last probe failed"
          />
        </section>

        <ProviderHealthClient initialRows={rows} />
      </PageShell>
    </div>
  );
}
