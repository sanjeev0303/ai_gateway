import type { Metadata } from "next";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
import {
  DataTable,
  EmptyState,
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
  formatNumber,
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { getApiClient } from "@/lib/session";
import type { TrendDirection } from "@/types";

export const metadata: Metadata = {
  title: "Usage · Sentinel",
  description: "Request and token volume across every route, with spend attribution.",
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

interface ModelRow {
  id: string;
  model: string;
  calls: number;
  tokens: number;
  shareFraction: number;
  cost: number;
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
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    align: "right",
    sort: "tokens",
  },
  {
    key: "share",
    header: "Share",
    primary: "shareFraction",
    meter: "shareFraction",
    meterMax: 1,
    format: "percent",
    align: "right",
    sort: "shareFraction",
  },
  {
    key: "cost",
    header: "Spend",
    primary: "cost",
    format: "usd",
    align: "right",
    sort: "cost",
  },
];

async function loadUsage() {
  const { usage, projects, teams } = await getApiClient();
  const window = { from: new Date(Date.now() - 30 * DAY) };

  const [
    summary,
    series,
    byModel,
    byProvider,
    byProject,
    byUser,
    byTeam,
    projectList,
    teamList,
  ] = await Promise.all([
    usage.summary(window),
    usage.timeseries(window),
    usage.byModel(window),
    usage.byProvider(window),
    usage.byProject(window),
    usage.byUser(window),
    usage.byTeam(window),
    projects.list(),
    teams.list(),
  ]);

  return {
    summary,
    series,
    byModel,
    byProvider,
    byProject,
    byUser,
    byTeam,
    projectNames: new Map(projectList.items.map((project) => [project.id, project.name])),
    teamNames: new Map(teamList.items.map((team) => [team.id, team.name])),
  };
}

export default async function UsagePage() {
  const {
    summary,
    series,
    byModel,
    byProvider,
    byProject,
    byUser,
    byTeam,
    projectNames,
    teamNames,
  } = await loadUsage();

  const totalSpend = microsToUsd(summary.TotalCostMicros);
  const totalByModel = byModel.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);

  const points = series.items.map((point) => ({
    date: point.BucketStart.slice(0, 10),
    requests: point.Requests,
    tokens: point.TotalTokens,
    spend: microsToUsd(point.CostMicros),
  }));

  const requestSeries = points.map((p) => ({ date: p.date, value: p.requests }));
  const tokenSeries = points.map((p) => ({ date: p.date, value: p.tokens }));
  const spendSeries = points.map((p) => ({ date: p.date, value: p.spend }));

  const rows: ModelRow[] = byModel.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByModel > 0 ? cost / totalByModel : 0;
    return {
      id: row.Model,
      model: row.Model,
      calls: row.Requests,
      tokens: row.TotalTokens,
      shareFraction,
      cost,
    };
  });

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="Request and token volume across every route, with spend attribution."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
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
            label="Spend"
            value={formatUsd(totalSpend)}
            hint="Blended across all traffic"
            sparkline={spendSeries}
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

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Requests and spend</CardTitle>
              <CardDescription>Daily request volume against daily spend.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <AreaChart
                label="Daily requests"
                series={[{ label: "Requests", color: chartColor(2), data: requestSeries }]}
                formatValue={formatCompact}
              />
              <AreaChart
                label="Daily spend"
                series={[{ label: "Spend", color: chartColor(3), data: spendSeries }]}
                formatValue={formatUsd}
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
                data={byProvider.items.map((row) => ({
                  label: row.Provider,
                  value: microsToUsd(row.CostMicros),
                }))}
                centerLabel={formatUsd(totalSpend)}
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
              <BarChart
                data={[...rows]
                  .sort((a, b) => b.cost - a.cost)
                  .map((row) => ({ label: row.model, value: row.cost }))}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Where the volume is landing.</CardDescription>
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
        </section>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by user</CardTitle>
              <CardDescription>Who is driving the bill.</CardDescription>
            </CardHeader>
            <CardContent>
              {byUser.items.length === 0 ? (
                <EmptyState
                  title="No per-user usage yet"
                  description="Once requests land in this window, spend will break down by user."
                />
              ) : (
                <BarChart
                  data={[...byUser.items]
                    .sort((a, b) => microsToUsd(b.CostMicros) - microsToUsd(a.CostMicros))
                    .map((row) => ({
                      label: row.UserID,
                      value: microsToUsd(row.CostMicros),
                    }))}
                  formatValue={formatUsd}
                />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>Which budget absorbs it.</CardDescription>
            </CardHeader>
            <CardContent>
              {byTeam.items.length === 0 ? (
                <EmptyState
                  title="No per-team usage yet"
                  description="Once requests land in this window, spend will break down by team."
                />
              ) : (
                <BarChart
                  data={[...byTeam.items]
                    .sort((a, b) => microsToUsd(b.CostMicros) - microsToUsd(a.CostMicros))
                    .map((row) => ({
                      label: teamNames.get(row.TeamID) ?? row.TeamID,
                      value: microsToUsd(row.CostMicros),
                    }))}
                  formatValue={formatUsd}
                />
              )}
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Per-model detail</CardTitle>
            <CardDescription>Requests and spend by model for the trailing window.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={modelColumns}
              rows={rows}
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
