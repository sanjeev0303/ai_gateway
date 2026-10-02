import type { Metadata } from "next";

import { LineChart, chartColor } from "@/components/charts";
import {
  DataTable,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatMs, formatPercent } from "@/lib/format";
import { platformHealth, providers } from "@/lib/mock-data";
import {
  PIPELINE_STATUS_TONE,
  PROVIDER_STATUS_TONE,
  type StatusTone,
} from "@/types";

export const metadata: Metadata = {
  title: "Provider health · Sentinel",
  description: "Gateway uptime, latency, and pipeline freshness by component.",
};

interface PipelineRow {
  id: string;
  component: string;
  status: string;
  tone: StatusTone;
  sla: string;
  lag: string;
  eps: string;
}

const PIPELINE_COLUMNS: readonly Column<PipelineRow>[] = [
  { key: "component", header: "Component", primary: "component" },
  { key: "status", header: "Status", primary: "status", tone: "tone" },
  { key: "sla", header: "Freshness SLA", primary: "sla", align: "right" },
  { key: "lag", header: "Ingest lag p50", primary: "lag", align: "right" },
  { key: "eps", header: "Events/s", primary: "eps", align: "right" },
];

export default function ProviderHealthPage() {
  const pipelineRows: PipelineRow[] = platformHealth.pipelines.map((pipeline) => ({
    id: pipeline.component,
    component: pipeline.component,
    status: pipeline.status.replace(/_/g, " "),
    tone: PIPELINE_STATUS_TONE[pipeline.status],
    sla: formatPercent(pipeline.freshnessSlaPercent),
    lag: formatMs(pipeline.ingestLagP50Seconds * 1000),
    eps: pipeline.eventsPerSecond.toLocaleString("en-US"),
  }));

  const worstProvider = [...providers].sort(
    (a, b) => a.uptime30d - b.uptime30d,
  )[0];

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Provider health"
        description="Gateway uptime and the telemetry pipelines behind it, refreshed continuously."
        eyebrow={
          <span className="flex items-center gap-2 text-xs text-muted-foreground">
            <StatusBadge tone="success">Operational</StatusBadge>
            <span>Updated {platformHealth.updatedAt.slice(11, 16)} UTC</span>
          </span>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Gateway uptime"
            value={formatPercent(platformHealth.overallScore)}
            tone="success"
            hint="Composite health score"
          />
          <StatCard
            label="API latency p50"
            value={formatMs(platformHealth.apiLatencyP50Ms)}
            hint="Median across all regions"
          />
          <StatCard
            label="API latency p95"
            value={formatMs(platformHealth.apiLatencyP95Ms)}
            hint="Tail latency budget"
          />
          <StatCard
            label="Error rate"
            value={formatPercent(platformHealth.apiErrorRate, 2)}
            tone="success"
            hint="5xx and gateway rejections"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Gateway request volume</CardTitle>
            <CardDescription>Rolling 30-day volume through the control plane.</CardDescription>
          </CardHeader>
          <CardContent>
            <LineChart
              data={platformHealth.requestVolume}
              color={chartColor(1)}
              height={260}
              formatValue={(v) => v.toLocaleString("en-US")}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Telemetry pipelines</CardTitle>
            <CardDescription>
              Ingest freshness against SLA, with median lag from call to dashboard.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={PIPELINE_COLUMNS}
              rows={pipelineRows}
              rowKey="id"
              emptyMessage="No pipelines reported."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Upstream providers</CardTitle>
            <CardDescription>
              Trailing 30-day uptime per provider. Weakest: {worstProvider.displayName}.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {providers.map((provider) => (
                <li
                  key={provider.id}
                  className="rounded-lg border p-3 text-sm"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-medium">{provider.displayName}</span>
                    <StatusBadge tone={PROVIDER_STATUS_TONE[provider.status]}>
                      {provider.status}
                    </StatusBadge>
                  </div>
                  <p className="mt-1.5 text-xs text-muted-foreground">
                    {formatPercent(provider.uptime30d)} uptime · p50{" "}
                    {formatMs(provider.latencyP50Ms)} · p95{" "}
                    {formatMs(provider.latencyP95Ms)}
                  </p>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
