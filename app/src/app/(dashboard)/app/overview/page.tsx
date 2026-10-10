

import { type ReactNode, Suspense } from "react";
import {
  AreaChart,
  BarChart,
  DonutChart,
  chartColor,
} from "@/components/charts";
import {
  DataTable,
  EmptyState,
  PageShell,
  StatCard,
  StatusBadge,
  type Column,
} from "@/components/dashboard";
import {
  formatCompact,
  formatMs,
  formatPercent,
  formatUsd,
  formatNumber,
  microsToUsd,
} from "@/lib/format";

import { getApiClient } from "@/lib/session";
import type {
  TimeSeriesPoint,
  CategoryDatum,
  StatusTone,
  TrendDirection,
} from "@/types";
import { RangeTabs } from "@/components/dashboard/range-tabs";
import { isDateRange, type DateRange } from "@/lib/date-range";

type Trend = { direction: TrendDirection; fraction: number };

/** A direction with no comparable prior window reports nothing rather than 0%. */
function periodChange(series: readonly TimeSeriesPoint[]): Trend | undefined {
  if (series.length < 4) return undefined;
  const half = Math.floor(series.length / 2);
  let prior = 0;
  let current = 0;
  for (const point of series.slice(0, half)) prior += point.value;
  for (const point of series.slice(half)) current += point.value;
  if (prior === 0) return undefined;
  const fraction = (current - prior) / prior;
  return {
    direction: fraction > 0 ? "up" : fraction < 0 ? "down" : "flat",
    fraction,
  };
}

function errorTone(rate: number): StatusTone {
  if (rate >= 0.05) return "danger";
  if (rate >= 0.02) return "warning";
  return "success";
}

interface ProjectRow {
  id: string;
  name: string;
  requests: number;
  cost: number;
  errorRate: number;
  share: number;
}

/** ------------------------------------------------------------------- */
/* STATE B – ZERO TRAFFIC SETUP STEPS                                    */
/** ------------------------------------------------------------------- */

function SetupStep({
  label,
  description,
  href,
  ctaLabel = "Go",
}: {
  label: string;
  description: string;
  href: string;
  ctaLabel?: string;
}) {
  return (
    <div className="space-y-3 p-4 rounded-xl border bg-card hover:bg-muted/50 transition-colors">
      <h3 className="text-sm font-medium">{label}</h3>
      <p className="text-xs text-muted-foreground">{description}</p>
      <a
        href={href}
        className="mt-2 inline-flex items-center gap-2 text-primary font-medium hover:underline"
      >
        {ctaLabel}
        <svg
          className="size-4"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
        >
          <path d="M5 12h14M12 5l7 7-7 7" />
        </svg>
      </a>
    </div>
  );
}

function ZeroTrafficState() {
  return (
    <div className="space-y-6">
      <h2 className="text-xl font-semibold">Get Sentinel running</h2>
      <p className="text-muted-foreground">
        You're all set. Let's wire up your first connections so usage data starts
        flowing.
      </p>

      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        Step 1: Connect an AI provider
      </h3>
      <SetupStep
        label="Connect AI provider"
        description="Add upstream provider accounts (OpenAI, Anthropic, Gemini, etc.)"
        href="/app/platform/provider-accounts"
        ctaLabel="Add provider"
      />

      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        Step 2: Create a project
      </h3>
      <SetupStep
        label="Create a project"
        description="Organize usage tracking by project for cost attribution"
        href="/app/organization/projects"
        ctaLabel="Create project"
      />

      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        Step 3: Create gateway credential
      </h3>
      <SetupStep
        label="Create gateway credential"
        description="Set up API keys and authentication for your gateway"
        href="/app/platform/gateway-credentials"
        ctaLabel="Add credential"
      />

      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        Step 4: Configure model access
      </h3>
      <SetupStep
        label="Configure model access"
        description="Grant your organization access to specific models"
        href="/app/govern/model-access"
        ctaLabel="Set access"
      />

      <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
        Step 5: Send your first request
      </h3>
      <SetupStep
        label="Send your first gateway request"
        description="Route traffic through the gateway and see usage appear in Sentinel"
        href="/app/developer/quick-start"
        ctaLabel="Quick start"
      />

      <section className="mt-6 p-4 rounded-xl border bg-muted/50">
        <h3 className="text-sm font-medium text-muted-foreground">No traffic yet</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Usage metrics (requests, tokens, spend, error rate, latency) will appear
          after your first gateway request. Provider health data requires
          background probe runs.
        </p>
      </section>
    </div>
  );
}

