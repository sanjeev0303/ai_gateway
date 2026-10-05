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

export const metadata: Metadata = {
  title: "Provider health · Sentinel",
  description: "Result of the last health probe for each upstream provider.",
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
  id: string;
  provider: string;
  status: string;
  tone: StatusTone;
  detail: string;
  checked: string;
  checkedSort: string;
}

const COLUMNS: readonly Column<HealthRow>[] = [
  { key: "provider", header: "Provider", primary: "provider", sort: "provider" },
  { key: "status", header: "Status", primary: "status", tone: "tone", sort: "status" },
  { key: "detail", header: "Reason", primary: "detail" },
  {
    key: "checked",
    header: "Last probe",
    primary: "checked",
    sort: "checkedSort",
    align: "right",
  },
];

async function loadProviderHealth() {
  const { providers } = await getApiClient();
  const list = await providers.health();
  const now = new Date();

  const rows: HealthRow[] = [...list.items]
    .sort((a, b) => a.provider_kind.localeCompare(b.provider_kind))
    .map((health) => ({
      id: health.provider_kind,
      provider: display(health.provider_kind),
      status: health.status,
      tone: toneFor(HEALTH_TONE, health.status),
      detail: health.last_error ?? health.status_reason ?? "—",
      checked: formatRelative(health.checked_at, now),
      checkedSort: health.checked_at,
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

        <Card>
          <CardHeader>
            <CardTitle>Probe results</CardTitle>
            <CardDescription>
              One row per provider, reflecting the latest probe rather than a
              rolling window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No provider health has been recorded yet."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Configured providers and backend reachability</CardTitle>
            <CardDescription>Not recorded by the control API today.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              A health record holds one probe outcome, its reason, and when it ran.
              It carries no uptime percentage, no latency percentile, and no error
              rate, so this page cannot show a composite score or a latency trend.
            </p>
            <p>
              Request volume and ingest freshness are not exposed either. Cost and
              request counts per provider are available under Observe.
            </p>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
