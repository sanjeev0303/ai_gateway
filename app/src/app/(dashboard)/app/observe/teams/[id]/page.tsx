import { Metadata } from "next";
import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { getApiClient } from "@/lib/session";
import { notFound } from "next/navigation";
import { formatCompact, formatUsd, formatDate } from "@/lib/format";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface UsageRow {
  id: string;
  name: string;
  href: string;
  calls: number;
  tokens: number;
  cost: number;
}

const createColumns = (header: string): readonly Column<UsageRow>[] => [
  { key: "name", header, primary: "name", sort: "name", href: "href" },
  { key: "calls", header: "Requests", primary: "calls", format: "compact", align: "right", sort: "calls" },
  { key: "tokens", header: "Tokens", primary: "tokens", format: "compact", align: "right", sort: "tokens" },
  { key: "cost", header: "Spend", primary: "cost", format: "usd", align: "right", sort: "cost" },
];

const modelColumns = createColumns("Model");
const projectColumns = createColumns("Project");

interface MemberRow {
  id: string;
  name: string;
  email: string;
  role: string;
  href: string;
}

const memberColumns: readonly Column<MemberRow>[] = [
  { key: "name", header: "Member", primary: "name", secondary: "email", sort: "name", href: "href" },
  { key: "role", header: "Role", primary: "role", sort: "role" },
];

export async function generateMetadata({ params }: { params: Promise<{ id: string }> }): Promise<Metadata> {
  const { id } = await params;
  return { title: `Team ${id} · Sentinel` };
}

export default async function TeamDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const { teams, usage, projects } = await getApiClient();

  try {
    const [
      team,
      usageSummary,
      membersList,
      byProject,
      byModel,
      projectList,
    ] = await Promise.all([
      teams.get(id),
      usage.summary({ team_id: id }),
      teams.listMembers(id),
      usage.byProject({ team_id: id }),
      usage.byModel({ team_id: id }),
      projects.list(),
    ]);

    const projectNames = new Map(projectList.items.map((p) => [p.id, p.name]));

    const memberRows: MemberRow[] = membersList.items.map((m) => ({
      id: m.user.id,
      name: m.user.display_name || m.user.email,
      email: m.user.email,
      role: m.role,
      href: `/app/observe/users/${encodeURIComponent(m.user.id)}`,
    }));

    const projectRows: UsageRow[] = byProject.items.map((row) => ({
      id: row.ProjectID,
      name: projectNames.get(row.ProjectID) ?? row.ProjectID,
      href: `/app/observe/projects/${encodeURIComponent(row.ProjectID)}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    const modelRows: UsageRow[] = byModel.items.map((row) => ({
      id: row.Model,
      name: row.Model,
      href: `/app/observe/usage?model=${encodeURIComponent(row.Model)}&team_id=${id}`,
      calls: row.Requests,
      tokens: row.TotalTokens,
      cost: (row.CostMicros ?? 0) / 1000000,
    }));

    return (
      <div className="p-4 sm:p-6 space-y-6">
        <PageShell title={team.name} description={`Team profile and usage.`}>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4 mb-6">
            <StatCard label="Spend 30d" value={formatUsd((usageSummary.TotalCostMicros ?? 0) / 1000000)} />
            <StatCard label="Requests 30d" value={formatCompact(usageSummary.Requests ?? 0)} />
            <StatCard label="Tokens 30d" value={formatCompact(usageSummary.TotalTokens ?? 0)} />
            <StatCard label="Errors 30d" value={formatCompact(usageSummary.Errors ?? 0)} />
          </div>

          <div className="grid gap-4 lg:grid-cols-2 mb-6">
            <Card>
              <CardHeader>
                <CardTitle>Members</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={memberColumns} rows={memberRows} rowKey="id" pageSize={5} emptyMessage="No members." />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Projects</CardTitle>
              </CardHeader>
              <CardContent>
                <DataTable columns={projectColumns} rows={projectRows} rowKey="id" pageSize={5} emptyMessage="No projects." />
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Models</CardTitle>
            </CardHeader>
            <CardContent>
              <DataTable columns={modelColumns} rows={modelRows} rowKey="id" pageSize={5} emptyMessage="No models." />
            </CardContent>
          </Card>
        </PageShell>
      </div>
    );
  } catch (err) {
    return notFound();
  }
}