/** ------------------------------------------------------------------- */
/* STATE A – ORGANIZATION HAS TRAFFIC                                    */
/** ------------------------------------------------------------------- */

function TrafficState({
  summary,
  latency,
  series,
  byProject,
  byModel,
  byProvider,
  byTeam,
  accounts,
  health,
  actions,
}: {
  summary: any;
  latency: any;
  series: any;
  byProject: any;
  byModel: any;
  byProvider: any;
  byTeam: any;
  accounts: any;
  health: any;
  actions?: ReactNode;
}) {
  const range: DateRange = "30d";
  const RANGE_MS: Record<DateRange, number> = {
    "24h": 24 * 60 * 60 * 1000,
    "7d": 7 * 24 * 60 * 60 * 1000,
    "30d": 30 * 24 * 60 * 60 * 1000,
    "90d": 90 * 24 * 60 * 60 * 1000,
  };
  const RANGE_HINT: Record<DateRange, string> = {
    "24h": "Last 24 hours",
    "7d": "Last 7 days",
    "30d": "Last 30 days",
    "90d": "Last 90 days",
  };

  const to = new Date();
  const query = { from: new Date(to.getTime() - RANGE_MS[range]), to };

  const totalSpend = microsToUsd(summary.TotalCostMicros);
  const errorRate =
    summary.Requests > 0 ? summary.Errors / summary.Requests : 0;
  // Find overall (no provider/model qualifier) p95/p50 latency
  const overall = latency.items.find(
    (row: {
      P95TotalMs?: number;
      P50TotalMs?: number;
      Provider?: string;
      Model?: string;
    }) => !row.Provider && !row.Model,
  );
  const p95 = overall?.P95TotalMs;
  const p50 = overall?.P50TotalMs;

  // --- Inline trend calculation ---
  const spendSeries = series.items.map((point: any) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: microsToUsd(point.CostMicros),
  }));
  const requestSeries = series.items.map((point: any) => ({
    date: point.BucketStart.slice(5, 16).replace("T", " "),
    value: point.Requests,
  }));

  const spendTrend = periodChange(spendSeries);
  const requestTrend = periodChange(requestSeries);

  // --- KPI cards ---
  const stats = [
    {
      label: "Spend",
      value: formatUsd(totalSpend),
      hint: `${formatCompact(summary.TotalTokens)} tokens`,
      sparkline: spendSeries,
      delta: spendTrend,
      href: "/app/observe/costs",
    },
    {
      label: "Requests",
      value: formatCompact(summary.Requests),
      hint: `${formatCompact(summary.TotalTokens)} tokens`,
      sparkline: requestSeries,
      delta: requestTrend,
      href: "/app/observe/usage",
    },
    {
      label: "Error rate",
      value: formatPercent(errorRate, 2),
      hint: `${formatCompact(summary.Errors)} failed`,
      tone: errorTone(errorRate),
      href: "/app/observe/usage",
    },
    {
      label: "Latency (p95)",
      value:
        p95 === undefined ? "n/a" : formatMs(p95),
      hint:
        p50 === undefined ? "No samples yet" : `p50 ${formatMs(p50)}`,
    },
  ];

  // --- Top models ---
  const topModels: CategoryDatum[] = [...byModel.items]
    .sort((a: { CostMicros: number }, b: { CostMicros: number }) =>
      b.CostMicros - a.CostMicros,
    )
    .slice(0, 6)
    .map((row) => ({
      label: row.Model,
      value: microsToUsd(row.CostMicros),
    }));

  // --- Project table rows ---
  const projectRows: ProjectRow[] = byProject.items.map((row: {
    ProjectID: string;
    Name: string;
    Requests: number;
    CostMicros: number;
    Errors: number;
  }) => ({
    id: row.ProjectID,
    name: row.Name,
    requests: row.Requests,
    cost: microsToUsd(row.CostMicros),
    errorRate: row.Requests > 0 ? row.Errors / row.Requests : 0,
    share:
      totalSpend > 0 ? microsToUsd(row.CostMicros) / totalSpend : 0,
  }));

  // --- Provider health counts ---
  // health.items has status and optionally last_error / status_reason
  const healthy = health.items.filter(
    (row: { status: string }) => row.status === "healthy" || row.status === "ok",
  ).length;
  const degraded = health.items.filter(
    (row: { status: string; last_error?: string }) =>
      (row.status !== "healthy" && row.status !== "ok") ||
      Boolean(row.last_error),
  ).length;
  const unhealthy = health.items.filter(
    (row: { status: string }) => row.status === "down" || row.status === "error",
  ).length;



  const columns: Column<ProjectRow>[] = [
    { key: "name", header: "Project", primary: "name", sort: "name" },
    {
      key: "requests",
      header: "Requests",
      primary: "requests",
      align: "right",
      format: "compact",
    },
    { key: "cost", header: "Spend", primary: "cost", align: "right", format: "usd" },
    {
      key: "errorRate",
      header: "Error rate",
      primary: "errorRate",
      align: "right",
      format: "percent",
    },
    {
      key: "share",
      header: "Share",
      primary: "share",
      align: "right",
      format: "percent",
      meter: "share",
      meterMax: 1,
    },
  ];

  return (
    <PageShell
      title="Overview"
      description="Requests, spend, error rate, and latency recorded for the selected window."
      actions={actions}
    >
      <div className="space-y-6">
        {/* KPI row */}
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {stats.map((stat, i) => (
            <StatCard key={stat.label} {...stat} />
          ))}
        </div>

        {/* Charts row */}
        <div className="grid gap-6 xl:grid-cols-3">
          <section className="space-y-3 xl:col-span-2">
            <h2 className="text-sm font-medium">Spend over time</h2>
            <AreaChart
              series={[
                {
                  label: "Spend",
                  color: chartColor(0),
                  data: spendSeries,
                },
              ]}
              formatValue={formatUsd}
            />
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-medium">Request outcomes</h2>
            <DonutChart
              data={[
                { label: "Succeeded", value: Math.max(0, summary.Requests - summary.Errors) },
                { label: "Failed", value: summary.Errors },
              ]}
              centerLabel="requests"
            />
          </section>
        </div>

        {/* Top models + Data table row */}
        <section className="grid gap-6">
          <section className="space-y-3">
            <h2 className="text-sm font-medium">Top models by spend</h2>
            {topModels.length === 0 ? (
              <EmptyState
                title="No model spend yet"
                description="Model costs appear here as usage lands."
              />
            ) : (
              <BarChart data={topModels} formatValue={formatUsd} />
            )}
          </section>

          <section className="space-y-3">
            <h2 className="text-sm font-medium">Project spend</h2>
            <DataTable
              columns={columns}
              rows={projectRows}
              rowKey="id"
              pageSize={8}
              emptyMessage="No project spend in this window."
            />
          </section>
        </section>
      </div>
    </PageShell>
  );
}

