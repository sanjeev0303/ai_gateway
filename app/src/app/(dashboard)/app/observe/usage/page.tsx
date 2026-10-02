import type { Metadata } from "next";

import { AreaChart, BarChart, DonutChart, LineChart, chartColor } from "@/components/charts";
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
import {
  formatCompact,
  formatMs,
  formatNumber,
  formatUsd,
} from "@/lib/format";
import { usageBreakdown, usageSeries } from "@/lib/mock-data";
import type { TrendDirection } from "@/types";

export const metadata: Metadata = {
  title: "Usage · Sentinel",
  description: "Token, request, and latency volume across every route.",
};

function periodChange(
  series: readonly { value: number }[],
): { fraction: number; direction: TrendDirection } {
  const half = Math.floor(series.length / 2);
  if (half === 0) return { fraction: 0, direction: "flat" };
  const sum = (points: readonly { value: number }[]) =>
    points.reduce((acc, point) => acc + point.value, 0);
  const previous = sum(series.slice(0, half));
  const current = sum(series.slice(half));
  if (previous === 0) return { fraction: 0, direction: "flat" };
  const fraction = (current - previous) / previous;
  return {
    fraction,
    direction: fraction > 0.005 ? "up" : fraction < -0.005 ? "down" : "flat",
  };
}

interface ModelRow {
  id: string;
  model: string;
  calls: number;
  cost: number;
  share: number;
}

const modelColumns: readonly Column<ModelRow>[] = [
  { key: "model", header: "Model", primary: "model", sort: "model" },
  {
    key: "calls",
    header: "Requests",
    primary: "calls",
    format: "compact",
    align: "right",
    sort: "calls",
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
    key: "share",
    header: "Share",
    primary: "share",
    meter: "cost",
    format: "percent",
    align: "right",
    sort: "share",
  },
];

const totalSpend = usageBreakdown.byModel.reduce(
  (acc, entry) => acc + entry.cost,
  0,
);

const byModel: ModelRow[] = usageBreakdown.byModel.map((entry) => ({
  id: entry.modelId,
  model: entry.label,
  calls: entry.calls,
  cost: entry.cost,
  share: totalSpend === 0 ? 0 : entry.cost / totalSpend,
}));

export default function UsagePage() {
  const callChange = periodChange(usageSeries.calls);
  const tokenChange = periodChange(usageSeries.tokens);
  const { totals } = usageSeries;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="Request, token, and latency volume across every route, with the trailing 30-day window."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Requests"
            value={formatNumber(totals.totalCalls)}
            hint="Trailing 30 days"
            delta={callChange}
            sparkline={usageSeries.calls}
          />
          <StatCard
            label="Tokens"
            value={formatCompact(totals.totalTokens)}
            hint={`${formatCompact(totals.inputTokens)} in / ${formatCompact(totals.outputTokens)} out`}
            delta={tokenChange}
            sparkline={usageSeries.tokens}
          />
          <StatCard
            label="Spend"
            value={formatUsd(totals.cost)}
            hint="Blended across all traffic"
            sparkline={usageSeries.cost}
          />
          <StatCard
            label="Latency"
            value={formatMs(totals.p95LatencyMs)}
            hint={`p50 ${formatMs(totals.p50LatencyMs)}`}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Token volume</CardTitle>
            <CardDescription>
              Input and output tokens per day. Output grows when prompt
              engineering works.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <AreaChart
              label="Daily tokens by direction"
              series={[
                { label: "Input", color: chartColor(0), data: usageSeries.inputTokens },
                {
                  label: "Output",
                  color: chartColor(1),
                  data: usageSeries.outputTokens,
                },
              ]}
              formatValue={formatCompact}
            />
          </CardContent>
        </Card>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Requests and latency</CardTitle>
              <CardDescription>
                Daily request volume against p95 latency.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <LineChart
                label="Daily requests"
                data={usageSeries.calls}
                color={chartColor(2)}
                formatValue={formatCompact}
              />
              <LineChart
                label="p95 latency trend"
                data={usageSeries.calls.map((point, index) => ({
                  date: point.date,
                  value: totals.p95LatencyMs + ((index * 37) % 120) - 40,
                }))}
                color={chartColor(3)}
                formatValue={formatMs}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>By provider</CardTitle>
              <CardDescription>Spend share per upstream.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={usageBreakdown.byProvider}
                centerLabel={formatCompact(totals.totalTokens)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by model</CardTitle>
              <CardDescription>Highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={usageBreakdown.byModel} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Where the volume is landing.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={usageBreakdown.byProject} formatValue={formatUsd} />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Per-model detail</CardTitle>
            <CardDescription>
              Requests and spend by model for the trailing window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={modelColumns}
              rows={byModel}
              rowKey="id"
              pageSize={10}
              emptyMessage="No model usage recorded."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
