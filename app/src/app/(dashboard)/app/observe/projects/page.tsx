import type { Metadata } from "next";

import { BarChart, DonutChart } from "@/components/charts";
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
  formatNumber,
  formatPercent,
  formatUsd,
  microsToUsd,
} from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Projects · Sentinel",
  description: "Every project routed through the gateway, and the spend each one accrues.",
};

const STATUS_TONE: Record<string, StatusTone> = {
  active: "success",
  archived: "neutral",
};

interface ProjectRow {
  id: string;
  name: string;
  slug: string;
  status: string;
  statusTone: StatusTone;
  requests: number;
  tokens: number;
  share: string;
  shareFraction: number;
  spend: number;
}

const projectColumns: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone", sort: "status" },
  {
    key: "requests",
    header: "Requests",
    primary: "requests",
    format: "compact",
    sort: "requests",
    align: "right",
  },
  {
    key: "tokens",
    header: "Tokens",
    primary: "tokens",
    format: "compact",
    sort: "tokens",
    align: "right",
  },
  {
    key: "share",
    header: "Share",
    primary: "share",
    meter: "shareFraction",
    meterMax: 1,
    sort: "shareFraction",
    align: "right",
  },
  {
    key: "spend",
    header: "Spend 30d",
    primary: "spend",
    format: "usd",
    align: "right",
    sort: "spend",
  },
];

export default async function ProjectsPage() {
  const { projects, usage } = await getApiClient();
  const [projectList, projectUsage] = await Promise.all([
    projects.list(),
    usage.byProject(),
  ]);

  const spendByProject = new Map(projectUsage.items.map((row) => [row.ProjectID, row]));

  const totalSpend = projectUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );
  const totalRequests = projectUsage.items.reduce((sum, row) => sum + row.Requests, 0);

  const rows: ProjectRow[] = projectList.items.map((project) => {
    const attributed = spendByProject.get(project.id);
    const spend = microsToUsd(attributed?.CostMicros ?? 0);
    const shareFraction = totalSpend > 0 ? spend / totalSpend : 0;
    return {
      id: project.id,
      name: project.name,
      slug: project.slug,
      status: project.status,
      statusTone: toneFor(STATUS_TONE, project.status),
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      share: formatPercent(shareFraction),
      shareFraction,
      spend,
    };
  });

  const active = rows.filter((row) => row.status === "active");
  const spendByStatus = [
    {
      label: "Active",
      value: active.reduce((sum, row) => sum + row.spend, 0),
    },
    {
      label: "Archived",
      value: rows
        .filter((row) => row.status === "archived")
        .reduce((sum, row) => sum + row.spend, 0),
    },
  ].filter((datum) => datum.value > 0);

  const spendByProjectChart = [...rows]
    .sort((a, b) => b.spend - a.spend)
    .map((project) => ({ label: project.name, value: project.spend }));

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Every project routed through the gateway, and how the bill divides between them."
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Active projects"
            value={formatNumber(active.length)}
            hint={`${formatNumber(rows.length)} total`}
          />
          <StatCard
            label="Requests"
            value={formatNumber(totalRequests)}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Spend 30d"
            value={formatUsd(totalSpend)}
            hint={`${formatUsd(totalSpend / Math.max(1, active.length))} per active project`}
          />
          <StatCard
            label="Avg per project"
            value={formatUsd(totalSpend / Math.max(1, rows.length))}
            hint="Trailing 30 days"
          />
        </section>

        <section className="grid gap-4 lg:grid-cols-3">
          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Trailing 30 days, highest first.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart data={spendByProjectChart} formatValue={formatUsd} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Spend by status</CardTitle>
              <CardDescription>
                Archived projects should stop accruing spend entirely.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart
                data={spendByStatus}
                centerLabel={formatCompact(totalSpend)}
                formatValue={formatUsd}
              />
            </CardContent>
          </Card>
        </section>

        <Card>
          <CardHeader>
            <CardTitle>Projects</CardTitle>
            <CardDescription>
              Each project&rsquo;s share of organization spend over the trailing 30 days.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={projectColumns}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No projects configured."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