export default async function OverviewPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const client = await getApiClient();

  const RANGE_MS: Record<DateRange, number> = {
    "24h": 24 * 60 * 60 * 1000,
    "7d": 7 * 24 * 60 * 60 * 1000,
    "30d": 30 * 24 * 60 * 60 * 1000,
    "90d": 90 * 24 * 60 * 60 * 1000,
  };
  const to = new Date();
  const from = new Date(to.getTime() - RANGE_MS[range]);
  const timeParams = { from, to };

  // Load all data in parallel
  const [
    summary,
    latency,
    series,
    byProject,
    byModel,
    byProvider,
    byTeam,
    accounts,
    health,
  ] = await Promise.all([
    client.usage.summary(timeParams),
    client.usage.latency(timeParams),
    client.usage.timeseries(timeParams),
    client.usage.byProject(timeParams),
    client.usage.byModel(timeParams),
    client.usage.byProvider(timeParams),
    client.usage.byTeam(timeParams),
    client.providers.listAccounts(),
    client.providers.health(),
  ]);

  // Determine state
  if (summary.Requests === 0 && summary.TotalCostMicros === 0) {
    return <ZeroTrafficState />;
  }

  return (
    <TrafficState
      summary={summary}
      latency={latency}
      series={series}
      byProject={byProject}
      byModel={byModel}
      byProvider={byProvider}
      byTeam={byTeam}
      accounts={accounts}
      health={health}
      actions={
        <Suspense fallback={null}>
          <RangeTabs />
        </Suspense>
      }
    />
  );
}