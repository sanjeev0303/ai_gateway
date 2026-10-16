import type { Metadata } from "next";

import { AreaChart, BarChart, DonutChart, chartColor } from "@/components/charts";
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
  formatDate,
  formatNumber,
  formatUsd,
  formatUsdExact,
  microsToUsd,
} from "@/lib/format";
import { getApiClient } from "@/lib/session";
import { ExportBillingUsageButton } from "./export-button";

export const metadata: Metadata = {
  title: "Usage · Sentinel",
  description: "Recorded metered spend for the trailing 30 days, with plan context.",
};

const DAY = 24 * 60 * 60 * 1000;

interface TeamRow {
  id: string;
  team: string;
  requests: number;
  tokens: number;
  cost: number;
  shareFraction: number;
}

const teamColumns: readonly Column<TeamRow>[] = [
  { key: "team", header: "Team", primary: "team", sort: "team" },
  {
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "compact",
    align: "right",
    sort: "requests",
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
    header: "Recorded cost",
    primary: "cost",
    format: "usd",
    align: "right",
    sort: "cost",
  },
];

async function loadBillingUsage() {
  const { billing, organization, usage, projects, teams } = await getApiClient();
  const window = { from: new Date(Date.now() - 30 * DAY) };

  const [org, summary, series, byProject, byTeam, projectList, teamList, invoiceList] =
    await Promise.all([
      organization.get(),
      usage.summary(window),
      usage.timeseries(window),
      usage.byProject(window),
      usage.byTeam(window),
      projects.list(),
      teams.list(),
      billing.listInvoices(),
    ]);

  return {
    organization: org,
    summary,
    series,
    byProject,
    byTeam,
    invoices: invoiceList.items,
    projectNames: new Map(projectList.items.map((project) => [project.id, project.name])),
    teamNames: new Map(teamList.items.map((team) => [team.id, team.name])),
  };
}

export default async function BillingUsagePage() {
  const {
    organization,
    summary,
    series,
    byProject,
    byTeam,
    invoices,
    projectNames,
    teamNames,
  } = await loadBillingUsage();

  const totalSpend = microsToUsd(summary.TotalCostMicros);
  const totalByTeam = byTeam.items.reduce((sum, row) => sum + microsToUsd(row.CostMicros), 0);
  const blendedPerMTok =
    summary.TotalTokens > 0 ? totalSpend / (summary.TotalTokens / 1_000_000) : 0;

  const spendSeries = series.items.map((point) => ({
    date: point.BucketStart.slice(0, 10),
    value: microsToUsd(point.CostMicros),
  }));

  const spendByProject = byProject.items
    .map((row) => ({
      label: projectNames.get(row.ProjectID) ?? row.ProjectID,
      value: microsToUsd(row.CostMicros),
    }))
    .sort((a, b) => b.value - a.value);

  const rows: TeamRow[] = byTeam.items.map((row) => {
    const cost = microsToUsd(row.CostMicros);
    return {
      id: row.TeamID,
      team: teamNames.get(row.TeamID) ?? row.TeamID,
      requests: row.Requests,
      tokens: row.TotalTokens,
      cost,
      shareFraction: totalByTeam > 0 ? cost / totalByTeam : 0,
    };
  });

  const spendByTeam = [...rows]
    .sort((a, b) => b.cost - a.cost)
    .map((row) => ({ label: row.team, value: row.cost }));

  const planInfo = organization.plan;
  const billingInfo = organization.billing;
  const periodParts = [
    billingInfo?.current_period_start
      ? formatDate(billingInfo.current_period_start)
      : null,
    billingInfo?.current_period_end ? formatDate(billingInfo.current_period_end) : null,
  ].filter((part): part is string => Boolean(part));

  const subscriptionLine = [
    planInfo ? `${planInfo.name} plan` : "No plan on record",
    billingInfo?.status ?? null,
    periodParts.length === 2 ? periodParts.join(" – ") : null,
    billingInfo?.cancel_at_period_end ? "cancels at period end" : null,
  ]
    .filter((part): part is string => Boolean(part))
    .join(" · ");

  const openInvoices = invoices.filter((invoice) => invoice.status === "open").length;

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Usage"
        description="Metered consumption recorded against this organization over the trailing 30 days."
        eyebrow={<p className="text-xs text-muted-foreground">{subscriptionLine}</p>}
        actions={<ExportBillingUsageButton summary={summary} />}
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Recorded cost"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Requests"
            value={formatNumber(summary.Requests)}
            hint="Metered completions"
          />
          <StatCard
            label="Tokens"
            value={formatCompact(summary.TotalTokens)}
            hint={`${formatCompact(summary.InputTokens)} in / ${formatCompact(summary.OutputTokens)} out`}
          />
          <StatCard
            label="Blended rate"
            value={`${formatUsdExact(blendedPerMTok)} / Mtok`}
            hint="Derived from recorded cost"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Allowance and invoicing</CardTitle>
            <CardDescription>
              What the control plane records, and what it does not.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-sm text-muted-foreground">
            <p>
              A plan carries a name, a code, and a status — no quota, no included
              allowance, and no overage rate is stored — so there is no allowance meter
              or overage projection to derive here. Treat the figures above as recorded
              consumption, not as an amount billed.
            </p>
            <p>
              {invoices.length === 0
                ? "No invoice has been recorded for this organization yet."
                : `${invoices.length} invoice${invoices.length === 1 ? "" : "s"} recorded, ${openInvoices} still open.`}{" "}
              An invoice is a record of priced usage with itemised lines, not a request
              for payment: no payment processor is integrated, so there is no card on
              file, no charge, and no paid state. Billing → Invoices has the detail.
            </p>
          </CardContent>
        </Card>

        <section className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>
                Trailing 30 days, highest first.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={spendByProject}
                centerLabel={formatUsd(totalSpend)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by team</CardTitle>
              <CardDescription>
                Where the metered cost is attributed.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendByTeam} formatValue={formatUsd} />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Daily recorded cost</CardTitle>
            <CardDescription>
              Derived cost per day, bucketed by the worker.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <AreaChart
              label="Daily cost"
              series={[{ label: "Cost", color: chartColor(3), data: spendSeries }]}
              formatValue={formatUsd}
            />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Per-team detail</CardTitle>
            <CardDescription>
              Requests, tokens, and recorded cost per team for the trailing window.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={teamColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No team-attributed usage recorded."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
