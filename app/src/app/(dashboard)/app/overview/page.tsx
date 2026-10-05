import type { Metadata } from "next";
import { Suspense } from "react";
import type { ComponentProps } from "react";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
import { DataTable } from "@/components/dashboard/data-table";
import type { Column } from "@/components/dashboard/data-table";
import { PageShell } from "@/components/dashboard/page-shell";
import { RangeTabs } from "@/components/dashboard/range-tabs";
import { isDateRange, type DateRange } from "@/lib/date-range";
import { StatCard } from "@/components/dashboard/stat-card";
import { EmptyState } from "@/components/dashboard/state";
import {
  formatCompact,
  formatMs,
  formatPercent,
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { CategoryDatum, StatusTone, TimeSeriesPoint, TrendDirection } from "@/types";

export const metadata: Metadata = {
  title: "Overview · Sentinel",
  description: "Requests, spend, error rate, and latency recorded for the selected window.",
};

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

export default async function OverviewPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const { usage } = await getApiClient();
  const to = new Date();
  const query = { from: new Date(to.getTime() - RANGE_MS[range]), to };

  const [summary, latency, series, byProject, byModel] = await Promise.all([
    usage.summary(query),
    usage.latency(query),
    usage.timeseries(query),
    usage.byProject(query),
    usage.byModel(query),
  ]);

  const hint = RANGE_HINT[range];
  const label = (bucketStart: string) =>
    range === "24h" ? bucketStart.slice(11, 16) : bucketStart.slice(0, 10);

  const totalSpend = microsToUsd(summary.TotalCostMicros);
  const errorRate = summary.Requests > 0 ? summary.Errors / summary.Requests : 0;

  const spendSeries: TimeSeriesPoint[] = series.items.map((point) => ({
    date: label(point.BucketStart),
    value: microsToUsd(point.CostMicros),
  }));
  const requestSeries: TimeSeriesPoint[] = series.items.map((point) => ({
    date: label(point.BucketStart),
    value: point.Requests,
  }));

  // Percentiles are absent, not zero, until a latency sample exists.
  const overall = latency.items.find((row) => !row.Provider && !row.Model);
  const p95 = overall?.P95TotalMs;
  const p50 = overall?.P50TotalMs;

  const outcomes: CategoryDatum[] = [
    { label: "Succeeded", value: Math.max(0, summary.Requests - summary.Errors) },
    { label: "Failed", value: summary.Errors },
  ];

  const topModels: CategoryDatum[] = [...byModel.items]
    .sort((a, b) => b.CostMicros - a.CostMicros)
    .slice(0, 6)
    .map((row) => ({ label: row.Model, value: microsToUsd(row.CostMicros) }));

  const rows: ProjectRow[] = byProject.items.map((row) => ({
    id: row.ProjectID,
    name: row.Name,
    requests: row.Requests,
    cost: microsToUsd(row.CostMicros),
    errorRate: row.Requests > 0 ? row.Errors / row.Requests : 0,
    share: totalSpend > 0 ? microsToUsd(row.CostMicros) / totalSpend : 0,
  }));

  const columns: readonly Column<ProjectRow>[] = [
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

  const spendTrend = periodChange(spendSeries);
  const requestTrend = periodChange(requestSeries);

  const stats: ComponentProps<typeof StatCard>[] = [
    {
      label: "Spend",
      value: formatUsd(totalSpend),
      hint,
      sparkline: spendSeries,
      ...(spendTrend ? { delta: { ...spendTrend, higherIsWorse: true } } : {}),
      href: "/app/observe/costs",
    },
    {
      label: "Requests",
      value: formatCompact(summary.Requests),
      hint: `${formatCompact(summary.TotalTokens)} tokens`,
      sparkline: requestSeries,
      ...(requestTrend ? { delta: { ...requestTrend, higherIsWorse: false } } : {}),
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
      value: p95 === undefined ? "n/a" : formatMs(p95),
      hint: p50 === undefined ? "No samples yet" : `p50 ${formatMs(p50)}`,
    },
  ];

  return (
    <PageShell
      title="Overview"
      description="Requests, spend, error rate, and latency recorded for the selected window."
      actions={
        <Suspense fallback={null}>
          <RangeTabs />
        </Suspense>
      }
    >
      {summary.Requests === 0 ? (
        <EmptyState
          title="No usage in this window"
          description={`Once requests are recorded, spend, errors, and latency for ${hint.toLowerCase()} will appear here.`}
        />
      ) : (
        <div className="space-y-6">
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {stats.map((stat) => (
              <StatCard key={stat.label} {...stat} />
            ))}
          </div>

          <div className="grid gap-6 xl:grid-cols-3">
            <section className="space-y-3 xl:col-span-2">
              <h2 className="text-sm font-medium">Spend over time</h2>
              <AreaChart
                series={[{ label: "Spend", color: chartColor(0), data: spendSeries }]}
                formatValue={formatUsd}
              />
            </section>

            <section className="space-y-3">
              <h2 className="text-sm font-medium">Request outcomes</h2>
              <DonutChart data={outcomes} centerLabel="requests" />
              <ul className="space-y-1.5 text-xs text-muted-foreground">
                {outcomes.map((outcome, index) => (
                  <li key={outcome.label} className="flex items-center justify-between gap-2">
                    <span className="flex items-center gap-2">
                      <span
                        className="size-2 rounded-full"
                        style={{ backgroundColor: chartColor(index) }}
                        aria-hidden
                      />
                      {outcome.label}
                    </span>
                    <span className="tabular-nums text-foreground">
                      {formatCompact(outcome.value)}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          </div>

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

          <DataTable
            columns={columns}
            rows={rows}
            rowKey="id"
            pageSize={8}
            emptyMessage="No project spend in this window."
          />
        </div>
      )}
    </PageShell>
  );
}