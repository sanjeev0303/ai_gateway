import type { Metadata } from "next";
import { Plus } from "lucide-react";

import { DataTable, PageShell, StatCard, type Column } from "@/components/dashboard";
import { Button } from "@ai-governance/console-ui/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { formatNumber } from "@ai-governance/console-ui/lib/format";
import { getApiClient } from "@/lib/session";
import { CreateTeamDialog } from "./teams-client";

export const metadata: Metadata = {
  title: "Teams · Tollseal",
  description: "Groups that own workloads and accrue spend.",
};

interface TeamRow {
  id: string;
  name: string;
  description: string;
  members: number;
  detailUrl: string;
}

const COLUMNS: readonly Column<TeamRow>[] = [
  { key: "name", header: "Team", primary: "name", secondary: "description", sort: "name", href: "detailUrl" },
  { key: "members", header: "Members", primary: "members", format: "number", align: "right", sort: "members" },
];

export default async function TeamsPage() {
  const { teams } = await getApiClient();
  const teamList = await teams.list();

  const memberCounts = new Map(
    await Promise.all(
      teamList.items.map(
        async (team) => [team.id, (await teams.listMembers(team.id)).items.length] as const,
      ),
    ),
  );

const rows: TeamRow[] = teamList.items.map((team) => {
    return {
      id: team.id,
      name: team.name,
      description: team.description,
      members: memberCounts.get(team.id) ?? 0,
      detailUrl: `/app/organization/teams/${team.id}`,
    };
  });

  const totalMembers = rows.reduce((sum, row) => sum + row.members, 0);

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Teams"
        description="Groups that own projects and are the unit of accountability for spend."
        actions={
          <CreateTeamDialog>
            <Button size="sm">
              <Plus aria-hidden /> New team
            </Button>
          </CreateTeamDialog>
        }
      >
        <section className="grid gap-3 sm:grid-cols-2 sm:gap-4">
          <StatCard label="Teams" value={String(rows.length)} hint="Across the organization" />
          <StatCard
            label="Seats filled"
            value={formatNumber(totalMembers)}
            hint="Team memberships held"
          />
        </section>

        <Card>
          <CardHeader>
            <CardTitle>All teams</CardTitle>
            <CardDescription>
              {formatNumber(totalMembers)} people are distributed across these groups.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <DataTable
              columns={COLUMNS}
              rows={rows}
              rowKey="id"
              emptyMessage="No teams created yet."
            />
          </CardContent>
        </Card>
      </PageShell>
    </div>
  );
}
