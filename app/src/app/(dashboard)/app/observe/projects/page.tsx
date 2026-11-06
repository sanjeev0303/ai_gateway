import type { Metadata } from "next";

import { DonutChart } from "@/components/charts";
import {
  DataTable,
  ObserveFilters,
  PageShell,
  StatCard,
  type Column,
  BreakdownTabs,
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
  title: "Projects · Tollseal",
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
  detailUrl: string;
}

const projectColumns: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name", href: "detailUrl" },
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

export default async function ProjectsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const filters = {
    project_id: typeof params.project_id === "string" ? params.project_id : undefined,
    user_id: typeof params.user_id === "string" ? params.user_id : undefined,
    team_id: typeof params.team_id === "string" ? params.team_id : undefined,
    service_account_id: typeof params.service_account_id === "string" ? params.service_account_id : undefined,
    provider: typeof params.provider === "string" ? params.provider : undefined,
    model: typeof params.model === "string" ? params.model : undefined,
    status: typeof params.status === "string" ? params.status : undefined,
  };

  const { projects, usage } = await getApiClient();
  const [projectList, projectUsage] = await Promise.all([
    projects.list(),
    usage.byProject(filters),
  ]);

  const spendByProject = new Map(projectUsage.items.map((row) => [row.ProjectID, row]));

  const totalSpend = projectUsage.items.reduce(
    (sum, row) => sum + microsToUsd(row.CostMicros),
    0,
  );

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
      detailUrl: `/app/organization/projects/${project.id}`,
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

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Every project routed through the gateway, and how the bill divides between them."
      >
        <BreakdownTabs />
        <ObserveFilters />
        <section className="grid gap-4 sm:grid-cols-2">
          <StatCard
            label="Active projects"
            value={formatNumber(active.length)}
            hint={`${formatNumber(rows.length)} total`}
          />
          <StatCard
            label="Avg per project"
            value={formatUsd(totalSpend / Math.max(1, rows.length))}
            hint="Trailing 30 days"
          />
        </section>

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
              centerLabel="spend"
              formatValue={formatUsd}
            />
          </CardContent>
        </Card>

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
