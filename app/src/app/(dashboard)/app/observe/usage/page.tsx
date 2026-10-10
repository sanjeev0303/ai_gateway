import type { Metadata } from "next";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
import {
  DataTable,
  EmptyState,
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
  type Column,
} from "@/components/dashboard";
import { isDateRange, type DateRange } from "@/lib/date-range";
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

interface UsageRow {
  id: string;
  name: string;
  href: string;
  calls: number;
  tokens: number;
  shareFraction: number;
  cost: number;
}

const createColumns = (header: string): readonly Column<UsageRow>[] => [
  { key: "name", header, primary: "name", sort: "name", href: "href" },
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

const modelColumns = createColumns("Model");
const providerColumns = createColumns("Provider");
const projectColumns = createColumns("Project");
const teamColumns = createColumns("Team");
const userColumns = createColumns("User");

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
  const { usage, projects, teams } = await getApiClient();
  const to = new Date();
  const query = { from: new Date(to.getTime() - RANGE_MS[range]), to, ...filters };

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
    usage.summary(query),
    usage.timeseries(query),
    usage.byModel(query),
    usage.byProvider(query),
    usage.byProject(query),
    usage.byUser(query),
    usage.byTeam(query),
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
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

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
  } = await loadUsage(range, filters);

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

  const modelRows: UsageRow[] = byModel.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByModel > 0 ? cost / totalByModel : 0;
    return {
      id: row.Model,
      name: row.Model,
      href: `/app/observe/usage?model=${encodeURIComponent(row.Model)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      shareFraction,
      cost,
    };
  });

  const totalByProvider = byProvider.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);
  const providerRows: UsageRow[] = byProvider.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByProvider > 0 ? cost / totalByProvider : 0;
    return {
      id: row.Provider,
      name: row.Provider,
      href: `/app/observe/usage?provider=${encodeURIComponent(row.Provider)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      shareFraction,
      cost,
    };
  });

  const totalByProject = byProject.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);
  const projectRows: UsageRow[] = byProject.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByProject > 0 ? cost / totalByProject : 0;
    return {
      id: row.ProjectID,
      name: projectNames.get(row.ProjectID) ?? row.ProjectID,
      href: `/app/observe/projects/${encodeURIComponent(row.ProjectID)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      shareFraction,
      cost,
    };
  });

  const totalByTeam = byTeam.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);
  const teamRows: UsageRow[] = byTeam.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByTeam > 0 ? cost / totalByTeam : 0;
    return {
      id: row.TeamID,
      name: teamNames.get(row.TeamID) ?? row.TeamID,
      href: `/app/observe/teams/${encodeURIComponent(row.TeamID)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      shareFraction,
      cost,
    };
  });

  const totalByUser = byUser.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);
  const userRows: UsageRow[] = byUser.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    const shareFraction = totalByUser > 0 ? cost / totalByUser : 0;
    return {
      id: row.UserID,
      name: row.UserID,
      href: `/app/observe/users/${encodeURIComponent(row.UserID)}`,
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
        actions={<RangeTabs />}
      >
        <ObserveFilters />
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
                data={[...modelRows]
                  .sort((a, b) => b.cost - a.cost)
                  .map((row) => ({ label: row.name, value: row.cost }))}
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
              rows={modelRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No model usage recorded."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Per-provider detail</CardTitle>
            <CardDescription>Requests and spend by provider for the trailing window.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={providerColumns}
              rows={providerRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No provider usage recorded."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Per-project detail</CardTitle>
            <CardDescription>Requests and spend by project for the trailing window.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={projectColumns}
              rows={projectRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No project usage recorded."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Per-team detail</CardTitle>
            <CardDescription>Requests and spend by team for the trailing window.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={teamColumns}
              rows={teamRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No team usage recorded."
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Per-user detail</CardTitle>
            <CardDescription>Requests and spend by user for the trailing window.</CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={userColumns}
              rows={userRows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No user usage recorded."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
