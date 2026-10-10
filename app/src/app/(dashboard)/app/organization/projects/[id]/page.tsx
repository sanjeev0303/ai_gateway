import { notFound } from "next/navigation";
import { getApiClient } from "@/lib/session";
import { PageShell } from "@/components/dashboard";
import { ProjectDetailClient } from "./project-detail-client";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Project Detail · Sentinel",
};

export default async function ProjectDetailPage({ params }: { params: { id: string } }) {
  const api = await getApiClient();
  
  try {
    const project = await api.projects.get(params.id);
    const members = await api.projects.listMembers(params.id);
    const orgMembers = await api.members.list();

    return (
      <div className="p-4 sm:p-6">
        <PageShell title={project.name} description={project.description || "No description provided."}>
          <ProjectDetailClient project={project} members={members.items} orgMembers={orgMembers.items} />
        </PageShell>
      </div>
    );
  } catch (err: any) {
    if (err.status === 404) notFound();
    throw err;
  }
}
