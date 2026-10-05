import type { Metadata } from "next";

import { BarChart, DonutChart, LineChart, chartColor } from "@/components/charts";
import {
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
  title: "Costs · Sentinel",
  description: "Spend attributed to models, teams, projects, and providers.",
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

async function loadCosts(range: DateRange) {
  const { usage, teams, projects } = await getApiClient();

  const now = Date.now();
  const ms = RANGE_MS[range];
  const current = { from: new Date(now - ms), to: new Date(now) };
  const previous = { from: new Date(now - 2 * ms), to: new Date(now - ms) };

  const [
    summary,
    priorSummary,
    series,
    byTeam,
    byProject,
    byProvider,
    byModel,
    byUser,
    teamList,
    projectList,
  ] = await Promise.all([
    usage.summary(current),
    usage.summary(previous),
    usage.timeseries(current),
    usage.byTeam(current),
    usage.byProject(current),
    usage.byProvider(current),
    usage.byModel(current),
    usage.byUser(current),
    teams.list(),
    projects.list(),
  ]);

  return {
    summary,
    priorSummary,
    series,
    byTeam,
    byProject,
    byProvider,
    byModel,
    byUser,
    teamNames: new Map(teamList.items.map((team) => [team.id, team.name])),
    projectNames: new Map(projectList.items.map((project) => [project.id, project.name])),
  };
}

export default async function CostsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const raw = typeof params.range === "string" ? params.range : null;
  const range: DateRange = isDateRange(raw) ? raw : "30d";

  const {
    summary,
    priorSummary,
    series,
    byTeam,
    byProject,
    byProvider,
    byModel,
    byUser,
    teamNames,
    projectNames,
  } = await loadCosts(range);

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
        description="Every dollar attributed to a model, team, project, and provider."
        actions={<RangeTabs />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
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

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>Attributed to each team over the trailing window.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={byTeam.items.map((row) => ({
                  label: teamNames.get(row.TeamID) ?? row.TeamID,
                  value: microsToUsd(row.CostMicros),
                }))}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by provider</CardTitle>
              <CardDescription>Where the spend lands upstream.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={byProvider.items.map((row) => ({
                  label: row.Provider,
                  value: microsToUsd(row.CostMicros),
                }))}
                centerLabel={formatUsd(spend)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={byProject.items.map((row) => ({
                  label: projectNames.get(row.ProjectID) ?? row.ProjectID,
                  value: microsToUsd(row.CostMicros),
                }))}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by model</CardTitle>
              <CardDescription>Highest first.</CardDescription>
            </CardHeader>
            <CardContent>
                <BarChart
                  data={[...byModel.items]
                    .sort((a, b) => b.CostMicros - a.CostMicros)
                    .map((row) => ({
                      label: row.Model,
                      value: microsToUsd(row.CostMicros),
                    }))}
                  formatValue={formatUsd}
                />
              </CardContent>
            </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by user</CardTitle>
              <CardDescription>Highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              {byUser.items.length === 0 ? (
                <EmptyState
                  title="No per-user spend yet"
                  description="Once requests land in this window, spend will break down by user."
                />
              ) : (
                <BarChart
                  data={[...byUser.items]
                    .sort((a, b) => b.CostMicros - a.CostMicros)
                    .map((row) => ({
                      label: row.UserID,
                      value: microsToUsd(row.CostMicros),
                    }))}
                  formatValue={formatUsd}
                />
              )}
            </CardContent>
          </Card>
        </section>
      </PageShell>
    </div>
  );
}
