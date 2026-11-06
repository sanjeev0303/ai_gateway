import type { Metadata } from "next";
import Link from "next/link";

import { LineChart, chartColor } from "@/components/charts";
import {
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
} from "@/components/dashboard";
import { isDateRange, type DateRange } from "@/lib/date-range";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatCompact, formatUsd, formatUsdExact, microsToUsd } from "@/lib/format";
import { getApiClient } from "@/lib/session";

export const metadata: Metadata = {
  title: "Costs · Tollseal",
  description: "Spend over time, against the prior period.",
};

const DAY = 24 * 60 * 60 * 1000;

const RANGE_MS: Record<DateRange, number> = {
  "24h": DAY,
  "7d": 7 * DAY,
  "30d": 30 * DAY,
  "90d": 90 * DAY,
};

const RANGE_HINT: Record<DateRange, string> = {
  "24h": "Last 24 hours",
  "7d": "Last 7 days",
  "30d": "Last 30 days",
  "90d": "Last 90 days",
};

async function loadCosts(
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

  const now = Date.now();
  const ms = RANGE_MS[range];
  const current = { from: new Date(now - ms), to: new Date(now), ...filters };
  const previous = { from: new Date(now - 2 * ms), to: new Date(now - ms), ...filters };

  const [summary, priorSummary, series] = await Promise.all([
    usage.summary(current),
    usage.summary(previous),
    usage.timeseries(current),
  ]);

  return { summary, priorSummary, series };
}

export default async function CostsPage({
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

  const { summary, priorSummary, series } = await loadCosts(range, filters);

  const spend = microsToUsd(summary.TotalCostMicros);
  const priorSpend = microsToUsd(priorSummary.TotalCostMicros);
  const changeFraction = priorSpend > 0 ? (spend - priorSpend) / priorSpend : 0;

  const spendSeries = series.items.map((point) => ({
    date: point.BucketStart.slice(0, 10),
    value: microsToUsd(point.CostMicros),
  }));

  const perMillionTokens =
    summary.TotalTokens > 0 ? (spend / summary.TotalTokens) * 1_000_000 : 0;
  const perRequest = summary.Requests > 0 ? spend / summary.Requests : 0;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Costs"
        description="Spend over time, against the prior period."
        actions={<RangeTabs />}
      >
        <ObserveFilters />
        <section className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
          <StatCard
            label="Spend"
            value={formatUsd(spend)}
            hint={RANGE_HINT[range]}
            delta={
              priorSpend > 0
                ? {
                    fraction: changeFraction,
                    direction:
                      changeFraction > 0.005
                        ? "up"
                        : changeFraction < -0.005
                          ? "down"
                          : "flat",
                    higherIsWorse: true,
                    label: "vs prior 30 days",
                  }
                : undefined
            }
            sparkline={spendSeries}
          />
          <StatCard
            label="Prior period"
            value={formatUsd(priorSpend)}
            hint="The equivalent period before that"
          />
          <StatCard
            label="Blended cost per Mtok"
            value={formatUsdExact(perMillionTokens)}
            hint="Across all traffic"
          />
          <StatCard
            label="Cost per request"
            value={formatUsdExact(perRequest)}
            hint={`${formatCompact(summary.TotalTokens)} tokens over ${summary.Requests} requests`}
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Daily spend</CardTitle>
            <CardDescription>Trailing 30 days, priced from the provider price sheet.</CardDescription>
          </CardHeader>
          <CardContent>
            <LineChart
              label="Daily spend"
              data={spendSeries}
              color={chartColor(0)}
              height={260}
              formatValue={formatUsd}
            />
          </CardContent>
        </Card>

        <p className="text-sm">
          <Link
            href="/app/observe/models"
            className="font-medium text-primary underline-offset-4 hover:underline"
          >
            See spend by model, provider, team and user
          </Link>
        </p>
      </PageShell>
    </div>
  );
}
