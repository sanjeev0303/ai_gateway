"use client";

import * as React from "react";
import Link from "next/link";
import {
  Activity,
  Ban,
  CircleDollarSign,
  Clock,
  TriangleAlert,
} from "lucide-react";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
import type { AreaSeries } from "@/components/charts";
import {
  DataTable,
  FilterBar,
  PageShell,
  StatCard,
  type Column,
} from "@/components/dashboard";
import { RangeTabs } from "@/components/dashboard/range-tabs";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useDemoStore } from "@/lib/demo/store";
import type { SeriesPoint } from "@/lib/demo/types";
import {
  formatCompact,
  formatMs,
  formatPercent,
  formatUsd,
} from "@/lib/format";
import type { StatusTone, TimeSeriesPoint, TrendDirection } from "@/types";

type SparkField = "cost" | "requests" | "errors";

function fieldValue(point: SeriesPoint, field: SparkField): number {
  return point[field];
}

/** Compare the recent half of a series against the older half. */
function periodChange(
  points: readonly SeriesPoint[],
  field: SparkField,
): { fraction: number; direction: TrendDirection } {
  if (points.length < 2) return { fraction: 0, direction: "flat" };
  const half = Math.floor(points.length / 2);
  let older = 0;
  let recent = 0;
  for (const [index, point] of points.entries()) {
    const value = fieldValue(point, field);
    if (index < half) older += value;
    else recent += value;
  }
  if (older === 0) return { fraction: 0, direction: "flat" };
  const fraction = (recent - older) / older;
  return {
    fraction,
    direction: fraction > 0.005 ? "up" : fraction < -0.005 ? "down" : "flat",
  };
}

function toSparkline(points: readonly SeriesPoint[], field: SparkField): TimeSeriesPoint[] {
  return points.map((point) => ({ date: point.label, value: fieldValue(point, field) }));
}

const RANGE_HINT: Record<string, string> = {
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 90 days",
};

function errorTone(rate: number): StatusTone {
  if (rate >= 0.05) return "danger";
  if (rate >= 0.02) return "warning";
  return "success";
}

interface ProjectRow {
  name: string;
  detail: string;
  requests: number;
  cost: number;
  sharePercent: number;
  errorLabel: string;
  errorTone: StatusTone;
}

export function OverviewContent() {
  const { range, series, totals, breakdown } = useDemoStore();
  const points = series.points;

  const spendSeries: AreaSeries[] = [
    {
      label: "Spend",
      color: chartColor(0),
      data: toSparkline(points, "cost"),
    },
  ];

  const statusMix = [
    { label: "Success", value: Math.max(0, totals.requests - totals.errors - totals.blocked) },
    { label: "Error", value: totals.errors },
    { label: "Blocked", value: totals.blocked },
  ];

  const topModels = breakdown.byModel.slice(0, 6).map((rollup) => ({
    label: rollup.name,
    value: rollup.cost,
  }));

  const projectRows: ProjectRow[] = breakdown.byProject.slice(0, 8).map((rollup) => ({
    name: rollup.name,
    detail: rollup.detail,
    requests: rollup.requests,
    cost: rollup.cost,
    sharePercent: rollup.share * 100,
    errorLabel: formatPercent(rollup.errorRate, 2),
    errorTone: errorTone(rollup.errorRate),
  }));

  const columns: Column<ProjectRow>[] = [
    {
      key: "name",
      header: "Project",
      primary: "name",
      secondary: "detail",
      sort: "name",
    },
    {
      key: "requests",
      header: "Requests",
      primary: "requests",
      format: "compact",
      align: "right",
    },
    {
      key: "cost",
      header: "Spend",
      primary: "cost",
      format: "usd",
      align: "right",
      sort: "cost",
    },
    {
      key: "sharePercent",
      header: "Share",
      primary: "sharePercent",
      format: "percent",
      align: "right",
      sort: "sharePercent",
    },
    {
      key: "error",
      header: "Error rate",
      primary: "errorLabel",
      tone: "errorTone",
      align: "right",
    },
  ];

  const requestDelta = periodChange(points, "requests");
  const spendDelta = periodChange(points, "cost");
  const errorDelta = periodChange(points, "errors");

  return (
    <PageShell
      title="Overview"
      description="Workspace health, spend, and policy posture at a glance."
      actions={<RangeTabs />}
    >
      <div className="space-y-6">
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Spend"
            value={formatUsd(totals.cost)}
            hint={RANGE_HINT[range]}
            icon={CircleDollarSign}
            href="/app/billing/cost"
            delta={{ ...spendDelta, higherIsWorse: true, label: "vs prior half" }}
            sparkline={toSparkline(points, "cost")}
          />
          <StatCard
            label="Requests"
            value={formatCompact(totals.requests)}
            hint={`${formatCompact(totals.totalTokens)} tokens`}
            icon={Activity}
            href="/app/observe/traffic"
            delta={{ ...requestDelta, label: "vs prior half" }}
            sparkline={toSparkline(points, "requests")}
          />
          <StatCard
            label="Error rate"
            value={formatPercent(totals.errorRate, 2)}
            hint={`${formatCompact(totals.errors)} failed`}
            icon={TriangleAlert}
            tone={errorTone(totals.errorRate)}
            href="/app/observe/errors"
            delta={{ ...errorDelta, higherIsWorse: true, label: "vs prior half" }}
            sparkline={toSparkline(points, "errors")}
          />
          <StatCard
            label="Latency p95"
            value={formatMs(totals.p95LatencyMs)}
            hint={`p50 ${formatMs(totals.p50LatencyMs)}`}
            icon={Clock}
            href="/app/observe/latency"
          />
        </div>

        <div className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend over time</CardTitle>
              <CardDescription>
                {RANGE_HINT[range]} · {formatUsd(totals.cost)} total
              </CardDescription>
            </CardHeader>
            <CardContent>
              <AreaChart
                series={spendSeries}
                height={260}
                label="Spend over time"
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Request outcomes</CardTitle>
              <CardDescription>
                {formatCompact(totals.blocked)} blocked by policy
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <DonutChart
                data={statusMix}
                centerLabel="requests"
                formatValue={formatCompact}
              />
              <ul className="space-y-1.5 text-sm">
                {statusMix.map((entry) => (
                  <li key={entry.label} className="flex items-center justify-between gap-2">
                    <span className="text-muted-foreground">{entry.label}</span>
                    <span className="tabular-nums">{formatCompact(entry.value)}</span>
                  </li>
                ))}
              </ul>
              <Link
                href="/app/govern/policies"
                className="inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:underline"
              >
                <Ban className="size-4" aria-hidden />
                Review blocking policies
              </Link>
            </CardContent>
          </Card>
        </div>

        <div className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Projects by spend</CardTitle>
              <CardDescription>Where the {RANGE_HINT[range].toLowerCase()} went</CardDescription>
            </CardHeader>
            <CardContent>
              <DataTable
                columns={columns}
                rows={projectRows}
                rowKey="name"
                pageSize={8}
                emptyMessage="No projects matched these filters."
                toolbar={<FilterBar />}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Top models by spend</CardTitle>
              <CardDescription>Concentration of spend</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={topModels}
                label="Top models by spend"
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </PageShell>
  );
}
