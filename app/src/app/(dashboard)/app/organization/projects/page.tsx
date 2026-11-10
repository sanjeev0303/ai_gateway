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
import { toneFor } from "@/lib/org-tones";
import { getApiClient } from "@/lib/session";
import type { StatusTone } from "@/types";
import { CreateProjectDialog } from "./projects-client";

export const metadata: Metadata = {
  title: "Projects · Tollseal",
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
  detailUrl: string;
}

const COLUMNS: readonly Column<ProjectRow>[] = [
  { key: "name", header: "Project", primary: "name", secondary: "slug", sort: "name", href: "detailUrl" },
  { key: "status", header: "Status", primary: "status", tone: "statusTone" },
];

export default async function ProjectsPage() {
  const { projects } = await getApiClient();
  const projectList = await projects.list();

const rows: ProjectRow[] = projectList.items.map((project) => {
    return {
      id: project.id,
      name: project.name,
      slug: project.slug,
      description: project.description,
      status: project.status,
      statusTone: toneFor(STATUS_TONE, project.status),
      detailUrl: `/app/organization/projects/${project.id}`,
    };
  });

  const active = rows.filter((row) => row.status === "active");
  const archived = rows.filter((row) => row.status === "archived");

  return (
    <div className="p-4 sm:p-6">
      <PageShell
        title="Projects"
        description="Each project is a workload scope and the unit that spend is attributed to."
        actions={
          <CreateProjectDialog>
            <Button size="sm">
              <Plus aria-hidden /> New project
            </Button>
          </CreateProjectDialog>
        }
      >
        <section className="grid gap-3 sm:grid-cols-2 sm:gap-4">
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
        </section>

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
