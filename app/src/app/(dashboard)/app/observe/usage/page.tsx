import type { Metadata } from "next";
import Link from "next/link";

import { AreaChart, chartColor } from "@/components/charts";
import {
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
} from "@/components/dashboard";
import { isDateRange, type DateRange } from "@ai-governance/console-ui/lib/date-range";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatCompact, formatNumber } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import type { TrendDirection } from "@/types";
import { ExportUsageButton } from "./export-button";

export const metadata: Metadata = {
  title: "Usage · Tollseal",
  description: "Request and token volume across every route.",
};

const DAY = 24 * 60 * 60 * 1000;

/** Returns undefined rather than a fabricated 0% when there is no prior window. */
function periodChange(series: readonly { value: number }[]):
  | { fraction: number; direction: TrendDirection }
  | undefined {
  const half = Math.floor(series.length / 2);
  if (half === 0) return undefined;
  const sum = (points: readonly { value: number }[]) =>
    points.reduce((acc, point) => acc + point.value, 0);
  const previous = sum(series.slice(0, half));
  const current = sum(series.slice(half));
  if (previous === 0) return undefined;
  const fraction = (current - previous) / previous;
  return {
    fraction,
    direction: fraction > 0.005 ? "up" : fraction < -0.005 ? "down" : "flat",
  };
}

const RANGE_MS: Record<DateRange, number> = {
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
  "30d": 30 * 24 * 60 * 60 * 1000,
  "90d": 90 * 24 * 60 * 60 * 1000,
};

async function loadUsage(
  range: DateRange,
  filters: {
    project_id?: string;
    user_id?: string;
    team_id?: string;
    provider?: string;
    model?: string;
    status?: string;
  },
) {
  const { usage } = await getApiClient();
  const to = new Date();
  const query = { from: new Date(to.getTime() - RANGE_MS[range]), to, ...filters };

  const [summary, series] = await Promise.all([
    usage.summary(query),
    usage.timeseries(query),
  ]);

  return { summary, series };
}

export default async function UsagePage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    service_account_id: typeof params.service_account_id === "string" ? params.service_account_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

  const { summary, series } = await loadUsage(range, filters);

  const points = series.items.map((point) => ({
    date: point.BucketStart.slice(0, 10),
    requests: point.Requests,
    tokens: point.TotalTokens,
  }));

  const requestSeries = points.map((p) => ({ date: p.date, value: p.requests }));
  const tokenSeries = points.map((p) => ({ date: p.date, value: p.tokens }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="Request and token volume across every route."
        actions={
          <div className="flex items-center gap-2">
            <RangeTabs />
            <ExportUsageButton summary={summary} />
          </div>
        }
      >
        <ObserveFilters />
        <section className="grid gap-3 sm:grid-cols-3 sm:gap-4">
          <StatCard
            label="Requests"
            value={formatNumber(summary.Requests)}
            hint="Trailing 30 days"
            delta={periodChange(requestSeries)}
            sparkline={requestSeries}
          />
          <StatCard
            label="Tokens"
            value={formatCompact(summary.TotalTokens)}
            hint={`${formatCompact(summary.InputTokens)} in / ${formatCompact(summary.OutputTokens)} out`}
            delta={periodChange(tokenSeries)}
            sparkline={tokenSeries}
          />
          <StatCard
            label="Cached tokens"
            value={formatCompact(summary.CachedTokens)}
            hint="Served from provider cache"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Token volume</CardTitle>
            <CardDescription>
              Total tokens per day. Input and output arrive as one figure at this
              granularity.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <AreaChart
              label="Daily tokens"
              series={[{ label: "Tokens", color: chartColor(0), data: tokenSeries }]}
              formatValue={formatCompact}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Requests per day</CardTitle>
            <CardDescription>Daily request volume across every route.</CardDescription>
          </CardHeader>
          <CardContent>
            <AreaChart
              label="Daily requests"
              series={[{ label: "Requests", color: chartColor(2), data: requestSeries }]}
              formatValue={formatCompact}
            />
          </CardContent>
        </Card>

        <p className="text-sm">
          <Link
            href="/app/observe/models"
            className="font-medium text-primary underline-offset-4 hover:underline"
          >
            See usage by model, provider, team and user
          </Link>
        </p>
      </PageShell>
    </div>
  );
}
