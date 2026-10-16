import type { Metadata } from "next";

import { BarChart, DonutChart, LineChart, chartColor } from "@/components/charts";
import {
  DataTable,
  ObserveFilters,
  PageShell,
  RangeTabs,
  StatCard,
  EmptyState,
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
  const { usage, teams, projects, members } = await getApiClient();

  const now = Date.now();
  const ms = RANGE_MS[range];
  const current = { from: new Date(now - ms), to: new Date(now), ...filters };
  const previous = { from: new Date(now - 2 * ms), to: new Date(now - ms), ...filters };

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
    membersList,
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
    members.list().catch(() => ({ items: [] })),
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
    userNames: new Map(membersList.items.map((m) => [m.user.id, m.user.display_name || m.user.email])),
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
    priorSummary,
    series,
    byTeam,
    byProject,
    byProvider,
    byModel,
    byUser,
    teamNames,
    projectNames,
    userNames,
  } = await loadCosts(range, filters);

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

  const totalByModel = byModel.items.reduce((acc, row) => acc + microsToUsd(row.CostMicros), 0);
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
      name: userNames.get(row.UserID) ?? row.UserID,
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
        title="Costs"
        description="Every dollar attributed to a model, team, project, and provider."
        actions={<RangeTabs />}
      >
        <ObserveFilters />
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
