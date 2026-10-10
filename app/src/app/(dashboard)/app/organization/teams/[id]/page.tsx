import { notFound } from "next/navigation";
import { getApiClient } from "@/lib/session";
import { PageShell } from "@/components/dashboard";
import { TeamDetailClient } from "./team-detail-client";
import { Metadata } from "next";

export const metadata: Metadata = {
  title: "Team Detail · Sentinel",
};

export default async function TeamDetailPage({ params }: { params: { id: string } }) {
  const api = await getApiClient();
  
  try {
    const team = await api.teams.get(params.id);
    const members = await api.teams.listMembers(params.id);
    const orgMembers = await api.members.list();

    return (
      <div className="p-4 sm:p-6">
        <PageShell title={team.name} description={team.description || "No description provided."}>
          <TeamDetailClient team={team} members={members.items} orgMembers={orgMembers.items} />
        </PageShell>
      </div>
    );
  } catch (err: any) {
    if (err.status === 404) notFound();
    throw err;
  }
}
