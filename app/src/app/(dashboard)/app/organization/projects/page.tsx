import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { BarChart, DonutChart } from "@/components/charts";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber, formatUsd, microsToUsd } from "@/lib/format";
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";

export const metadata: Metadata = {
  title: "Projects · Sentinel",
  description: "Workloads that map spend and model access onto a real system of record.",
};

const STATUS_TONE: Record<string, StatusTone> = {
  active: "success",
  archived: "neutral",
};

interface ProjectRow {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: string;
  statusTone: StatusTone;
  requests: number;
  tokens: number;
  spend: number;
}

const COLUMNS: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
  { key: "requests", header: "Requests", primary: "requests", format: "compact", align: "right", sort: "requests" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "spend", header: "Spend (30d)", primary: "spend", format: "usd", align: "right", sort: "spend" },
];

export default async function ProjectsPage() {
  const { projects, usage } = await getApiClient();
  const [projectList, projectUsage] = await Promise.all([
    projects.list(),
    usage.byProject(),
  ]);

  const spendByProject = new Map(projectUsage.items.map((row) => [row.ProjectID, row]));

  const rows: ProjectRow[] = projectList.items.map((project) => {
    const attributed = spendByProject.get(project.id);
    return {
      id: project.id,
      name: project.name,
      slug: project.slug,
      description: project.description,
      status: project.status,
      statusTone: toneFor(STATUS_TONE, project.status),
      requests: attributed?.Requests ?? 0,
      tokens: attributed?.TotalTokens ?? 0,
      spend: microsToUsd(attributed?.CostMicros ?? 0),
    };
  });

  const active = rows.filter((row) => row.status === "active");
  const archived = rows.filter((row) => row.status === "archived");
  const totalSpend = rows.reduce((sum, row) => sum + row.spend, 0);
  const topSpend = [...rows].sort((a, b) => b.spend - a.spend).slice(0, 8);

  const byStatus = [
    { label: "Active", value: active.length },
    { label: "Archived", value: archived.length },
  ].filter((datum) => datum.value > 0);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Each project is a workload scope and the unit that spend is attributed to."
        actions={
          <Button size="sm">
            <Plus aria-hidden /> New project
          </Button>
        }
      >
        <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <StatCard
            label="Projects"
            value={String(rows.length)}
            hint={`${active.length} active`}
          />
          <StatCard
            label="Archived"
            value={String(archived.length)}
            tone={archived.length > 0 ? "neutral" : "success"}
            hint="Retired from routing"
          />
          <StatCard
            label="Requests"
            value={formatNumber(rows.reduce((sum, row) => sum + row.requests, 0))}
            hint="Trailing 30 days"
          />
          <StatCard
            label="Spend"
            value={formatUsd(totalSpend)}
            hint="Trailing 30 days"
          />
        </section>

        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Spend by project</CardTitle>
              <CardDescription>Top eight projects by trailing 30-day spend.</CardDescription>
            </CardHeader>
            <CardContent>
              <BarChart
                data={topSpend.map((row) => ({ label: row.name, value: row.spend }))}
                formatValue={(value) => formatUsd(value)}
              />
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>By status</CardTitle>
              <CardDescription>How projects are distributed across lifecycle states.</CardDescription>
            </CardHeader>
            <CardContent>
              <DonutChart data={byStatus} centerLabel="projects" size={190} />
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>All projects</CardTitle>
            <CardDescription>
              Every request the gateway prices is attributed to one of these projects.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              pageSize={10}
              emptyMessage="No projects created yet."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
